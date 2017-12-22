package eRTMPEvent

import (
	"github.com/use-go/websocket-streamserver/wssAPI"
)

const (
	PullRTMPStream = "PullRTMPStream"
)

type EvePullRTMPStream struct {
	SourceName string //用来创建和删除源，源名称和app+streamName 并不一样
	Protocol   string //RTMP,RTMPS,RTMPS and so on
	App        string
	Instance   string
	Address    string
	Port       int
	StreamName string
	Src        chan wssAPI.MsgHandler
}

func (evePullRTMPStream *EvePullRTMPStream) Receiver() string {
	return wssAPI.OBJRTMPServer
}

func (evePullRTMPStream *EvePullRTMPStream) Type() string {
	return PullRTMPStream
}

func (evePullRTMPStream *EvePullRTMPStream) Init(protocol, app, instance, addr, streamName, sourceName string, port int) {
	evePullRTMPStream.Protocol = protocol
	evePullRTMPStream.App = app
	evePullRTMPStream.Address = addr
	evePullRTMPStream.Instance = instance
	evePullRTMPStream.Port = port
	evePullRTMPStream.StreamName = streamName
	evePullRTMPStream.SourceName = sourceName
	evePullRTMPStream.Src = make(chan wssAPI.MsgHandler)
}

func (evePullRTMPStream *EvePullRTMPStream) Copy() (out *EvePullRTMPStream) {
	out = &EvePullRTMPStream{}
	out.Protocol = evePullRTMPStream.Protocol
	out.App = evePullRTMPStream.App
	out.Instance = evePullRTMPStream.Instance
	out.Address = evePullRTMPStream.Address
	out.Port = evePullRTMPStream.Port
	out.StreamName = evePullRTMPStream.StreamName
	out.SourceName = evePullRTMPStream.SourceName
	out.Src = evePullRTMPStream.Src
	return
}
