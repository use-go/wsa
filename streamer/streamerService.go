package streamer

import (
	"container/list"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/use-go/websocket-streamserver/events/eLiveListCtrl"
	"github.com/use-go/websocket-streamserver/events/eStreamerEvent"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssapi"
	"github.com/use-go/websocket-streamserver/utils"
)

const (
	streamTypeSource = "streamSource"
	streamTypeSink   = "streamSink"
)

// StreamerService for Datatranformation
type StreamerService struct {
	parent         wssapi.MsgHandler
	mutexSources   sync.RWMutex
	sources        map[string]*streamSource
	mutexBlackList sync.RWMutex
	blacks         map[string]string
	mutexWhiteList sync.RWMutex
	whites         map[string]string
	blackOn        bool
	whiteOn        bool
	mutexUpStream  sync.RWMutex
	upApps         *list.List
	upAppIdx       int
}

// StreamerConfig for StreamerService
type StreamerConfig struct {
	Upstreams           []eLiveListCtrl.EveSetUpStreamApp `json:"upstreams"`
	UpstreamTimeoutSec  int                               `json:"upstreamsTimeoutSec"`
	MediaDataTimeoutSec int                               `json:"mediaDataTimeoutSec"`
}

var service *StreamerService
var serviceConfig StreamerConfig

//Init Streamer
func (streamer *StreamerService) Init(msg *wssapi.Msg) (err error) {
	streamer.sources = make(map[string]*streamSource)
	streamer.blacks = make(map[string]string)
	streamer.whites = make(map[string]string)
	streamer.upApps = list.New()
	service = streamer
	streamer.blackOn = false
	streamer.whiteOn = false
	if msg != nil {
		fileName := msg.Param1.(string)
		err = streamer.loadConfigFile(fileName)
	}
	if err != nil {
		streamer.badIni()
	}
	// init the upstreamer
	for _, v := range serviceConfig.Upstreams {
		streamer.InitUpstream(v)
	}
	return
}

func (streamer *StreamerService) loadConfigFile(fileName string) (err error) {
	data, err := utils.ReadFileAll(fileName)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	return
}

//Start func
func (streamer *StreamerService) Start(msg *wssapi.Msg) (err error) {
	return
}

//Stop func
func (streamer *StreamerService) Stop(msg *wssapi.Msg) (err error) {
	return
}

//GetType : Get current Server Type to Identity which server it is
func (streamer *StreamerService) GetType() string {
	return wssapi.OBJStreamerServer
}

//HandleTask func
func (streamer *StreamerService) HandleTask(task wssapi.Task) (err error) {

	if task == nil || task.Receiver() != streamer.GetType() {
		logger.LOGE("bad stask")
		return errors.New("invalid task")
	}
	switch task.Type() {
	case eStreamerEvent.AddSource:
		taskAddsrc, ok := task.(*eStreamerEvent.EveAddSource)
		if false == ok {
			return errors.New("invalid param")
		}
		taskAddsrc.SrcObj, taskAddsrc.ID, err = streamer.addsource(taskAddsrc.StreamName, taskAddsrc.Producer, taskAddsrc.RemoteIp)

		return
	case eStreamerEvent.GetSource:
		taskGetSrc, ok := task.(*eStreamerEvent.EveGetSource)
		if false == ok {
			return errors.New("invalid param")
		}
		streamer.mutexSources.Lock()
		defer streamer.mutexSources.Unlock()

		taskGetSrc.SrcObj, ok = streamer.sources[taskGetSrc.StreamName]
		if false == ok {
			return errors.New("not found:" + taskGetSrc.StreamName)
		}
		taskGetSrc.HasProducer = streamer.sources[taskGetSrc.StreamName].bProducer

		logger.LOGT(taskGetSrc.StreamName)
		//id zero
		return
	case eStreamerEvent.DelSource:
		taskDelSrc, ok := task.(*eStreamerEvent.EveDelSource)
		if false == ok {
			return errors.New("invalid param")
		}
		//		taskDelSrc.StreamName = taskDelSrc.StreamName
		err = streamer.delSource(taskDelSrc.StreamName, taskDelSrc.ID)
		return
	case eStreamerEvent.AddSink:
		taskAddSink, ok := task.(*eStreamerEvent.EveAddSink)
		if false == ok {
			return errors.New("invalid param")
		}
		err = streamer.addSink(taskAddSink)
		return
	case eStreamerEvent.DelSink:
		taskDelSink, ok := task.(*eStreamerEvent.EveDelSink)
		if false == ok {
			return errors.New("invalid param")
		}
		err = streamer.delSink(taskDelSink.StreamName, taskDelSink.SinkId)
		return
	case eLiveListCtrl.EnableBlackList:
		taskEnableBlack, ok := task.(*eLiveListCtrl.EveEnableBlackList)
		if false == ok {
			return errors.New("invalid param")
		}
		err = enableBlackList(taskEnableBlack.Enable)
		return
	case eLiveListCtrl.EnableWhiteList:
		taskEnableWhite, ok := task.(*eLiveListCtrl.EveEnableWhiteList)
		if false == ok {
			return errors.New("invalid param")
		}
		err = enableWhiteList(taskEnableWhite.Enable)
	case eLiveListCtrl.SetBlackList:
		taskSetBlackList, ok := task.(*eLiveListCtrl.EveSetBlackList)
		if false == ok {
			return errors.New("invalid param")
		}
		if taskSetBlackList.Add == true {
			err = addBlackList(taskSetBlackList.Names)
		} else {
			err = delBlackList(taskSetBlackList.Names)
		}
		return
	case eLiveListCtrl.SetWhiteList:
		taskSetWhite, ok := task.(*eLiveListCtrl.EveSetWhiteList)
		if false == ok {
			return errors.New("invalid param")
		}
		if taskSetWhite.Add {
			err = addWhiteList(taskSetWhite.Names)
		} else {
			err = delWhiteList(taskSetWhite.Names)
		}
		return
	case eLiveListCtrl.GetLiveList:
		taskGetLiveList, ok := task.(*eLiveListCtrl.EveGetLiveList)
		if false == ok {
			return errors.New("invalid param")
		}
		taskGetLiveList.Lives, err = getLiveList()
		return
	case eLiveListCtrl.GetLivePlayerCount:
		taskGetLivePlayerCount, ok := task.(*eLiveListCtrl.EveGetLivePlayerCount)
		if false == ok {
			return errors.New("invalid param")
		}
		taskGetLivePlayerCount.Count, err = getPlayerCount(taskGetLivePlayerCount.LiveName)
		return
	case eLiveListCtrl.SetUpStreamApp:
		taskSetUpStream, ok := task.(*eLiveListCtrl.EveSetUpStreamApp)
		if false == ok {
			return errors.New("invalid param set upstream")
		}
		if taskSetUpStream.Add {
			err = streamer.addUpstream(taskSetUpStream)
		} else {
			err = streamer.delUpstream(taskSetUpStream)
		}
		return
	default:
		return errors.New("invalid task type:" + task.Type())
	}
	return
}

//ProcessMessage of streamer
func (streamer *StreamerService) ProcessMessage(msg *wssapi.Msg) (err error) {
	return
}

//src control sink
//add source:not start src,start sinks
//del source:not stop src,stop sinks
func (streamer *StreamerService) addsource(path string, producer wssapi.MsgHandler, addr net.Addr) (src wssapi.MsgHandler, id int64, err error) {

	if false == streamer.checkStreamAddAble(path) {
		return nil, -1, errors.New("bad name")
	}
	streamer.mutexSources.Lock()
	defer streamer.mutexSources.Unlock()
	logger.LOGT("add source:" + path)
	oldSrc, exist := streamer.sources[path]
	if exist == false {
		oldSrc = &streamSource{}
		msg := &wssapi.Msg{}
		msg.Param1 = path
		oldSrc.Init(msg)
		oldSrc.SetProducer(true)
		streamer.sources[path] = oldSrc
		src = oldSrc
		oldSrc.mutexID.Lock()
		oldSrc.createID++
		id = oldSrc.createID
		oldSrc.dataProducer = producer
		oldSrc.addr = addr
		oldSrc.mutexID.Unlock()
		return
	}
	if oldSrc.HasProducer() {
		err = errors.New("bad name")
		return
	}
	logger.LOGT("source:" + path + " is idle")
	oldSrc.SetProducer(true)
	src = oldSrc
	oldSrc.mutexID.Lock()
	oldSrc.createID++
	id = oldSrc.createID
	oldSrc.dataProducer = producer
	oldSrc.mutexID.Unlock()
	return

}

func (streamer *StreamerService) delSource(path string, id int64) (err error) {
	streamer.mutexSources.Lock()
	defer streamer.mutexSources.Unlock()
	logger.LOGT("del source:" + path)
	oldSrc, exist := streamer.sources[path]

	if exist == false {
		return errors.New(path + " not found")
	}
	if id < oldSrc.createID {
		logger.LOGW("delete with id:" + strconv.Itoa(int(id)) + " failed")
		logger.LOGD(oldSrc.createID)
		return errors.New(path + "is old id:" + strconv.Itoa(int(id)) + " can not delete")
	}
	/*remove := */ oldSrc.SetProducer(false)
	//if remove == true {
	if 0 == len(oldSrc.sinks) {
		delete(streamer.sources, path)
	}
	//}
	return

}

//add sink:auto start sink by src
//del sink:not stop sink,stop by sink itself
//将add sink 改成异步
func (streamer *StreamerService) addSink(sinkInfo *eStreamerEvent.EveAddSink) (err error) {
	streamer.mutexSources.Lock()
	defer streamer.mutexSources.Unlock()
	path := sinkInfo.StreamName
	sinkID := sinkInfo.SinkId
	sinker := sinkInfo.Sinker
	sinkInfo.Added = false
	src, exist := streamer.sources[path]
	hasProducer := false
	if nil != src {
		hasProducer = src.bProducer
	}
	if false == exist || false == hasProducer {
		tmpStrings := strings.Split(path, "/")
		if len(tmpStrings) < 2 {
			return errors.New("add sink bad path:" + path)
		}
		app := strings.TrimSuffix(path, tmpStrings[len(tmpStrings)-1])
		app = strings.TrimSuffix(app, "/")
		streamName := tmpStrings[len(tmpStrings)-1]
		logger.LOGT("create upstream:" + path)
		go streamer.pullStream(app, streamName, sinkID, sinkInfo.Sinker)
	} else {
		err = src.AddSink(sinkID, sinker)
		if err == nil {
			sinkInfo.Added = true
			msg := &wssapi.Msg{}
			msg.Type = wssapi.MsgGetSourceNotify
			sinker.ProcessMessage(msg)
		}
	}
	return
}

func (streamer *StreamerService) delSink(path, sinkID string) (err error) {

	streamer.mutexSources.Lock()
	defer streamer.mutexSources.Unlock()
	src, exist := streamer.sources[path]
	if false == exist {
		return errors.New("source not found in del sink")
	}
	logger.LOGD("delete sinker:" + path + " " + sinkID)
	src.mutexSink.Lock()
	defer src.mutexSink.Unlock()
	delete(src.sinks, sinkID)
	if 0 == len(src.sinks) && src.bProducer == false {
		delete(streamer.sources, path)
	}

	return
}
