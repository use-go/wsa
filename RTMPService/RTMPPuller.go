package RTMPService

import (
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/mediaTypes/flv"
	"github.com/use-go/websocketStreamServer/wssAPI"

	"github.com/use-go/websocketStreamServer/events/eLiveListCtrl"
	"github.com/use-go/websocketStreamServer/events/eRTMPEvent"
	"github.com/use-go/websocketStreamServer/events/eStreamerEvent"
)

//RTMPPuller struct
type RTMPPuller struct {
	rtmp       RTMP
	parent     wssAPI.MsgHandler
	src        wssAPI.MsgHandler
	pullParams *eRTMPEvent.EvePullRTMPStream
	waitRead   *sync.WaitGroup
	reading    bool
	srcID      int64
	chValid    bool
	metaDatas  *list.List
}

//PullRTMPLive From Server
func PullRTMPLive(task *eRTMPEvent.EvePullRTMPStream) {
	rtmppuller := &RTMPPuller{}
	msg := &wssAPI.Msg{}
	msg.Param1 = task
	rtmppuller.Init(msg)
	rtmppuller.Start(nil)
}

//Init Action
func (rtmppuller *RTMPPuller) Init(msg *wssAPI.Msg) (err error) {
	rtmppuller.pullParams = msg.Param1.(*eRTMPEvent.EvePullRTMPStream).Copy()
	rtmppuller.initRTMPLink()
	rtmppuller.waitRead = new(sync.WaitGroup)
	rtmppuller.chValid = true
	rtmppuller.metaDatas = list.New()
	return
}

func (rtmppuller *RTMPPuller) closeCh() {
	if rtmppuller.chValid {
		rtmppuller.chValid = false
		logger.LOGD("close ch")
		close(rtmppuller.pullParams.Src)
	}
}

func (rtmppuller *RTMPPuller) initRTMPLink() {
	rtmppuller.rtmp.Link.Protocol = rtmppuller.pullParams.Protocol
	rtmppuller.rtmp.Link.App = rtmppuller.pullParams.App
	rtmppuller.rtmp.Link.Path = rtmppuller.pullParams.StreamName
	rtmppuller.rtmp.Link.TcUrl = rtmppuller.pullParams.Protocol + "://" +
		rtmppuller.pullParams.Address + ":" +
		strconv.Itoa(rtmppuller.pullParams.Port) + "/" +
		rtmppuller.pullParams.App
	//if len(rtmppuller.pullParams.Instance)>0{
	//	rtmppuller.rtmp.Link.TcUrl+="/"+rtmppuller.pullParams.Instance
	//}
	logger.LOGD(rtmppuller.rtmp.Link.TcUrl)
}

//Start Action
func (rtmppuller *RTMPPuller) Start(msg *wssAPI.Msg) (err error) {
	defer func() {
		if err != nil {
			logger.LOGE("start failed")
			rtmppuller.closeCh()
			if nil != rtmppuller.rtmp.Conn {
				rtmppuller.rtmp.Conn.Close()
				rtmppuller.rtmp.Conn = nil
			}
		}
	}()
	//start pull
	//connect
	addr := rtmppuller.pullParams.Address + ":" + strconv.Itoa(rtmppuller.pullParams.Port)

	conn, err := net.Dial("tcp", addr)
	logger.LOGT(addr)
	if err != nil {
		logger.LOGE("connect failed:" + err.Error())
		return
	}
	rtmppuller.rtmp.Init(conn)
	//just simple handshake
	err = rtmppuller.handleShake()
	if err != nil {
		logger.LOGE("handle shake failed")
		return
	}
	//start read thread
	rtmppuller.rtmp.BytesIn = 3073
	go rtmppuller.threadRead()
	//play
	err = rtmppuller.play()
	if err != nil {
		logger.LOGE("play failed")
		return
	}
	return
}

//Stop Action
func (rtmppuller *RTMPPuller) Stop(msg *wssAPI.Msg) (err error) {
	//stop pull
	logger.LOGT("stop rtmppuller")
	rtmppuller.reading = false
	rtmppuller.waitRead.Wait()

	if wssAPI.InterfaceValid(rtmppuller.rtmp.Conn) {
		rtmppuller.rtmp.Conn.Close()
		rtmppuller.rtmp.Conn = nil
	}
	//del src
	if wssAPI.InterfaceValid(rtmppuller.src) {
		taskDelSrc := &eStreamerEvent.EveDelSource{}
		taskDelSrc.StreamName = rtmppuller.pullParams.SourceName
		taskDelSrc.Id = rtmppuller.srcID
		err = wssAPI.HandleTask(taskDelSrc)
		if err != nil {
			logger.LOGE(err.Error())
		}
		rtmppuller.src = nil
	}

	return
}

func (rtmppuller *RTMPPuller) handleShake() (err error) {
	randomSize := 1528
	//send c0
	conn := rtmppuller.rtmp.Conn
	c0 := make([]byte, 1)
	c0[0] = 3
	_, err = wssAPI.TCPWriteTimeDuration(conn, c0, time.Duration(serviceConfig.TimeoutSec)*time.Second)
	if err != nil {
		logger.LOGE("send c0 failed")
		return
	}
	//send c1
	c1 := make([]byte, randomSize+4+4)
	for idx := 8; idx < len(c1); idx++ {
		c1[idx] = byte(rand.Intn(255))
	}
	_, err = wssAPI.TCPWriteTimeDuration(conn, c1, time.Duration(serviceConfig.TimeoutSec)*time.Second)
	if err != nil {
		logger.LOGE("send c1 failed")
		return
	}
	//read s0
	s0, err := wssAPI.TCPReadTimeDuration(conn, 1, time.Duration(serviceConfig.TimeoutSec)*time.Second)
	if err != nil {
		logger.LOGE("read s0 failed")
		return
	}
	logger.LOGT(s0)
	//read s1
	s1, err := wssAPI.TCPReadTimeDuration(conn, randomSize+8, time.Duration(serviceConfig.TimeoutSec)*time.Second)
	if err != nil {
		logger.LOGE("read s1 failed")
		return
	}
	//send c2
	_, err = wssAPI.TCPWriteTimeDuration(conn, s1, time.Duration(serviceConfig.TimeoutSec)*time.Second)
	if err != nil {
		logger.LOGE("send c2 failed")
		return
	}
	//read s2
	s2, err := wssAPI.TCPReadTimeDuration(conn, randomSize+8, time.Duration(serviceConfig.TimeoutSec)*time.Second)
	if err != nil {
		logger.LOGE("read s2 failed")
		return
	}
	for idx := 0; idx < len(s2); idx++ {
		if c1[idx] != s2[idx] {
			logger.LOGE("invalid s2")
			return errors.New("invalid s2")
		}
	}
	logger.LOGT("handleshake ok")
	return
}

func (rtmppuller *RTMPPuller) GetType() string {
	return rtmpTypePuller
}

func (rtmppuller *RTMPPuller) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (rtmppuller *RTMPPuller) ProcessMessage(msg *wssAPI.Msg) (err error) {
	switch msg.Type {
	case wssAPI.MSG_SourceClosed_Force:
		logger.LOGT("rtmp rtmppuller data sink closed")
		rtmppuller.src = nil
		rtmppuller.reading = false
	default:
		logger.LOGE(msg.Type + " not processed")
	}
	return
}

func (rtmppuller *RTMPPuller) SetParent(parent wssAPI.MsgHandler) {
	rtmppuller.parent = parent
}

func (rtmppuller *RTMPPuller) HandleControl(pkt *RTMPPacket) (err error) {
	ctype, err := AMF0DecodeInt16(pkt.Body)
	if err != nil {
		return
	}
	switch ctype {
	case RTMP_CTRL_streamBegin:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGT(fmt.Sprintf("stream begin:%d", streamID))
	case RTMP_CTRL_streamEof:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGT(fmt.Sprintf("stream eof:%d", streamID))
		err = errors.New("stream eof ")
	case RTMP_CTRL_streamDry:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGT(fmt.Sprintf("stream dry:%d", streamID))
	case RTMP_CTRL_setBufferLength:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		buffMS, _ := AMF0DecodeInt32(pkt.Body[6:])
		rtmppuller.rtmp.buffMS = uint32(buffMS)
		rtmppuller.rtmp.StreamID = uint32(streamID)
		//logger.LOGI(fmt.Sprintf("set buffer length --streamid:%d--buffer length:%d", rtmppuller.StreamID, rtmppuller.buffMS))
	case RTMP_CTRL_streamIsRecorded:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGT(fmt.Sprintf("stream %d is recorded", streamID))
	case RTMP_CTRL_pingRequest:
		timestamp, _ := AMF0DecodeInt32(pkt.Body[2:])
		rtmppuller.rtmp.pingResponse(timestamp)
		logger.LOGT(fmt.Sprintf("ping :%d", timestamp))
	case RTMP_CTRL_pingResponse:
		timestamp, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGF(fmt.Sprintf("pong :%d", timestamp))
	case RTMP_CTRL_streamBufferEmpty:
		//logger.LOGT(fmt.Sprintf("buffer empty"))
	case RTMP_CTRL_streamBufferReady:
		//logger.LOGT(fmt.Sprintf("buffer ready"))
	default:
		logger.LOGI(fmt.Sprintf("rtmp control type:%d not processed", ctype))
	}
	return
}

func (rtmppuller *RTMPPuller) threadRead() {
	rtmppuller.reading = true
	rtmppuller.waitRead.Add(1)
	defer func() {
		rtmppuller.waitRead.Done()
		rtmppuller.Stop(nil)
		rtmppuller.closeCh()
		logger.LOGT("stop read,close conn")
	}()
	for rtmppuller.reading {
		packet, err := rtmppuller.readRTMPPkt()
		if err != nil {
			logger.LOGE(err.Error())
			rtmppuller.reading = false
			return
		}
		switch packet.MessageTypeID {
		case RTMP_PACKET_TYPE_CHUNK_SIZE:
			rtmppuller.rtmp.RecvChunkSize, err = AMF0DecodeInt32(packet.Body)
			logger.LOGT(fmt.Sprintf("chunk size:%d", rtmppuller.rtmp.RecvChunkSize))
		case RTMP_PACKET_TYPE_CONTROL:
			err = rtmppuller.rtmp.HandleControl(packet)
		case RTMP_PACKET_TYPE_BYTES_READ_REPORT:
			logger.LOGT("bytes read report")
		case RTMP_PACKET_TYPE_SERVER_BW:
			rtmppuller.rtmp.AcknowledgementWindowSize, err = AMF0DecodeInt32(packet.Body)
			logger.LOGT(fmt.Sprintf("acknowledgment size %d", rtmppuller.rtmp.TargetBW))
		case RTMP_PACKET_TYPE_CLIENT_BW:
			rtmppuller.rtmp.SelfBW, err = AMF0DecodeInt32(packet.Body)
			rtmppuller.rtmp.LimitType = uint32(packet.Body[4])
			logger.LOGT(fmt.Sprintf("peer band width %d %d ", rtmppuller.rtmp.SelfBW, rtmppuller.rtmp.LimitType))
		case RTMP_PACKET_TYPE_FLEX_MESSAGE:
			err = rtmppuller.handleInvoke(packet)
		case RTMP_PACKET_TYPE_INVOKE:
			err = rtmppuller.handleInvoke(packet)
		case RTMP_PACKET_TYPE_AUDIO:
			err = rtmppuller.sendFlvToSrc(packet)
		case RTMP_PACKET_TYPE_VIDEO:
			err = rtmppuller.sendFlvToSrc(packet)
		case RTMP_PACKET_TYPE_INFO:
			rtmppuller.metaDatas.PushBack(packet.Copy())
		case RTMP_PACKET_TYPE_FLASH_VIDEO:
			err = rtmppuller.processAggregation(packet)
		default:
			logger.LOGW(fmt.Sprintf("rtmp packet type %d not processed", packet.MessageTypeID))
		}
		if err != nil {
			rtmppuller.reading = false
		}
	}
}

func (rtmppuller *RTMPPuller) sendFlvToSrc(pkt *RTMPPacket) (err error) {
	if wssAPI.InterfaceIsNil(rtmppuller.src) && pkt.MessageTypeID != flv.FLV_TAG_ScriptData {

		rtmppuller.CreatePlaySRC()
	}
	if wssAPI.InterfaceValid(rtmppuller.src) {
		if rtmppuller.metaDatas.Len() > 0 {
			for e := rtmppuller.metaDatas.Front(); e != nil; e = e.Next() {
				metaDataPkt := e.Value.(*RTMPPacket).ToFLVTag()
				msg := &wssAPI.Msg{Type: wssAPI.MSG_FLV_TAG, Param1: metaDataPkt}
				err = rtmppuller.src.ProcessMessage(msg)
				if err != nil {
					logger.LOGE(err.Error())
					rtmppuller.Stop(nil)
				}
			}
			rtmppuller.metaDatas = list.New()
		}
		msg := &wssAPI.Msg{}
		msg.Type = wssAPI.MSG_FLV_TAG
		msg.Param1 = pkt.ToFLVTag()
		err = rtmppuller.src.ProcessMessage(msg)
		if err != nil {
			logger.LOGE(err.Error())
			rtmppuller.Stop(nil)
		}
		return
	}
	logger.LOGE("bad status")

	return
}

func (rtmppuller *RTMPPuller) processAggregation(pkt *RTMPPacket) (err error) {
	cur := 0
	firstAggTime := uint32(0xffffffff)
	for cur < len(pkt.Body) {
		flvPkt := &flv.FlvTag{}
		flvPkt.StreamID = 0
		flvPkt.Timestamp = 0
		flvPkt.TagType = pkt.Body[cur]
		pktLength, _ := AMF0DecodeInt24(pkt.Body[cur+1 : cur+4])
		TimeStamp, _ := AMF0DecodeInt24(pkt.Body[cur+4 : cur+7])
		TimeStampExtended := uint32(pkt.Body[7])
		TimeStamp |= (TimeStampExtended << 24)
		if 0xffffffff == firstAggTime {
			firstAggTime = TimeStamp
		}
		flvPkt.Timestamp = pkt.TimeStamp + TimeStamp - firstAggTime
		flvPkt.Data = make([]byte, pktLength)

		copy(flvPkt.Data, pkt.Body[cur+11:cur+11+int(pktLength)])
		cur += 11 + int(pktLength) + 4
		msg := &wssAPI.Msg{}
		msg.Type = wssAPI.MSG_FLV_TAG
		msg.Param1 = flvPkt
		err = rtmppuller.src.ProcessMessage(msg)
		if err != nil {
			logger.LOGE(fmt.Sprintf("send aggregation pkts failed"))
			return
		}
	}

	return
}

func (rtmppuller *RTMPPuller) readRTMPPkt() (packet *RTMPPacket, err error) {
	err = rtmppuller.rtmp.Conn.SetReadDeadline(time.Now().Add(time.Duration(serviceConfig.TimeoutSec) * time.Second))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	defer rtmppuller.rtmp.Conn.SetReadDeadline(time.Time{})
	packet, err = rtmppuller.rtmp.ReadPacket()
	return
}

func (rtmppuller *RTMPPuller) play() (err error) {
	//connect
	err = rtmppuller.rtmp.Connect(false)
	if err != nil {
		logger.LOGE("rtmp connect to play failed")
		return
	}
	return
}

func (rtmppuller *RTMPPuller) getProp(pkt *RTMPPacket) (amfobj *AMF0Object, err error) {
	if RTMP_PACKET_TYPE_FLEX_MESSAGE == pkt.MessageTypeID {
		amfobj, err = AMF0DecodeObj(pkt.Body[1:])
	} else {
		amfobj, err = AMF0DecodeObj(pkt.Body)
	}
	if err != nil {
		logger.LOGE("recved invalid amf0 object")
		return
	}
	if amfobj.Props.Len() == 0 {
		logger.LOGT(pkt.Body)
		logger.LOGT(string(pkt.Body))
		return nil, errors.New("no props")
	}
	return
}

func (rtmppuller *RTMPPuller) handleInvoke(pkt *RTMPPacket) (err error) {

	amfobj, err := rtmppuller.getProp(pkt)
	if err != nil || amfobj == nil {
		logger.LOGE("decode amf failed")
		return
	}
	methodProp := amfobj.Props.Front().Value.(*AMF0Property)
	logger.LOGD(methodProp.Value.StrValue)
	switch methodProp.Value.StrValue {

	case "_result":
		err = rtmppuller.handleRTMPResult(amfobj)
		logger.LOGD(pkt.Body, pkt.MessageTypeID)
	case "onBWDone":
		rtmppuller.rtmp.SendCheckBW()
		rtmppuller.rtmp.SendReleaseStream()
		rtmppuller.rtmp.SendFCPublish()
	case "_onbwcheck":
		err = rtmppuller.rtmp.SendCheckBWResult(amfobj.AMF0GetPropByIndex(1).Value.NumValue)
	case "onFCPublish":
	case "_error":
	case "_onbwdone":
	case "onStatus":
		code := ""
		level := ""
		desc := ""
		if amfobj.Props.Len() >= 4 {
			objStatus := amfobj.AMF0GetPropByIndex(3).Value.ObjValue
			for e := objStatus.Props.Front(); e != nil; e = e.Next() {
				prop := e.Value.(*AMF0Property)
				switch prop.Name {
				case "code":
					code = prop.Value.StrValue
				case "level":
					level = prop.Value.StrValue
				case "description":
					desc = prop.Value.StrValue
				}
			}
		}
		logger.LOGT(level)
		logger.LOGT(desc)
		//close
		if code == "NetStream.Failed" || code == "NetStream.Play.Failed" ||
			code == "NetStream.Play.StreamNotFound" || code == "NetConnection.Connect.InvalidApp" ||
			code == "NetStream.Publish.Rejected" || code == "NetStream.Publish.Denied" ||
			code == "NetConnection.Connect.Rejected" {
			return errors.New("error and close")
		}
		//stop play
		if code == "NetStream.Play.Complete" || code == "NetStream.Play.Stop" ||
			code == "NetStream.Play.UnpublishNotify" {
			return errors.New("stop play")
		}
		//start play
		if code == "NetStream.Play.Start" || code == "NetStream.Play.PublishNotify" {
			//rtmppuller.CreatePlaySRC()
			logger.LOGW("start by media data")
		}
	default:
		logger.LOGW(fmt.Sprintf("method %s not processed", methodProp.Value.StrValue))
		amfobj.Dump()
		logger.LOGD(pkt.Body, pkt.MessageTypeID)
	}

	return
}

func (rtmppuller *RTMPPuller) handleRTMPResult(amfobj *AMF0Object) (err error) {
	idx := int32(amfobj.AMF0GetPropByIndex(1).Value.NumValue)
	rtmppuller.rtmp.mutexMethod.Lock()
	methodRet, ok := rtmppuller.rtmp.methodCache[idx]
	if ok == true {
		delete(rtmppuller.rtmp.methodCache, idx)
	}
	rtmppuller.rtmp.mutexMethod.Unlock()
	if !ok {
		logger.LOGW("method not found")
		return
	}
	switch methodRet {
	case "connect":
		err = rtmppuller.rtmp.AcknowledgementBW()
		if err != nil {
			logger.LOGE("acknowledgementBW failed")
			return
		}
		err = rtmppuller.rtmp.CreateStream()
		if err != nil {
			logger.LOGE("createStream failed")
			return
		}
	case "createStream":
		rtmppuller.rtmp.StreamID = uint32(amfobj.AMF0GetPropByIndex(3).Value.NumValue)
		err = rtmppuller.rtmp.SendPlay()
		logger.LOGD("send play &&&")
		if err != nil {
			logger.LOGE("send play failed")
			return
		}
	case "releaseStream":
	case "FCPublish":
	default:
		logger.LOGE(fmt.Sprintf("%s result not processed", methodRet))
	}
	return
}

//CreatePlaySRC for stream
func (rtmppuller *RTMPPuller) CreatePlaySRC() {
	if rtmppuller.src == nil {
		taskGet := &eStreamerEvent.EveGetSource{}
		taskGet.StreamName = rtmppuller.pullParams.SourceName
		err := wssAPI.HandleTask(taskGet)
		if err != nil {
			logger.LOGE(err.Error())
		}
		logger.LOGD(rtmppuller.pullParams.Address)
		if wssAPI.InterfaceValid(taskGet.SrcObj) && taskGet.HasProducer {
			//已经被其他人抢先了
			logger.LOGD("some other pulled rtmppuller stream:"+taskGet.StreamName, rtmppuller.pullParams.Address)
			logger.LOGD(taskGet.HasProducer)
			if rtmppuller.chValid {
				rtmppuller.pullParams.Src <- taskGet.SrcObj
			}
			rtmppuller.srcID = 0
			rtmppuller.reading = false
			return
		}
		taskAdd := &eStreamerEvent.EveAddSource{}
		taskAdd.Producer = rtmppuller
		taskAdd.StreamName = rtmppuller.pullParams.SourceName
		taskAdd.RemoteIp = rtmppuller.rtmp.Conn.RemoteAddr()
		err = wssAPI.HandleTask(taskAdd)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		if wssAPI.InterfaceIsNil(taskAdd.SrcObj) {
			rtmppuller.closeCh()
			rtmppuller.reading = false
			return
		}
		rtmppuller.src = taskAdd.SrcObj
		rtmppuller.srcID = taskAdd.Id
		rtmppuller.pullParams.Src <- rtmppuller.src
		go rtmppuller.checkPlayerCounts()
		logger.LOGT("add src ok..")
		return
	}
}

func (rtmppuller *RTMPPuller) checkPlayerCounts() {
	for rtmppuller.reading && wssAPI.InterfaceValid(rtmppuller.src) {
		time.Sleep(time.Duration(2) * time.Minute)
		eve := &eLiveListCtrl.EveGetLivePlayerCount{LiveName: rtmppuller.pullParams.SourceName}

		err := wssAPI.HandleTask(eve)
		if err != nil {
			logger.LOGD(err.Error())
			continue
		}
		if 1 > eve.Count {
			logger.LOGI("no player for rtmppuller rtmppuller ,close itself")
			rtmppuller.reading = false
			return
		}
	}
}
