package eStreamerEvent

import (
	"github.com/use-go/websocket-streamserver/wssapi"
)

const (
	AddSink = "AddSink"
	DelSink = "DelSink"
)

type EveAddSink struct {
	StreamName string     //in
	SinkId     string     //in
	Sinker     wssapi.MsgHandler //in
	Added      bool       //out
}

func (eveAddSink *EveAddSink) Receiver() string {
	return wssapi.OBJStreamerServer
}

func (eveAddSink *EveAddSink) Type() string {
	return AddSink
}

type EveDelSink struct {
	StreamName string //in
	SinkId     string //in
}

func (eveAddSink *EveDelSink) Receiver() string {
	return wssapi.OBJStreamerServer
}

func (eveAddSink *EveDelSink) Type() string {
	return DelSink
}
