package RTMPService

import (
	"fmt"
	"sync"

	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

type rtmpPublisher struct {
	parent      wssAPI.MsgHandler
	bPublishing bool
	mutexStatus sync.RWMutex
	rtmp        *RTMP
}

func (rtmppublisher *rtmpPublisher) Init(msg *wssAPI.Msg) (err error) {
	rtmppublisher.rtmp = msg.Param1.(*RTMP)
	rtmppublisher.bPublishing = false
	return
}

func (rtmppublisher *rtmpPublisher) Start(msg *wssAPI.Msg) (err error) {
	rtmppublisher.startPublish()
	return
}

func (rtmppublisher *rtmpPublisher) Stop(msg *wssAPI.Msg) (err error) {
	rtmppublisher.stopPublish()
	return
}

func (rtmppublisher *rtmpPublisher) GetType() string {
	return ""
}

func (rtmppublisher *rtmpPublisher) HandleTask(task *wssAPI.Task) (err error) {
	return
}

func (rtmppublisher *rtmpPublisher) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

func (rtmppublisher *rtmpPublisher) isPublishing() bool {
	return rtmppublisher.bPublishing
}

func (rtmppublisher *rtmpPublisher) startPublish() bool {
	rtmppublisher.mutexStatus.Lock()
	defer rtmppublisher.mutexStatus.Unlock()
	if rtmppublisher.bPublishing == true {
		return false
	}
	err := rtmppublisher.rtmp.SendCtrl(RTMP_CTRL_streamBegin, 1, 0)
	if err != nil {
		logger.LOGE(err.Error())
		return false
	}
	err = rtmppublisher.rtmp.CmdStatus("status", "NetStream.Publish.Start",
		fmt.Sprintf("publish %s", rtmppublisher.rtmp.Link.Path), "", 0, RTMP_channel_Invoke)
	if err != nil {
		logger.LOGE(err.Error())
		return false
	}
	rtmppublisher.bPublishing = true
	return true
}

func (rtmppublisher *rtmpPublisher) stopPublish() bool {
	rtmppublisher.mutexStatus.Lock()
	defer rtmppublisher.mutexStatus.Unlock()
	if rtmppublisher.bPublishing == false {
		return false
	}
	err := rtmppublisher.rtmp.SendCtrl(RTMP_CTRL_streamEof, 1, 0)
	if err != nil {
		logger.LOGE(err.Error())
		return false
	}
	err = rtmppublisher.rtmp.CmdStatus("status", "NetStream.Unpublish.Succes",
		fmt.Sprintf("unpublish %s", rtmppublisher.rtmp.Link.Path), "", 0, RTMP_channel_Invoke)
	if err != nil {
		logger.LOGE(err.Error())
		return false
	}
	rtmppublisher.bPublishing = false
	return true
}
func (rtmppublisher *rtmpPublisher) SetParent(parent wssAPI.MsgHandler) {
	rtmppublisher.parent = parent
}
