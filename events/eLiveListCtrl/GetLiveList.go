package eLiveListCtrl

import (
	"container/list"

	"github.com/use-go/websocket-streamserver/wssapi"
)

type LiveInfo struct {
	StreamName  string
	PlayerCount int
	IP          string
}

type EveGetLiveList struct {
	Lives *list.List //value =*LiveInfo
}

func (eveGetLiveList *EveGetLiveList) Receiver() string {
	return wssapi.OBJStreamerServer
}

func (eveGetLiveList *EveGetLiveList) Type() string {
	return GetLiveList
}
