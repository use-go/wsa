package streamer

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type streamSource struct {
	parent       wssAPI.MsgHandler
	addr         net.Addr
	bProducer    bool
	mutexSink    sync.RWMutex
	sinks        map[string]*streamSink
	streamName   string
	metadata     *flv.FlvTag
	audioHeader  *flv.FlvTag
	videoHeader  *flv.FlvTag
	lastKeyFrame *flv.FlvTag
	createId     int64
	mutexId      sync.RWMutex
	dataProducer wssAPI.MsgHandler
}

func (source *streamSource) Init(msg *wssAPI.Msg) (err error) {
	source.sinks = make(map[string]*streamSink)
	source.streamName = msg.Param1.(string)
	return
}

func (source *streamSource) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (source *streamSource) Stop(msg *wssAPI.Msg) (err error) {
	return
}

func (source *streamSource) GetType() string {
	return streamTypeSource
}

func (source *streamSource) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (source *streamSource) ProcessMessage(msg *wssAPI.Msg) (err error) {
	switch msg.Type {
	case wssAPI.MSG_FLV_TAG:
		if false == source.bProducer {
			return errors.New("src may closed or invalid")
		}
		tag := msg.Param1.(*flv.FlvTag)
		switch tag.TagType {
		case flv.FLV_TAG_Audio:
			if source.audioHeader == nil {
				source.audioHeader = tag.Copy()
				source.audioHeader.Timestamp = 0
			}
		case flv.FLV_TAG_Video:
			if source.videoHeader == nil {
				source.videoHeader = tag.Copy()
				source.videoHeader.Timestamp = 0
			}
			if (tag.Data[0] >> 4) == 1 {
				source.lastKeyFrame = tag.Copy()
			}

		case flv.FLV_TAG_ScriptData:
			if source.metadata == nil {
				source.metadata = tag.Copy()
			}
		}
		source.mutexSink.RLock()
		defer source.mutexSink.RUnlock()
		for k, v := range source.sinks {
			err = v.ProcessMessage(msg)
			if err != nil {
				logger.LOGE("send msg to sink failed,delete it:" + k)
				delete(source.sinks, k)
				v.Stop(nil)
				err = nil //这不是源的锅
			}
		}
		return
	default:
		logger.LOGW(fmt.Sprintf("msg type %d not processed", msg.Type))
	}
	return
}

func (source *streamSource) HasProducer() bool {
	return source.bProducer
}

func (source *streamSource) SetProducer(status bool) (remove bool) {
	if status == source.bProducer {
		return
	}
	source.bProducer = status
	if source.bProducer == false {
		//通知生产者
		logger.LOGD(source.dataProducer)
		if wssAPI.InterfaceValid(source.dataProducer) {
			logger.LOGD("force closed")
			msg := &wssAPI.Msg{Type: wssAPI.MSG_SourceClosed_Force}
			source.dataProducer.ProcessMessage(msg)
			source.dataProducer = nil
		}
		//clear cache
		source.clearCache()
		//notify sinks stop
		if 0 == len(source.sinks) {
			return true
		}
		source.mutexSink.RLock()
		defer source.mutexSink.RUnlock()
		for _, v := range source.sinks {
			v.Stop(nil)
		}
		return
	} else {
		//notify sinks start
		source.mutexSink.RLock()
		defer source.mutexSink.RUnlock()
		for _, v := range source.sinks {
			v.Start(nil)
		}
		return
	}
}

func (source *streamSource) AddSink(id string, sinker wssAPI.MsgHandler) (err error) {
	source.mutexSink.Lock()
	defer source.mutexSink.Unlock()
	logger.LOGT(source.streamName + " add sink:" + id)
	_, exist := source.sinks[id]
	if true == exist {
		return errors.New("sink " + id + " exist")
	}
	sink := &streamSink{}
	msg := &wssAPI.Msg{}
	msg.Param1 = id
	msg.Param2 = sinker
	err = sink.Init(msg)
	if err != nil {
		logger.LOGE("sink init failed")
		return
	}

	source.sinks[id] = sink
	if source.bProducer {
		err = sink.Start(nil)
		if source.metadata != nil {
			msg.Param1 = source.metadata
			msg.Type = wssAPI.MSG_FLV_TAG
			sink.ProcessMessage(msg)
		}
		if source.audioHeader != nil {
			msg.Param1 = source.audioHeader
			msg.Type = wssAPI.MSG_FLV_TAG
			sink.ProcessMessage(msg)
		}
		if source.videoHeader != nil {
			msg.Param1 = source.videoHeader
			msg.Type = wssAPI.MSG_FLV_TAG
			sink.ProcessMessage(msg)
		}
		if source.lastKeyFrame != nil {
			msg.Param1 = source.lastKeyFrame
			msg.Type = wssAPI.MSG_FLV_TAG
			logger.LOGD("not send last keyframe")
			//			sink.ProcessMessage(msg)
		}
	}
	return
}

func (source *streamSource) clearCache() {
	logger.LOGT("clear cache")
	source.metadata = nil
	source.audioHeader = nil
	source.videoHeader = nil
	source.lastKeyFrame = nil
}

func (source *streamSource) SetParent(parent wssAPI.MsgHandler) {
	source.parent = parent
}
