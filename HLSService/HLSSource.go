package HLSService

import (
	"container/list"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/use-go/websocket-streamserver/events/eStreamerEvent"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/mediaTypes/ts"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type hlsTsData struct {
	buf        []byte
	durationMs float64
	idx        int
}

const TsCacheLength = 4
const (
	MasterM3U8 = "master.m3u8"
	videoPref  = "v"
	audioPref  = "a"
)

type HLSSource struct {
	sinkAdded    bool
	inSvrMap     bool
	chValid      bool
	chSvr        chan bool
	streamName   string
	urlPref      string
	clientID     string
	audioHeader  *flv.FlvTag
	videoHeader  *flv.FlvTag
	segIdx       int64
	tsCur        *ts.TsCreater
	audioCur     *audioCache
	tsCache      *list.List
	muxCache     sync.RWMutex
	beginTime    uint32
	waitsChannel *list.List
	muxWaits     sync.RWMutex
}

func (hlsSource *HLSSource) Init(msg *wssAPI.Msg) (err error) {
	hlsSource.sinkAdded = false
	hlsSource.inSvrMap = false
	hlsSource.chValid = false
	hlsSource.tsCache = list.New()
	hlsSource.waitsChannel = list.New()
	hlsSource.segIdx = 0
	hlsSource.beginTime = 0
	var ok bool
	hlsSource.streamName, ok = msg.Param1.(string)
	if false == ok {
		return errors.New("invalid param init hls source")
	}
	hlsSource.chSvr, ok = msg.Param2.(chan bool)
	if false == ok {
		return errors.New("invalid param init hls source")
	}
	hlsSource.chValid = true

	//create source
	hlsSource.clientID = wssAPI.GenerateGUID()
	taskAddSink := &eStreamerEvent.EveAddSink{
		StreamName: hlsSource.streamName,
		SinkId:     hlsSource.clientID,
		Sinker:     hlsSource}

	wssAPI.HandleTask(taskAddSink)

	logger.LOGD("init end")
	if strings.Contains(hlsSource.streamName, "/") {
		//hlsSource.urlPref="/"+serviceConfig.Route+"/"+hlsSource.streamName
		hlsSource.urlPref = "/" + serviceConfig.Route + "/" + hlsSource.streamName
	}
	return
}

func (hlsSource *HLSSource) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (hlsSource *HLSSource) Stop(msg *wssAPI.Msg) (err error) {
	defer func() {
		if err := recover(); err != nil {
			logger.LOGD(err)
		}
	}()
	//从源移除
	if hlsSource.sinkAdded {
		logger.LOGD("del sink")
		taskDelSink := &eStreamerEvent.EveDelSink{}
		taskDelSink.StreamName = hlsSource.streamName
		taskDelSink.SinkId = hlsSource.clientID
		go wssAPI.HandleTask(taskDelSink)
		hlsSource.sinkAdded = false
		logger.LOGT("del sinker:" + hlsSource.clientID)
	}
	//从service移除
	if hlsSource.inSvrMap {
		hlsSource.inSvrMap = false
		service.DelSource(hlsSource.streamName, hlsSource.clientID)
	}
	//清理数据
	if hlsSource.chValid {
		close(hlsSource.chSvr)
		hlsSource.chValid = false
	}

	hlsSource.muxWaits.Lock()
	defer hlsSource.muxWaits.Unlock()
	if hlsSource.waitsChannel != nil {
		for e := hlsSource.waitsChannel.Front(); e != nil; e = e.Next() {
			close(e.Value.(chan bool))
		}
		hlsSource.waitsChannel = list.New()
	}
	return
}

func (hlsSource *HLSSource) GetType() string {
	return ""
}

func (hlsSource *HLSSource) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (hlsSource *HLSSource) ProcessMessage(msg *wssAPI.Msg) (err error) {
	switch msg.Type {
	case wssAPI.MSG_GetSource_NOTIFY:
		//对端还没有监听 所以写不进去
		if hlsSource.chValid {
			hlsSource.chSvr <- true
			hlsSource.inSvrMap = true
			logger.LOGD("get source notify")
		}
		hlsSource.sinkAdded = true
		logger.LOGD("get source by notify")
	case wssAPI.MSG_GetSource_Failed:
		hlsSource.Stop(nil)
	case wssAPI.MSG_PLAY_START:
		hlsSource.sinkAdded = true
		logger.LOGD("get source by start play")
	case wssAPI.MSG_PLAY_STOP:
		//hls 停止就结束移除，不像RTMP等待
		hlsSource.Stop(nil)
	case wssAPI.MSG_FLV_TAG:
		tag := msg.Param1.(*flv.FlvTag)
		hlsSource.AddFlvTag(tag)
	default:
		logger.LOGT(msg.Type)
	}
	return
}

func (hlsSource *HLSSource) ServeHTTP(w http.ResponseWriter, req *http.Request, param string) {
	if strings.HasSuffix(param, ".ts") {
		//get ts file
		hlsSource.serveTs(w, req, param)
	} else {
		//get m3u8 file
		hlsSource.serveM3u8(w, req, param)
	}
}

func (hlsSource *HLSSource) serveTs(w http.ResponseWriter, req *http.Request, param string) {
	if strings.HasPrefix(param, videoPref) {
		hlsSource.serveVideo(w, req, param)
	} else {
		logger.LOGE("no audio now")
	}
}

func (hlsSource *HLSSource) serveM3u8(w http.ResponseWriter, req *http.Request, param string) {

	if MasterM3U8 == param {
		hlsSource.serveMaster(w, req, param)
		return
	}
}

func (hlsSource *HLSSource) createVideoM3U8(tsCacheCopy *list.List) (strOut string) {
	//max duration
	maxDuration := 0

	if tsCacheCopy.Len() == TsCacheLength {
		tsCacheCopy.Remove(tsCacheCopy.Front())
	}
	for e := tsCacheCopy.Front(); e != nil; e = e.Next() {
		dura := int(e.Value.(*hlsTsData).durationMs / 1000.0)
		if dura > maxDuration {
			maxDuration = dura
		}
	}
	if maxDuration == 0 {
		maxDuration = 1
	}
	//sequence
	sequence := tsCacheCopy.Front().Value.(*hlsTsData).idx

	strOut = "#EXTM3U\n"
	strOut += "#EXT-X-TARGETDURATION:" + strconv.Itoa(maxDuration) + "\n"
	strOut += "#EXT-X-VERSION:3\n"
	strOut += "#EXT-X-MEDIA-SEQUENCE:" + strconv.Itoa(sequence) + "\n"
	//strOut += "#EXT-X-PLAYLIST-TYPE:VOD\n"
	strOut += "#EXT-X-INDEPENDENT-SEGMENTS\n"
	//last two ts？？
	//if tsCacheCopy.Len()<=TsCacheLength{
	if true {
		for e := tsCacheCopy.Front(); e != nil; e = e.Next() {
			tmp := e.Value.(*hlsTsData)
			strOut += fmt.Sprintf("#EXTINF:%f,\n", tmp.durationMs/1000.0)
			//strOut += hlsSource.urlPref+"/"+strconv.Itoa(tmp.idx) + ".ts" + "\n"
			strOut += "v" + strconv.Itoa(tmp.idx) + ".ts" + "\n"
		}
	} else {
		e := tsCacheCopy.Front()
		for i := 0; i < tsCacheCopy.Len()-2; i++ {
			e = e.Next()
		}
		for ; e != nil; e = e.Next() {
			tmp := e.Value.(*hlsTsData)
			strOut += fmt.Sprintf("#EXTINF:%f,\n", tmp.durationMs/1000.0)
			//strOut += hlsSource.urlPref+"/"+strconv.Itoa(tmp.idx) + ".ts" + "\n"
			strOut += "v" + strconv.Itoa(tmp.idx) + ".ts" + "\n"
		}
	}
	//strOut += "#EXT-X-ENDLIST\n"
	return strOut
}

func (hlsSource *HLSSource) serveMaster(w http.ResponseWriter, req *http.Request, param string) {

	hlsSource.muxCache.RLock()
	tsCacheCopy := list.New()
	for e := hlsSource.tsCache.Front(); e != nil; e = e.Next() {
		tsCacheCopy.PushBack(e.Value)
	}
	hlsSource.muxCache.RUnlock()
	if tsCacheCopy.Len() > 0 {
		w.Header().Set("Content-Type", "Application/vnd.apple.mpegurl")
		strOut := hlsSource.createVideoM3U8(tsCacheCopy)
		w.Write([]byte(strOut))
	} else {
		//wait for new
		chWait := make(chan bool, 1)
		hlsSource.muxWaits.Lock()
		hlsSource.waitsChannel.PushBack(chWait)
		hlsSource.muxWaits.Unlock()
		select {
		case ret, ok := <-chWait:
			if false == ok || false == ret {
				w.WriteHeader(404)
				logger.LOGE("no data now")
				return
			} else {
				hlsSource.muxCache.RLock()
				tsCacheCopy := list.New()
				for e := hlsSource.tsCache.Front(); e != nil; e = e.Next() {
					tsCacheCopy.PushBack(e.Value)
				}
				hlsSource.muxCache.RUnlock()
				strOut := hlsSource.createVideoM3U8(tsCacheCopy)
				w.Header().Set("Content-Type", "Application/vnd.apple.mpegurl")
				w.Write([]byte(strOut))
			}
		case <-time.After(time.Minute):
			w.WriteHeader(404)
			logger.LOGE("time out")
			return
		}
	}
}

func (hlsSource *HLSSource) serveVideo(w http.ResponseWriter, req *http.Request, param string) {
	strIdx := strings.TrimPrefix(param, "v")
	strIdx = strings.TrimSuffix(strIdx, ".ts")
	idx, _ := strconv.Atoi(strIdx)
	hlsSource.muxCache.RLock()
	defer hlsSource.muxCache.RUnlock()
	for e := hlsSource.tsCache.Front(); e != nil; e = e.Next() {
		tsData := e.Value.(*hlsTsData)
		if tsData.idx == idx {
			w.Write(tsData.buf)
			return
		}
	}
}

func (hlsSource *HLSSource) AddFlvTag(tag *flv.FlvTag) {
	if hlsSource.audioHeader == nil && tag.TagType == flv.FLV_TAG_Audio {
		hlsSource.audioHeader = tag.Copy()
		return
	}
	if hlsSource.videoHeader == nil && tag.TagType == flv.FLV_TAG_Video {
		hlsSource.videoHeader = tag.Copy()
		return
	}

	//if idr,new slice
	if tag.TagType == flv.FLV_TAG_Video && tag.Data[0] == 0x17 && tag.Data[1] == 1 {
		hlsSource.createNewTSSegment(tag)
	} else {
		hlsSource.appendTag(tag)
	}
}

func (hlsSource *HLSSource) createNewTSSegment(keyframe *flv.FlvTag) {

	if hlsSource.tsCur == nil {
		hlsSource.tsCur = &ts.TsCreater{}
		if hlsSource.audioHeader != nil {
			hlsSource.tsCur.AddTag(hlsSource.audioHeader)

			hlsSource.audioCur = &audioCache{}
			hlsSource.audioCur.Init(hlsSource.audioHeader)
		}
		if hlsSource.videoHeader != nil {
			hlsSource.tsCur.AddTag(hlsSource.videoHeader)
		}
		hlsSource.tsCur.AddTag(keyframe)

	} else {
		//flush data
		if hlsSource.tsCur.GetDuration() < 10000 {
			hlsSource.appendTag(keyframe)
			return
		}
		data := hlsSource.tsCur.FlushTsList()
		hlsSource.muxCache.Lock()
		defer hlsSource.muxCache.Unlock()
		if hlsSource.tsCache.Len() > TsCacheLength {
			hlsSource.tsCache.Remove(hlsSource.tsCache.Front())
		}
		tsdata := &hlsTsData{}
		tsdata.durationMs = float64(hlsSource.tsCur.GetDuration())
		tsdata.buf = make([]byte, ts.TS_length*data.Len())
		ptr := 0
		for e := data.Front(); e != nil; e = e.Next() {
			copy(tsdata.buf[ptr:], e.Value.([]byte))
			ptr += ts.TS_length
		}
		tsdata.idx = int(hlsSource.segIdx & 0xffffffff)
		hlsSource.segIdx++
		hlsSource.tsCache.PushBack(tsdata)
		hlsSource.muxWaits.Lock()
		if hlsSource.waitsChannel.Len() > 0 {
			for e := hlsSource.waitsChannel.Front(); e != nil; e = e.Next() {
				e.Value.(chan bool) <- true
			}
			hlsSource.waitsChannel = list.New()
		}
		hlsSource.muxWaits.Unlock()

		if hlsSource.segIdx < 10 {
			//if true{
			wssAPI.CreateDirectory("audio")
			fileName := "audio/" + strconv.Itoa(int(hlsSource.segIdx-1)) + ".ts"
			logger.LOGD(tsdata.durationMs)
			fp, err := os.Create(fileName)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
			defer fp.Close()
			fp.Write(tsdata.buf)

			if hlsSource.audioHeader != nil {

				aacData := hlsSource.audioCur.Flush()
				fpaac, _ := os.Create("aac.aac")
				defer fpaac.Close()
				fpaac.Write(aacData)
				hlsSource.audioCur.Init(hlsSource.audioHeader)
			}
		}

		hlsSource.tsCur.Reset()
		//		hlsSource.tsCur = &ts.TsCreater{}
		//		if hlsSource.videoHeader != nil {
		//			hlsSource.tsCur.AddTag(hlsSource.videoHeader)
		//		}
		hlsSource.tsCur.AddTag(keyframe)

	}
}

func (hlsSource *HLSSource) appendTag(tag *flv.FlvTag) {
	if hlsSource.tsCur != nil {
		if hlsSource.beginTime == 0 && tag.Timestamp > 0 {
			hlsSource.beginTime = tag.Timestamp
		}
		tagIn := tag.Copy()
		//tagIn.Timestamp-=hlsSource.beginTime
		hlsSource.tsCur.AddTag(tagIn)
	}
	if flv.FLV_TAG_Audio == tag.TagType && hlsSource.audioCur != nil {
		hlsSource.audioCur.AddTag(tag)
	}
}
