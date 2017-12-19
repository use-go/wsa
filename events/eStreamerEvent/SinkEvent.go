package eStreamerEvent

import (
	"github.com/use-go/websocketStreamServer/wssAPI"
)

const (
	AddSink = "AddSink"
	DelSink = "DelSink"
)

type EveAddSink struct {
	StreamName string     //in
	SinkId     string     //in
	Sinker     wssAPI.Obj //in
	Added      bool       //out
}

func (eveAddSink *EveAddSink) Receiver() string {
	return wssAPI.OBJ_StreamerServer
}

func (eveAddSink *EveAddSink) Type() string {
	return AddSink
}

type EveDelSink struct {
	StreamName string //in
	SinkId     string //in
}

func (eveAddSink *EveDelSink) Receiver() string {
	return wssAPI.OBJ_StreamerServer
}

func (eveAddSink *EveDelSink) Type() string {
	return DelSink
}
