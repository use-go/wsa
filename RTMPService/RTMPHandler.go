package RTMPService

import (
	"container/list"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/use-go/websocket-streamserver/events/eStreamerEvent"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type RTMPHandler struct {
	parent       wssAPI.MsgHandler
	mutexStatus  sync.RWMutex
	rtmpInstance *RTMP
	source       wssAPI.MsgHandler
	sinke        wssAPI.MsgHandler
	srcAdded     bool
	sinkAdded    bool
	streamName   string
	clientID     string
	playInfo     RTMPPlayInfo
	app          string
	player       rtmpPlayer
	publisher    rtmpPublisher
	srcID        int64
}
type RTMPPlayInfo struct {
	playReset      bool
	playing        bool //true for thread send playing data
	waitPlaying    *sync.WaitGroup
	mutexCache     sync.RWMutex
	cache          *list.List
	audioHeader    *flv.FlvTag
	videoHeader    *flv.FlvTag
	metadata       *flv.FlvTag
	keyFrameWrited bool
	beginTime      uint32
	startTime      float32
	duration       float32
	reset          bool
}

func (rtmpHandler *RTMPHandler) Init(msg *wssAPI.Msg) (err error) {
	rtmpHandler.rtmpInstance = msg.Param1.(*RTMP)
	msgInit := &wssAPI.Msg{}
	msgInit.Param1 = rtmpHandler.rtmpInstance
	rtmpHandler.player.Init(msgInit)
	rtmpHandler.publisher.Init(msgInit)
	return
}

func (rtmpHandler *RTMPHandler) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (rtmpHandler *RTMPHandler) Stop(msg *wssAPI.Msg) (err error) {
	if rtmpHandler.srcAdded {
		taskDelSrc := &eStreamerEvent.EveDelSource{}
		taskDelSrc.StreamName = rtmpHandler.streamName
		taskDelSrc.Id = rtmpHandler.srcID
		wssAPI.HandleTask(taskDelSrc)
		logger.LOGT("del source:" + rtmpHandler.streamName)
		rtmpHandler.srcAdded = false
	}
	if rtmpHandler.sinkAdded {
		taskDelSink := &eStreamerEvent.EveDelSink{}
		taskDelSink.StreamName = rtmpHandler.streamName
		taskDelSink.SinkId = rtmpHandler.clientID
		wssAPI.HandleTask(taskDelSink)
		rtmpHandler.sinkAdded = false
		logger.LOGT("del sinker:" + rtmpHandler.clientID)
	}
	rtmpHandler.player.Stop(msg)
	rtmpHandler.publisher.Stop(msg)
	return
}

func (rtmpHandler *RTMPHandler) GetType() string {
	return rtmpTypeHandler
}

func (rtmpHandler *RTMPHandler) HandleTask(task wssAPI.Task) (err error) {
	if task.Receiver() != rtmpHandler.GetType() {
		logger.LOGW(fmt.Sprintf("invalid task receiver in rtmpHandler :%s", task.Receiver()))
		return errors.New("invalid task")
	}
	return
}

func (rtmpHandler *RTMPHandler) ProcessMessage(msg *wssAPI.Msg) (err error) {

	if msg == nil {
		return errors.New("nil message")
	}
	switch msg.Type {
	case wssAPI.MSG_GetSource_NOTIFY:
		rtmpHandler.sinkAdded = true
	case wssAPI.MSG_GetSource_Failed:
		//发送404
		rtmpHandler.rtmpInstance.CmdStatus("error", "NetStream.Play.StreamNotFound",
			"paly failed", rtmpHandler.streamName, 0, RTMP_channel_Invoke)
	case wssAPI.MSG_SourceClosed_Force:
		rtmpHandler.srcAdded = false
	case wssAPI.MSG_FLV_TAG:
		tag := msg.Param1.(*flv.FlvTag)
		err = rtmpHandler.player.appendFlvTag(tag)

	case wssAPI.MSG_PLAY_START:
		rtmpHandler.player.startPlay()
		return
	case wssAPI.MSG_PLAY_STOP:
		rtmpHandler.mutexStatus.Lock()
		defer rtmpHandler.mutexStatus.Unlock()
		rtmpHandler.sourceInvalid()
		return
	case wssAPI.MSG_PUBLISH_START:
		rtmpHandler.mutexStatus.Lock()
		defer rtmpHandler.mutexStatus.Unlock()
		if err != nil {
			logger.LOGE("start publish failed")
			return
		}
		if false == rtmpHandler.publisher.startPublish() {
			logger.LOGE("start publish falied")
			if true == rtmpHandler.srcAdded {
				taskDelSrc := &eStreamerEvent.EveDelSource{}
				taskDelSrc.StreamName = rtmpHandler.streamName
				taskDelSrc.Id = rtmpHandler.srcID
				wssAPI.HandleTask(taskDelSrc)
			}
		}
		return
	case wssAPI.MSG_PUBLISH_STOP:
		rtmpHandler.mutexStatus.Lock()
		defer rtmpHandler.mutexStatus.Unlock()
		if err != nil {
			logger.LOGE("stop publish failed")
			return
		}
		rtmpHandler.publisher.stopPublish()
		return
	default:
		logger.LOGW(fmt.Sprintf("msg type: %s not processed", msg.Type))
		return
	}
	return
}

func (rtmpHandler *RTMPHandler) sourceInvalid() {
	logger.LOGT("stop play,keep sink")
	rtmpHandler.player.stopPlay()
}

func (rtmpHandler *RTMPHandler) HandleRTMPPacket(packet *RTMPPacket) (err error) {
	if nil == packet {
		rtmpHandler.Stop(nil)
		return
	}
	switch packet.MessageTypeID {
	case RTMP_PACKET_TYPE_CHUNK_SIZE:
		rtmpHandler.rtmpInstance.RecvChunkSize, err = AMF0DecodeInt32(packet.Body)
		logger.LOGT(fmt.Sprintf("chunk size:%d", rtmpHandler.rtmpInstance.RecvChunkSize))
	case RTMP_PACKET_TYPE_CONTROL:
		err = rtmpHandler.rtmpInstance.HandleControl(packet)
	case RTMP_PACKET_TYPE_BYTES_READ_REPORT:
		//		logger.LOGT(packet.TimeStamp)
		//		logger.LOGT(packet.Body)
		//		logger.LOGT("bytes read repost")
	case RTMP_PACKET_TYPE_SERVER_BW:
		rtmpHandler.rtmpInstance.TargetBW, err = AMF0DecodeInt32(packet.Body)
		logger.LOGT(fmt.Sprintf("确认窗口大小 %d", rtmpHandler.rtmpInstance.TargetBW))
	case RTMP_PACKET_TYPE_CLIENT_BW:
		rtmpHandler.rtmpInstance.SelfBW, err = AMF0DecodeInt32(packet.Body)
		rtmpHandler.rtmpInstance.LimitType = uint32(packet.Body[4])
		logger.LOGT(fmt.Sprintf("设置对端宽带 %d %d ", rtmpHandler.rtmpInstance.SelfBW, rtmpHandler.rtmpInstance.LimitType))
	case RTMP_PACKET_TYPE_FLEX_MESSAGE:
		err = rtmpHandler.handleInvoke(packet)
	case RTMP_PACKET_TYPE_INVOKE:
		err = rtmpHandler.handleInvoke(packet)
	case RTMP_PACKET_TYPE_AUDIO:
		return rtmpHandler.sendFlvToSrc(packet)
	case RTMP_PACKET_TYPE_VIDEO:
		return rtmpHandler.sendFlvToSrc(packet)
	case RTMP_PACKET_TYPE_INFO:
		return rtmpHandler.sendFlvToSrc(packet)
	default:
		logger.LOGW(fmt.Sprintf("rtmp packet type %d not processed", packet.MessageTypeID))
	}
	return
}

func (rtmpHandler *RTMPHandler) sendFlvToSrc(pkt *RTMPPacket) (err error) {
	if rtmpHandler.publisher.isPublishing() && wssAPI.InterfaceValid(rtmpHandler.source) {
		msg := &wssAPI.Msg{}
		msg.Type = wssAPI.MSG_FLV_TAG
		msg.Param1 = pkt.ToFLVTag()
		err = rtmpHandler.source.ProcessMessage(msg)
		if err != nil {
			logger.LOGE(err.Error())
			rtmpHandler.Stop(nil)
		}
		return
	} else {
		logger.LOGE("bad status")
	}
	return
}

func (rtmpHandler *RTMPHandler) handleInvoke(packet *RTMPPacket) (err error) {
	var amfobj *AMF0Object
	if RTMP_PACKET_TYPE_FLEX_MESSAGE == packet.MessageTypeID {
		amfobj, err = AMF0DecodeObj(packet.Body[1:])
	} else {
		amfobj, err = AMF0DecodeObj(packet.Body)
	}
	if err != nil {
		logger.LOGE("recved invalid amf0 object")
		return
	}
	if amfobj.Props.Len() == 0 {
		logger.LOGT(packet.Body)
		logger.LOGT(string(packet.Body))
		return
	}

	method := amfobj.Props.Front().Value.(*AMF0Property)

	switch method.Value.StrValue {
	case "connect":
		cmdObj := amfobj.AMF0GetPropByIndex(2)
		if cmdObj != nil {
			rtmpHandler.app = cmdObj.Value.ObjValue.AMF0GetPropByName("app").Value.StrValue
			if strings.HasSuffix(rtmpHandler.app, "/") {
				rtmpHandler.app = strings.TrimSuffix(rtmpHandler.app, "/")
			}
		}
		if rtmpHandler.app != serviceConfig.LivePath {
			logger.LOGE(rtmpHandler.app)
			logger.LOGE(serviceConfig.LivePath)
			logger.LOGW("path wrong")
		}
		err = rtmpHandler.rtmpInstance.AcknowledgementBW()
		if err != nil {
			return
		}
		err = rtmpHandler.rtmpInstance.SetPeerBW()
		if err != nil {
			return
		}
		//err = rtmpHandler.rtmpInstance.SetChunkSize(RTMP_better_chunk_size)
		//		if err != nil {
		//			return
		//		}
		err = rtmpHandler.rtmpInstance.OnBWDone()
		if err != nil {
			return
		}
		err = rtmpHandler.rtmpInstance.ConnectResult(amfobj)
		if err != nil {
			return
		}
	case "_checkbw":
		err = rtmpHandler.rtmpInstance.OnBWCheck()
	case "_result":
		rtmpHandler.handle_result(amfobj)
	case "releaseStream":
		//		idx := amfobj.AMF0GetPropByIndex(1).Value.NumValue
		//		err = rtmpHandler.rtmpInstance.CmdError("error", "NetConnection.Call.Failed",
		//			fmt.Sprintf("Method not found (%s).", "releaseStream"), idx)
	case "FCPublish":
		//		idx := amfobj.AMF0GetPropByIndex(1).Value.NumValue
		//		err = rtmpHandler.rtmpInstance.CmdError("error", "NetConnection.Call.Failed",
		//			fmt.Sprintf("Method not found (%s).", "FCPublish"), idx)
	case "createStream":
		idx := amfobj.AMF0GetPropByIndex(1).Value.NumValue
		err = rtmpHandler.rtmpInstance.CmdNumberResult(idx, 1.0)
	case "publish":
		//check prop
		if amfobj.Props.Len() < 4 {
			logger.LOGE("invalid props length")
			err = errors.New("invalid amf obj for publish")
			return
		}

		rtmpHandler.mutexStatus.Lock()
		defer rtmpHandler.mutexStatus.Unlock()
		//check status
		if true == rtmpHandler.publisher.isPublishing() {
			logger.LOGE("publish on bad status ")
			idx := amfobj.AMF0GetPropByIndex(1).Value.NumValue
			err = rtmpHandler.rtmpInstance.CmdError("error", "NetStream.Publish.Denied",
				fmt.Sprintf("can not publish (%s).", "publish"), idx)
			return
		}
		//add to source
		rtmpHandler.streamName = rtmpHandler.app + "/" + amfobj.AMF0GetPropByIndex(3).Value.StrValue
		taskAddSrc := &eStreamerEvent.EveAddSource{}
		taskAddSrc.Producer = rtmpHandler
		taskAddSrc.StreamName = rtmpHandler.streamName
		taskAddSrc.RemoteIp = rtmpHandler.rtmpInstance.Conn.RemoteAddr()
		err = wssAPI.HandleTask(taskAddSrc)
		if err != nil {
			logger.LOGE("add source failed:" + err.Error())
			err = rtmpHandler.rtmpInstance.CmdStatus("error", "NetStream.Publish.BadName",
				fmt.Sprintf("publish %s.", rtmpHandler.streamName), "", 0, RTMP_channel_Invoke)
			rtmpHandler.streamName = ""
			return err
		}
		rtmpHandler.source = taskAddSrc.SrcObj
		rtmpHandler.srcID = taskAddSrc.Id
		if rtmpHandler.source == nil {
			logger.LOGE("add source failed:")
			err = rtmpHandler.rtmpInstance.CmdStatus("error", "NetStream.Publish.BadName",
				fmt.Sprintf("publish %s.", rtmpHandler.streamName), "", 0, RTMP_channel_Invoke)
			rtmpHandler.streamName = ""
			return errors.New("bad name")
		}
		rtmpHandler.srcAdded = true
		rtmpHandler.rtmpInstance.Link.Path = amfobj.AMF0GetPropByIndex(2).Value.StrValue
		if false == rtmpHandler.publisher.startPublish() {
			logger.LOGE("start publish failed:" + rtmpHandler.streamName)
			//streamer.DelSource(rtmpHandler.streamName)
			taskDelSrc := &eStreamerEvent.EveDelSource{}
			taskDelSrc.StreamName = rtmpHandler.streamName
			taskDelSrc.Id = rtmpHandler.srcID
			wssAPI.HandleTask(taskDelSrc)
			return
		}
	case "FCUnpublish":
		rtmpHandler.mutexStatus.Lock()
		defer rtmpHandler.mutexStatus.Unlock()
	case "deleteStream":
		rtmpHandler.mutexStatus.Lock()
		defer rtmpHandler.mutexStatus.Unlock()
	//do nothing now
	case "play":
		rtmpHandler.streamName = rtmpHandler.app + "/" + amfobj.AMF0GetPropByIndex(3).Value.StrValue
		rtmpHandler.rtmpInstance.Link.Path = rtmpHandler.streamName
		startTime := -2
		duration := -1
		reset := false
		rtmpHandler.playInfo.startTime = -2
		rtmpHandler.playInfo.duration = -1
		rtmpHandler.playInfo.reset = false
		if amfobj.Props.Len() >= 5 {
			rtmpHandler.playInfo.startTime = float32(amfobj.AMF0GetPropByIndex(4).Value.NumValue)
		}
		if amfobj.Props.Len() >= 6 {
			rtmpHandler.playInfo.duration = float32(amfobj.AMF0GetPropByIndex(5).Value.NumValue)
			if rtmpHandler.playInfo.duration < 0 {
				rtmpHandler.playInfo.duration = -1
			}
		}
		if amfobj.Props.Len() >= 7 {
			rtmpHandler.playInfo.reset = amfobj.AMF0GetPropByIndex(6).Value.BoolValue
		}

		//check player status,if playing,error
		if false == rtmpHandler.player.setPlayParams(rtmpHandler.streamName, startTime, duration, reset) {
			err = rtmpHandler.rtmpInstance.CmdStatus("error", "NetStream.Play.Failed",
				"paly failed", rtmpHandler.streamName, 0, RTMP_channel_Invoke)

			return nil
		}
		err = rtmpHandler.rtmpInstance.SendCtrl(RTMP_CTRL_streamBegin, 1, 0)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}

		if true == rtmpHandler.playInfo.playReset {
			err = rtmpHandler.rtmpInstance.CmdStatus("status", "NetStream.Play.Reset",
				fmt.Sprintf("Playing and resetting %s", rtmpHandler.rtmpInstance.Link.Path),
				rtmpHandler.rtmpInstance.Link.Path, 0, RTMP_channel_Invoke)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		}

		err = rtmpHandler.rtmpInstance.CmdStatus("status", "NetStream.Play.Start",
			fmt.Sprintf("Started playing %s", rtmpHandler.rtmpInstance.Link.Path), rtmpHandler.rtmpInstance.Link.Path, 0, RTMP_channel_Invoke)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}

		rtmpHandler.clientID = wssAPI.GenerateGUID()
		taskAddSink := &eStreamerEvent.EveAddSink{}
		taskAddSink.StreamName = rtmpHandler.streamName
		taskAddSink.SinkId = rtmpHandler.clientID
		taskAddSink.Sinker = rtmpHandler
		err = wssAPI.HandleTask(taskAddSink)
		if err != nil {
			//404
			err = rtmpHandler.rtmpInstance.CmdStatus("error", "NetStream.Play.StreamNotFound",
				"paly failed", rtmpHandler.streamName, 0, RTMP_channel_Invoke)
			return nil
		}
		rtmpHandler.sinkAdded = taskAddSink.Added
	case "_error":
		amfobj.Dump()
	case "closeStream":
		amfobj.Dump()
	default:
		logger.LOGW(fmt.Sprintf("rtmp method <%s> not processed", method.Value.StrValue))
	}
	return
}

func (rtmpHandler *RTMPHandler) handle_result(amfobj *AMF0Object) {
	transactionID := int32(amfobj.AMF0GetPropByIndex(1).Value.NumValue)
	resultMethod := rtmpHandler.rtmpInstance.methodCache[transactionID]
	switch resultMethod {
	case "_onbwcheck":
	default:
		logger.LOGW("result of " + resultMethod + " not processed")
	}
}

func (rtmpHandler *RTMPHandler) startPublishing() (err error) {
	err = rtmpHandler.rtmpInstance.SendCtrl(RTMP_CTRL_streamBegin, 1, 0)
	if err != nil {
		logger.LOGE(err.Error())
		return nil
	}
	err = rtmpHandler.rtmpInstance.CmdStatus("status", "NetStream.Publish.Start",
		fmt.Sprintf("publish %s", rtmpHandler.rtmpInstance.Link.Path), "", 0, RTMP_channel_Invoke)
	if err != nil {
		logger.LOGE(err.Error())
		return nil
	}
	rtmpHandler.publisher.startPublish()
	return
}

func (rtmpHandler *RTMPHandler) isPlaying() bool {
	return rtmpHandler.player.IsPlaying()
}

func (rtmpHandler *RTMPHandler) SetParent(parent wssAPI.MsgHandler) {
	rtmpHandler.parent = parent
}
