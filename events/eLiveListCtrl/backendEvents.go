package eLiveListCtrl

import (
	"container/list"

	"github.com/use-go/websocket-streamserver/wssAPI"
)

const (
	EnableBlackList    = "EnableBlackList"
	SetBlackList       = "SetBlackList"
	EnableWhiteList    = "EnableWhiteList"
	SetWhiteList       = "SetWhiteList"
	GetLiveList        = "GetLiveList"
	GetLivePlayerCount = "GetLivePlayerCount"
	SetUpStreamApp     = "SetUpStreamApp"
)

//black list
type EveEnableBlackList struct {
	Enable bool
}

func (eveEnableBlackList *EveEnableBlackList) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveEnableBlackList *EveEnableBlackList) Type() string {
	return EnableBlackList
}

type EveSetBlackList struct {
	Add   bool
	Names *list.List
}

func (eveSetBlackList *EveSetBlackList) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveSetBlackList *EveSetBlackList) Type() string {
	return SetBlackList
}

//white list
type EveEnableWhiteList struct {
	Enable bool
}

func (eveEnableWhiteList *EveEnableWhiteList) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveEnableWhiteList *EveEnableWhiteList) Type() string {
	return EnableWhiteList
}

type EveSetWhiteList struct {
	Add   bool
	Names *list.List
}

func (eveSetWhiteList *EveSetWhiteList) Receiver() string {
	return wssAPI.OBJStreamerServer
}

func (eveSetWhiteList *EveSetWhiteList) Type() string {
	return SetWhiteList
}
