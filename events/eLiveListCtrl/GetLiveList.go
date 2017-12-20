package eLiveListCtrl

import (
	"container/list"

	"github.com/use-go/websocket-streamserver/wssAPI"
)

type LiveInfo struct {
	StreamName  string
	PlayerCount int
	Ip          string
}

type EveGetLiveList struct {
	Lives *list.List //value =*LiveInfo
}

func (eveGetLiveList *EveGetLiveList) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveGetLiveList *EveGetLiveList) Type() string {
	return GetLiveList
}
