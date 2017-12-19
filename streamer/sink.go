package streamer

import (
	"errors"

	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

type streamSink struct {
	id     string
	sinker wssAPI.Obj
	parent wssAPI.Obj
}

func (sink *streamSink) Init(msg *wssAPI.Msg) (err error) {
	if nil == msg || msg.Param1 == nil || msg.Param2 == nil {
		return errors.New("invalid init stream sink")
	}
	sink.id = msg.Param1.(string)
	sink.sinker = msg.Param2.(wssAPI.Obj)
	return
}

func (sink *streamSink) Start(msg *wssAPI.Msg) (err error) {
	//notify sinker stream start
	if sink.sinker == nil {
		logger.LOGE("sinker no seted")
		return errors.New("no sinker to start")
	}
	msg = &wssAPI.Msg{}
	msg.Type = wssAPI.MSG_PLAY_START
	logger.LOGT("start sink")
	//go sink.sinker.ProcessMessage(msg)
	sink.sinker.ProcessMessage(msg)
	return
}

func (sink *streamSink) Stop(msg *wssAPI.Msg) (err error) {
	//notify sinker stream stop
	if sink.sinker == nil {
		logger.LOGE("sinker no seted")
		return errors.New("no sinker to stop")
	}
	msg = &wssAPI.Msg{}
	msg.Type = wssAPI.MSG_PLAY_STOP
	//go sink.sinker.ProcessMessage(msg)
	sink.sinker.ProcessMessage(msg)
	return
}

func (sink *streamSink) GetType() string {
	return streamTypeSink
}

func (sink *streamSink) HandleTask(task *wssAPI.Task) (err error) {
	return
}

func (sink *streamSink) ProcessMessage(msg *wssAPI.Msg) (err error) {

	if sink.sinker != nil && msg.Type == wssAPI.MSG_FLV_TAG {
		return sink.sinker.ProcessMessage(msg)
	}
	return
}

func (sink *streamSink) Id() string {
	return sink.id
}

func (sink *streamSink) SetParent(parent wssAPI.Obj) {
	sink.parent = parent
}
