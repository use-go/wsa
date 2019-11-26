package eLiveListCtrl

import (
	"container/list"

	"github.com/use-go/websocket-streamserver/wssapi"
)

// const string to describe the action
const (
	EnableBlackList    = "EnableBlackList"
	SetBlackList       = "SetBlackList"
	EnableWhiteList    = "EnableWhiteList"
	SetWhiteList       = "SetWhiteList"
	GetLiveList        = "GetLiveList"
	GetLivePlayerCount = "GetLivePlayerCount"
	SetUpStreamApp     = "SetUpStreamApp"
)

//EveEnableBlackList black list
type EveEnableBlackList struct {
	Enable bool
}

//Receiver of Event
func (eveEnableBlackList *EveEnableBlackList) Receiver() string {
	return wssapi.OBJStreamerServer
}

//Type of Event
func (eveEnableBlackList *EveEnableBlackList) Type() string {
	return EnableBlackList
}

//EveSetBlackList  Event
type EveSetBlackList struct {
	Add   bool
	Names *list.List
}

//Receiver of EveSetBlackList
func (eveSetBlackList *EveSetBlackList) Receiver() string {
	return wssapi.OBJStreamerServer
}

//Type of EveSetBlackList
func (eveSetBlackList *EveSetBlackList) Type() string {
	return SetBlackList
}

//EveEnableWhiteList white list
type EveEnableWhiteList struct {
	Enable bool
}

//Receiver of eveEnableWhiteList
func (eveEnableWhiteList *EveEnableWhiteList) Receiver() string {
	return wssapi.OBJStreamerServer
}

// Type of eveEnableWhiteList
func (eveEnableWhiteList *EveEnableWhiteList) Type() string {
	return EnableWhiteList
}

//EveSetWhiteList struct
type EveSetWhiteList struct {
	Add   bool
	Names *list.List
}

//Receiver of eveSetWhiteList
func (eveSetWhiteList *EveSetWhiteList) Receiver() string {
	return wssapi.OBJStreamerServer
}

//Type of eveSetWhiteList
func (eveSetWhiteList *EveSetWhiteList) Type() string {
	return SetWhiteList
}
