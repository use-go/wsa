package eLiveListCtrl

import (
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type EveGetLivePlayerCount struct {
	LiveName string
	Count    int
}

func (eveGetLivePlayerCount *EveGetLivePlayerCount) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveGetLivePlayerCount *EveGetLivePlayerCount) Type() string {
	return GetLivePlayerCount
}
