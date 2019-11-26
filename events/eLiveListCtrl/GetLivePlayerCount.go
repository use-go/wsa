package eLiveListCtrl

import (
	"github.com/use-go/websocket-streamserver/wssapi"
)

type EveGetLivePlayerCount struct {
	LiveName string
	Count    int
}

func (eveGetLivePlayerCount *EveGetLivePlayerCount) Receiver() string {
	return wssapi.OBJStreamerServer
}

func (eveGetLivePlayerCount *EveGetLivePlayerCount) Type() string {
	return GetLivePlayerCount
}
