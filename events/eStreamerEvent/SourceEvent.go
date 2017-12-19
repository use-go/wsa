package eStreamerEvent

import (
	"net"

	"github.com/use-go/websocketStreamServer/wssAPI"
)

const (
	AddSource = "AddSource"
	DelSource = "DelSource"
	GetSource = "GetSource"
)

type EveAddSource struct {
	StreamName string
	RemoteIp   net.Addr
	Producer   wssAPI.Obj
	Id         int64      //outPut
	SrcObj     wssAPI.Obj //out
}

func (eveAddsource *EveAddSource) Receiver() string {
	return wssAPI.OBJ_StreamerServer
}

func (eveAddsource *EveAddSource) Type() string {
	return AddSource
}

type EveDelSource struct {
	StreamName string //in
	Id         int64  //in
}

func (eveAddsource *EveDelSource) Receiver() string {
	return wssAPI.OBJ_StreamerServer
}

func (eveAddsource *EveDelSource) Type() string {
	return DelSource
}

type EveGetSource struct {
	StreamName  string
	SrcObj      wssAPI.Obj
	HasProducer bool
}

func (eveAddsource *EveGetSource) Receiver() string {
	return wssAPI.OBJ_StreamerServer
}

func (eveAddsource *EveGetSource) Type() string {
	return GetSource
}
