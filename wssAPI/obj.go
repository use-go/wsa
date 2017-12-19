package wssAPI

import (
	"container/list"
	"errors"
)

//task
type Task interface {
	Receiver() string
	Type() string
}

//msg
type Msg struct {
	Type    string
	Version string
	Param1  interface{}
	Param2  interface{}
	Params  *list.List
}

//obj
type Obj interface {
	Init(msg *Msg) error
	Start(msg *Msg) error
	Stop(msg *Msg) error
	GetType() string
	HandleTask(task Task) error
	ProcessMessage(msg *Msg) error
	//	SetParent(parent Obj)
}

var svrbus Obj

func SetBus(bus Obj) {
	svrbus = bus
}

//Handle Task
func HandleTask(task Task) error {
	if svrbus != nil {
		return svrbus.HandleTask(task)
	}
	return errors.New("bus not ready")
}
