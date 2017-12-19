package streamer

import (
	"container/list"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/use-go/websocketStreamServer/events/eLiveListCtrl"
	"github.com/use-go/websocketStreamServer/events/eStreamerEvent"
	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

const (
	streamTypeSource = "streamSource"
	streamTypeSink   = "streamSink"
)

// StreamerService for Datatranformation
type StreamerService struct {
	parent         wssAPI.Obj
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

type StreamerConfig struct {
	Upstreams           []eLiveListCtrl.EveSetUpStreamApp `json:"upstreams"`
	UpstreamTimeoutSec  int                               `json:"upstreamsTimeoutSec"`
	MediaDataTimeoutSec int                               `json:"mediaDataTimeoutSec"`
}

var service *StreamerService
var serviceConfig StreamerConfig

func (streamerService *StreamerService) Init(msg *wssAPI.Msg) (err error) {
	streamerService.sources = make(map[string]*streamSource)
	streamerService.blacks = make(map[string]string)
	streamerService.whites = make(map[string]string)
	streamerService.upApps = list.New()
	service = streamerService
	streamerService.blackOn = false
	streamerService.whiteOn = false
	if msg != nil {
		fileName := msg.Param1.(string)
		err = streamerService.loadConfigFile(fileName)
	}
	if err != nil {
		streamerService.badIni()
	}
	// init the upstreamer
	for _, v := range serviceConfig.Upstreams {
		streamerService.InitUpstream(v)
	}
	return
}

func (streamerService *StreamerService) loadConfigFile(fileName string) (err error) {
	data, err := wssAPI.ReadFileAll(fileName)
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

func (streamerService *StreamerService) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (streamerService *StreamerService) Stop(msg *wssAPI.Msg) (err error) {
	return
}

func (streamerService *StreamerService) GetType() string {
	return wssAPI.OBJ_StreamerServer
}

func (streamerService *StreamerService) HandleTask(task wssAPI.Task) (err error) {

	if task == nil || task.Receiver() != streamerService.GetType() {
		logger.LOGE("bad stask")
		return errors.New("invalid task")
	}
	switch task.Type() {
	case eStreamerEvent.AddSource:
		taskAddsrc, ok := task.(*eStreamerEvent.EveAddSource)
		if false == ok {
			return errors.New("invalid param")
		}
		taskAddsrc.SrcObj, taskAddsrc.Id, err = streamerService.addsource(taskAddsrc.StreamName, taskAddsrc.Producer, taskAddsrc.RemoteIp)

		return
	case eStreamerEvent.GetSource:
		taskGetSrc, ok := task.(*eStreamerEvent.EveGetSource)
		if false == ok {
			return errors.New("invalid param")
		}
		streamerService.mutexSources.Lock()
		defer streamerService.mutexSources.Unlock()

		taskGetSrc.SrcObj, ok = streamerService.sources[taskGetSrc.StreamName]
		if false == ok {
			return errors.New("not found:" + taskGetSrc.StreamName)
		} else {
			taskGetSrc.HasProducer = streamerService.sources[taskGetSrc.StreamName].bProducer
		}
		logger.LOGT(taskGetSrc.StreamName)
		//id zero
		return
	case eStreamerEvent.DelSource:
		taskDelSrc, ok := task.(*eStreamerEvent.EveDelSource)
		if false == ok {
			return errors.New("invalid param")
		}
		taskDelSrc.StreamName = taskDelSrc.StreamName
		err = streamerService.delSource(taskDelSrc.StreamName, taskDelSrc.Id)
		return
	case eStreamerEvent.AddSink:
		taskAddSink, ok := task.(*eStreamerEvent.EveAddSink)
		if false == ok {
			return errors.New("invalid param")
		}
		err = streamerService.addSink(taskAddSink)
		return
	case eStreamerEvent.DelSink:
		taskDelSink, ok := task.(*eStreamerEvent.EveDelSink)
		if false == ok {
			return errors.New("invalid param")
		}
		err = streamerService.delSink(taskDelSink.StreamName, taskDelSink.SinkId)
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
			err = streamerService.addUpstream(taskSetUpStream)
		} else {
			err = streamerService.delUpstream(taskSetUpStream)
		}
		return
	default:
		return errors.New("invalid task type:" + task.Type())
	}
	return
}

func (streamerService *StreamerService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

//src control sink
//add source:not start src,start sinks
//del source:not stop src,stop sinks
func (streamerService *StreamerService) addsource(path string, producer wssAPI.Obj, addr net.Addr) (src wssAPI.Obj, id int64, err error) {

	if false == streamerService.checkStreamAddAble(path) {
		return nil, -1, errors.New("bad name")
	}
	streamerService.mutexSources.Lock()
	defer streamerService.mutexSources.Unlock()
	logger.LOGT("add source:" + path)
	oldSrc, exist := streamerService.sources[path]
	if exist == false {
		oldSrc = &streamSource{}
		msg := &wssAPI.Msg{}
		msg.Param1 = path
		oldSrc.Init(msg)
		oldSrc.SetProducer(true)
		streamerService.sources[path] = oldSrc
		src = oldSrc
		oldSrc.mutexId.Lock()
		oldSrc.createId++
		id = oldSrc.createId
		oldSrc.dataProducer = producer
		oldSrc.addr = addr
		oldSrc.mutexId.Unlock()
		return
	} else {
		if oldSrc.HasProducer() {
			err = errors.New("bad name")
			return
		} else {
			logger.LOGT("source:" + path + " is idle")
			oldSrc.SetProducer(true)
			src = oldSrc
			oldSrc.mutexId.Lock()
			oldSrc.createId++
			id = oldSrc.createId
			oldSrc.dataProducer = producer
			oldSrc.mutexId.Unlock()
			return
		}
	}
	return
}

func (streamerService *StreamerService) delSource(path string, id int64) (err error) {
	streamerService.mutexSources.Lock()
	defer streamerService.mutexSources.Unlock()
	logger.LOGT("del source:" + path)
	oldSrc, exist := streamerService.sources[path]

	if exist == false {
		return errors.New(path + " not found")
	} else {
		if id < oldSrc.createId {
			logger.LOGW("delete with id:" + strconv.Itoa(int(id)) + " failed")
			logger.LOGD(oldSrc.createId)
			return errors.New(path + "is old id:" + strconv.Itoa(int(id)) + " can not delete")
		}
		/*remove := */ oldSrc.SetProducer(false)
		//if remove == true {
		if 0 == len(oldSrc.sinks) {
			delete(streamerService.sources, path)
		}
		//}
		return
	}
	return
}

//add sink:auto start sink by src
//del sink:not stop sink,stop by sink itself
//将add sink 改成异步
func (streamerService *StreamerService) addSink(sinkInfo *eStreamerEvent.EveAddSink) (err error) {
	streamerService.mutexSources.Lock()
	defer streamerService.mutexSources.Unlock()
	path := sinkInfo.StreamName
	sinkId := sinkInfo.SinkId
	sinker := sinkInfo.Sinker
	sinkInfo.Added = false
	src, exist := streamerService.sources[path]
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
		go streamerService.pullStream(app, streamName, sinkId, sinkInfo.Sinker)
	} else {
		err = src.AddSink(sinkId, sinker)
		if err == nil {
			sinkInfo.Added = true
			msg := &wssAPI.Msg{}
			msg.Type = wssAPI.MSG_GetSource_NOTIFY
			sinker.ProcessMessage(msg)
		}
	}
	return
}

func (streamerService *StreamerService) delSink(path, sinkId string) (err error) {

	streamerService.mutexSources.Lock()
	defer streamerService.mutexSources.Unlock()
	src, exist := streamerService.sources[path]
	if false == exist {
		return errors.New("source not found in del sink")
	} else {
		logger.LOGD("delete sinker:" + path + " " + sinkId)
		src.mutexSink.Lock()
		defer src.mutexSink.Unlock()
		delete(src.sinks, sinkId)
		if 0 == len(src.sinks) && src.bProducer == false {
			delete(streamerService.sources, path)
		}
	}
	return
}
