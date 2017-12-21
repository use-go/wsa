package eStreamerEvent

import (
	"net"

	"github.com/use-go/websocket-streamserver/wssAPI"
)

const (
	AddSource = "AddSource"
	DelSource = "DelSource"
	GetSource = "GetSource"
)

type EveAddSource struct {
	StreamName string
	RemoteIp   net.Addr
	Producer   wssAPI.MsgHandler
	ID         int64      //outPut
	SrcObj     wssAPI.MsgHandler //out
}

func (eveAddsource *EveAddSource) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveAddsource *EveAddSource) Type() string {
	return AddSource
}

type EveDelSource struct {
	StreamName string //in
	ID         int64  //in
}

func (eveAddsource *EveDelSource) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveAddsource *EveDelSource) Type() string {
	return DelSource
}

type EveGetSource struct {
	StreamName  string
	SrcObj      wssAPI.MsgHandler
	HasProducer bool
}

func (eveAddsource *EveGetSource) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveAddsource *EveGetSource) Type() string {
	return GetSource
}
