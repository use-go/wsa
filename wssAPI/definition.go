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

var svrbus MsgHandler

//SetBus to associate the action handler
func SetBus(bus MsgHandler) {
	svrbus = bus
}

//HandleTask to deal Task from service bus
func HandleTask(task Task) error {
	if svrbus != nil {
		return svrbus.HandleTask(task)
	}
	return errors.New("service bus not ready")
}
