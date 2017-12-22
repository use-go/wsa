package webSocketService

import (
	"container/list"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/mediaTypes/mp4"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

const (
	wsHandler = "websocketHandler"
)

type websocketHandler struct {
	parent       wssAPI.MsgHandler
	conn         *websocket.Conn
	app          string
	streamName   string
	playName     string
	pubName      string
	clientID     string
	isPlaying    bool
	mutexPlaying sync.RWMutex
	waitPlaying  *sync.WaitGroup
	stPlay       playInfo
	isPublish    bool
	mutexPublish sync.RWMutex
	hasSink      bool
	mutexbSink   sync.RWMutex
	hasSource    bool
	mutexbSource sync.RWMutex
	source       wssAPI.MsgHandler
	sourceIdx    int
	lastCmd      int
	mutexWs      sync.Mutex
}

type playInfo struct {
	cache          *list.List
	mutexCache     sync.RWMutex
	audioHeader    *flv.FlvTag
	videoHeader    *flv.FlvTag
	metadata       *flv.FlvTag
	keyFrameWrited bool
	beginTime      uint32
}

func (websockHandler *websocketHandler) Init(msg *wssAPI.Msg) (err error) {
	websockHandler.conn = msg.Param1.(*websocket.Conn)
	websockHandler.app = msg.Param2.(string)
	websockHandler.waitPlaying = new(sync.WaitGroup)
	websockHandler.lastCmd = WSCClose
	return
}

func (websockHandler *websocketHandler) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (websockHandler *websocketHandler) Stop(msg *wssAPI.Msg) (err error) {
	websockHandler.doClose()
	return
}

func (websockHandler *websocketHandler) GetType() string {
	return wsHandler
}

func (websockHandler *websocketHandler) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (websockHandler *websocketHandler) ProcessMessage(msg *wssAPI.Msg) (err error) {
	switch msg.Type {
	case wssAPI.MsgGetSourceNotify:
		websockHandler.hasSink = true
	case wssAPI.MsgGetSourceFailed:
		websockHandler.hasSink = false
		websockHandler.sendWsStatus(websockHandler.conn, WSStatusError, NetStreamPlayFailed, 0)
	case wssAPI.MsgFlvTag:
		tag := msg.Param1.(*flv.FlvTag)
		err = websockHandler.appendFlvTag(tag)
	case wssAPI.MsgPlayStart:
		websockHandler.startPlay()
	case wssAPI.MsgPlayStop:
		websockHandler.stopPlay()
		logger.LOGT("play stop message")
	case wssAPI.MsgPublishStart:
	case wssAPI.MsgPublishStop:
	}
	return
}

func (websockHandler *websocketHandler) processWSMessage(data []byte) (err error) {
	if nil == data || len(data) < 4 {
		websockHandler.Stop(nil)
		return
	}
	msgType := int(data[0])
	switch msgType {
	case WSPktAudio:
	case WSPktVideo:
	case WSPktControl:
		logger.LOGD("recv control data:")
		logger.LOGD(data)
		return websockHandler.controlMsg(data[1:])
	default:
		err = fmt.Errorf("msg type %d not supported", msgType)
		logger.LOGW("invalid binary data")
		return
	}
	return
}

func (websockHandler *websocketHandler) SetParent(parent wssAPI.MsgHandler) {
	websockHandler.parent = parent
}

func (websockHandler *playInfo) reset() {
	websockHandler.mutexCache.Lock()
	defer websockHandler.mutexCache.Unlock()
	websockHandler.cache = list.New()
	websockHandler.audioHeader = nil
	websockHandler.videoHeader = nil
	websockHandler.metadata = nil
	websockHandler.keyFrameWrited = false
	websockHandler.beginTime = 0
}

func (websockHandler *playInfo) addInitPkts() {
	websockHandler.mutexCache.Lock()
	defer websockHandler.mutexCache.Unlock()
	if websockHandler.audioHeader != nil {
		websockHandler.cache.PushBack(websockHandler.audioHeader)
	}
	if websockHandler.videoHeader != nil {
		websockHandler.cache.PushBack(websockHandler.videoHeader)
	}
	if websockHandler.metadata != nil {
		websockHandler.cache.PushBack(websockHandler.metadata)
	}
}

func (websockHandler *websocketHandler) startPlay() {
	websockHandler.stPlay.reset()
	websockHandler.isPlaying = true
	go websockHandler.threadPlay()
}

func (websockHandler *websocketHandler) threadPlay() {
	websockHandler.isPlaying = true
	websockHandler.waitPlaying.Add(1)
	defer func() {
		websockHandler.waitPlaying.Done()
		websockHandler.stPlay.reset()
	}()
	fmp4Creater := &mp4.FMP4Creater{}
	for true == websockHandler.isPlaying {
		websockHandler.stPlay.mutexCache.Lock()
		if websockHandler.stPlay.cache == nil || websockHandler.stPlay.cache.Len() == 0 {
			websockHandler.stPlay.mutexCache.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		tag := websockHandler.stPlay.cache.Front().Value.(*flv.FlvTag)
		websockHandler.stPlay.cache.Remove(websockHandler.stPlay.cache.Front())
		websockHandler.stPlay.mutexCache.Unlock()
		if WSCPause == websockHandler.lastCmd {
			continue
		}
		if tag.TagType == flv.FLV_TAG_ScriptData {
			err := websockHandler.sendWsControl(websockHandler.conn, WSCOnMetaData, tag.Data)
			if err != nil {
				logger.LOGE(err.Error())
				websockHandler.isPlaying = false
			}
			continue
		}
		slice := fmp4Creater.AddFlvTag(tag)
		if slice != nil {
			err := websockHandler.sendFmp4Slice(slice)
			if err != nil {
				logger.LOGE(err.Error())
				websockHandler.isPlaying = false
			}
		}
	}
}

func (websockHandler *websocketHandler) stopPlay() {
	websockHandler.isPlaying = false
	websockHandler.waitPlaying.Wait()
	websockHandler.stPlay.reset()
	websockHandler.sendWsStatus(websockHandler.conn, WSStatusStatus, NetStreamPlayStop, 0)
}

func (websockHandler *websocketHandler) stopPublish() {
	logger.LOGE("stop publish not code")
}

func (websockHandler *websocketHandler) sendWsControl(conn *websocket.Conn, ctrlType int, data []byte) (err error) {
	websockHandler.mutexWs.Lock()
	defer websockHandler.mutexWs.Unlock()
	dataSend := make([]byte, len(data)+4)
	dataSend[0] = WSPktControl
	dataSend[1] = byte((ctrlType >> 16) & 0xff)
	dataSend[2] = byte((ctrlType >> 8) & 0xff)
	dataSend[3] = byte((ctrlType >> 0) & 0xff)
	copy(dataSend[4:], data)
	return conn.WriteMessage(websocket.BinaryMessage, dataSend)
}

func (websockHandler *websocketHandler) sendWsStatus(conn *websocket.Conn, level, code string, req int) (err error) {
	websockHandler.mutexWs.Lock()
	defer websockHandler.mutexWs.Unlock()
	st := &stResult{Level: level, Code: code, Req: req}
	dataJSON, err := json.Marshal(st)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	dataSend := make([]byte, len(dataJSON)+4)
	dataSend[0] = WSPktControl
	dataSend[1] = 0
	dataSend[2] = 0
	dataSend[3] = 0
	copy(dataSend[4:], dataJSON)
	err = conn.WriteMessage(websocket.BinaryMessage, dataSend)
	return
}
