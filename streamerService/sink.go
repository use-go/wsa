package streamerService

import (
	"errors"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type streamSink struct {
	id     string
	sinker wssAPI.MsgHandler
	parent wssAPI.MsgHandler
}

//Init Of Sink
func (sink *streamSink) Init(msg *wssAPI.Msg) (err error) {
	if nil == msg || msg.Param1 == nil || msg.Param2 == nil {
		return errors.New("invalid init stream sink")
	}
	sink.id = msg.Param1.(string)
	sink.sinker = msg.Param2.(wssAPI.MsgHandler)
	return
}

//Start of sink
func (sink *streamSink) Start(msg *wssAPI.Msg) (err error) {
	//notify sinker stream start
	if sink.sinker == nil {
		logger.LOGE("sinker no seted")
		return errors.New("no sinker to start")
	}
	msg = &wssAPI.Msg{}
	msg.Type = wssAPI.MsgPlayStart
	logger.LOGT("start sink")
	//go sink.sinker.ProcessMessage(msg)
	sink.sinker.ProcessMessage(msg)
	return
}

//Stop of Sink
func (sink *streamSink) Stop(msg *wssAPI.Msg) (err error) {
	//notify sinker stream stop
	if sink.sinker == nil {
		logger.LOGE("sinker no seted")
		return errors.New("no sinker to stop")
	}
	msg = &wssAPI.Msg{}
	msg.Type = wssAPI.MsgPlayStop
	//go sink.sinker.ProcessMessage(msg)
	sink.sinker.ProcessMessage(msg)
	return
}

// GetType of sink
func (sink *streamSink) GetType() string {
	return streamTypeSink
}

//HandleTask of Sink
func (sink *streamSink) HandleTask(task *wssAPI.Task) (err error) {
	return
}

//ProcessMessage of sink
func (sink *streamSink) ProcessMessage(msg *wssAPI.Msg) (err error) {

	if sink.sinker != nil && msg.Type == wssAPI.MsgFlvTag {
		return sink.sinker.ProcessMessage(msg)
	}
	return
}

//ID of sink
func (sink *streamSink) ID() string {
	return sink.id
}

//SetParent of the handler
func (sink *streamSink) SetParent(parent wssAPI.MsgHandler) {
	sink.parent = parent
}
