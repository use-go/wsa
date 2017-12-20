package wssAPI

import (
	"container/list"
	"errors"
)

//Task type of the msg
type Task interface {
	Receiver() string
	Type() string
}

//Msg content of the websocket
type Msg struct {
	Type    string
	Version string
	Param1  interface{}
	Param2  interface{}
	Params  *list.List
}

//MsgHandler : interface for websocket msg
type MsgHandler interface {
	Init(msg *Msg) error
	Start(msg *Msg) error
	Stop(msg *Msg) error
	GetType() string
	HandleTask(task Task) error
	ProcessMessage(msg *Msg) error
	//	SetParent(parent MsgHandler)
}

var handler MsgHandler

//SetHandler to associate the action handler for some object
func SetHandler(hdlr MsgHandler) {
	handler = hdlr
}

//HandleTask to deal Task from service bus
func HandleTask(task Task) error {
	if handler != nil {
		return handler.HandleTask(task)
	}
	return errors.New("service process not ready")
}

// Server Type
const (
	OBJProcess         = "HostProcess"
	OBJRTMPServer      = "RTMPServer"
	OBJWebSocketServer = "WebsocketServer"
	OBJBackendServer   = "BackendServer"
	OBJStreamerServer  = "StreamerServer"
	OBJRTSPServer      = "RTSPServer"
	OBJHLSServer       = "HLSServer"
	OBJDASHServer      = `DASHServer`
)

// MSG Event Notify Type
const (
	MsgFlvTag            = "FLVTag"
	MsgGetSourceNotify   = "MSG.GetSource.Notify.Async"
	MsgGetSourceFailed   = "MSG.GetSource.Failed"
	MsgSourceClosedForce = "MSG.SourceClosed.Force"
	MsgPublishStart      = "NetStream.Publish.Start"
	MsgPublishStop       = "NetStream.Publish.Stop"
	MsgPlayStart         = "NetStream.Play.Start"
	MsgPlayStop          = "NetStream.Play.Stop"
)
