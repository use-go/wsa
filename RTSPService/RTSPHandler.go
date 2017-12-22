package RTSPService

import (
	"container/list"
	"errors"
	"net"

	"github.com/use-go/websocket-streamserver/logger"

	"github.com/use-go/websocket-streamserver/events/eStreamerEvent"
	"github.com/use-go/websocket-streamserver/mediaTypes/amf"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/mediaTypes/h264"
	//	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/use-go/websocket-streamserver/wssAPI"
)

type RTSPHandler struct {
	conn        net.Conn
	mutexConn   sync.Mutex
	session     string
	streamName  string
	sinkAdded   bool
	sinkRunning bool
	audioHeader *flv.FlvTag
	videoHeader *flv.FlvTag
	isPlaying   bool
	waitPlaying *sync.WaitGroup
	videoCache  *list.List
	mutexVideo  sync.RWMutex
	audioCache  *list.List
	mutexAudio  sync.RWMutex
	tracks      map[string]*trackInfo
	mutexTracks sync.RWMutex
	tcpTimeout  bool //just for vlc(live555) no heart beat
}

type trackInfo struct {
	unicast      bool
	transPort    string
	firstSeq     int
	seq          int
	mark         bool
	RTPStartTime uint32
	trackID      string
	ssrc         uint32
	clockRate    uint32
	byteSend     int64
	pktSend      int64
	RTPChannel   int
	RTPCliPort   int
	RTPSvrPort   int
	RTCPChannel  int
	RTCPCliPort  int
	RTCPSvrPort  int
	RTPCliConn   *net.UDPConn //用来向客户端发送数据
	RTCPCliConn  *net.UDPConn //
	RTPSvrConn   *net.UDPConn //接收客户端的数据
	RTCPSvrConn  *net.UDPConn //
}

func (trackinfo *trackInfo) reset() {
	trackinfo.unicast = false
	trackinfo.transPort = "udp"
	trackinfo.firstSeq = 0
	trackinfo.seq = 0
	trackinfo.RTPStartTime = 0
	trackinfo.trackID = ""
	trackinfo.ssrc = 0
	trackinfo.clockRate = 0
	trackinfo.byteSend = 0
	trackinfo.pktSend = 0
	trackinfo.RTPChannel = 0
	trackinfo.RTPCliPort = 0
	trackinfo.RTPSvrPort = 0
	trackinfo.RTCPChannel = 0
	trackinfo.RTCPCliPort = 0
	trackinfo.RTCPSvrPort = 0
	if nil != trackinfo.RTPCliConn {
		trackinfo.RTPCliConn.Close()
	}
	if nil != trackinfo.RTCPCliConn {
		trackinfo.RTCPCliConn.Close()
	}
	if nil != trackinfo.RTPSvrConn {
		trackinfo.RTPSvrConn.Close()

	}
	if nil != trackinfo.RTCPSvrConn {
		trackinfo.RTCPSvrConn.Close()
	}
}

func (rtspHandler *RTSPHandler) Init(msg *wssAPI.Msg) (err error) {
	rtspHandler.session = wssAPI.GenerateGUID()
	rtspHandler.sinkAdded = false
	rtspHandler.tracks = make(map[string]*trackInfo)
	rtspHandler.waitPlaying = new(sync.WaitGroup)
	rtspHandler.tcpTimeout = true
	return
}

func (rtspHandler *RTSPHandler) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (rtspHandler *RTSPHandler) Stop(msg *wssAPI.Msg) (err error) {
	rtspHandler.isPlaying = false
	rtspHandler.waitPlaying.Wait()
	rtspHandler.delSink()
	return
}

func (rtspHandler *RTSPHandler) GetType() string {
	return "RTSPHandler"
}

func (rtspHandler *RTSPHandler) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (rtspHandler *RTSPHandler) ProcessMessage(msg *wssAPI.Msg) (err error) {
	switch msg.Type {
	case wssAPI.MsgFlvTag:
		return rtspHandler.appendFlvTag(msg)
	case wssAPI.MsgPlayStart:
		//这个状态下，rtsp 可以play
		rtspHandler.sinkRunning = true
	case wssAPI.MsgPlayStop:
		//如果在play,停止
		rtspHandler.sinkRunning = false
	default:
		logger.LOGE("msg not processed")
	}
	return
}

func (rtspHandler *RTSPHandler) appendFlvTag(msg *wssAPI.Msg) (err error) {
	tag := msg.Param1.(*flv.FlvTag)

	if rtspHandler.audioHeader == nil && tag.TagType == flv.FlvTagAudio {
		rtspHandler.audioHeader = tag.Copy()
		return
	}
	if rtspHandler.videoHeader == nil && tag.TagType == flv.FlvTagVideo {
		rtspHandler.videoHeader = tag.Copy()
		return
	}

	if tag.TagType == flv.FlvTagVideo {
		rtspHandler.mutexVideo.Lock()
		if rtspHandler.videoCache == nil || rtspHandler.videoCache.Len() > 0xff {
			rtspHandler.videoCache = list.New()
		}
		rtspHandler.videoCache.PushBack(tag.Copy())
		rtspHandler.mutexVideo.Unlock()
	}

	if flv.FlvTagAudio == tag.TagType {
		rtspHandler.mutexAudio.Lock()
		if rtspHandler.audioCache == nil || rtspHandler.audioCache.Len() > 0xff {
			rtspHandler.audioCache = list.New()
		}
		rtspHandler.audioCache.PushBack(tag.Copy())
		rtspHandler.mutexAudio.Unlock()
	}

	return
}

func (rtspHandler *RTSPHandler) handlePacket(data []byte) (err error) {
	//连接关闭
	if nil == data {
		return rtspHandler.Stop(nil)
	}
	if '$' == data[0] {
		return rtspHandler.handleRTPRTCP(data[1:])
	}

	return rtspHandler.handleRTSP(data)
}

func (rtspHandler *RTSPHandler) handleRTPRTCP(data []byte) (err error) {
	logger.LOGT("RTP RTCP not processed")
	return
}

func (rtspHandler *RTSPHandler) handleRTSP(data []byte) (err error) {
	lines := strings.Split(string(data), RTSP_EL)

	//取出方法
	if len(lines) < 1 {
		err = errors.New("rtsp cmd invalid")
		logger.LOGE(err.Error())
		return err
	}
	cmd := ""
	{
		strs := strings.Split(lines[0], " ")
		if len(strs) < 1 {
			logger.LOGE("invalid cmd")
			return errors.New("invalid cmd")
		}
		cmd = strs[0]
	}
	//处理每个方法
	switch cmd {
	case RTSP_METHOD_OPTIONS:
		return rtspHandler.serveOptions(lines)
	case RTSP_METHOD_DESCRIBE:
		return rtspHandler.serveDescribe(lines)
	case RTSP_METHOD_SETUP:
		return rtspHandler.serveSetup(lines)
	case RTSP_METHOD_PLAY:
		return rtspHandler.servePlay(lines)
	case RTSP_METHOD_PAUSE:
		return rtspHandler.servePause(lines)
	default:
		logger.LOGE("method " + cmd + " not support now")
		return rtspHandler.sendErrorReply(lines, 551)
	}
	return
}

func (rtspHandler *RTSPHandler) send(data []byte) (err error) {
	rtspHandler.mutexConn.Lock()
	defer rtspHandler.mutexConn.Unlock()
	_, err = wssAPI.TCPWriteTimeOut(rtspHandler.conn, data, serviceConfig.TimeoutSec)
	return
}

func (rtspHandler *RTSPHandler) addSink() bool {
	if true == rtspHandler.sinkAdded {
		logger.LOGE("sink not deleted")
		return false
	}
	taskAddSink := &eStreamerEvent.EveAddSink{}
	taskAddSink.StreamName = rtspHandler.streamName
	taskAddSink.SinkId = rtspHandler.session
	taskAddSink.Sinker = rtspHandler
	err := wssAPI.HandleTask(taskAddSink)
	if err != nil {
		logger.LOGE(err.Error())
		return false
	}
	rtspHandler.sinkAdded = true
	return true
}

func (rtspHandler *RTSPHandler) delSink() {
	if false == rtspHandler.sinkAdded {
		return
	}
	taskDelSink := &eStreamerEvent.EveDelSink{}
	taskDelSink.StreamName = rtspHandler.streamName
	taskDelSink.SinkId = rtspHandler.session
	wssAPI.HandleTask(taskDelSink)
	rtspHandler.sinkAdded = false
}

func (rtspHandler *RTSPHandler) threadPlay() {
	rtspHandler.isPlaying = true
	rtspHandler.mutexTracks.RLock()
	defer rtspHandler.mutexTracks.RUnlock()
	rtspHandler.waitPlaying.Add(1)
	defer func() {
		logger.LOGT("set play to false")
		rtspHandler.isPlaying = false
		for _, v := range rtspHandler.tracks {
			v.reset()
		}
		rtspHandler.waitPlaying.Done()
	}()
	chList := list.New()
	for _, v := range rtspHandler.tracks {
		ch := make(chan int)
		if v.transPort == "udp" {
			go rtspHandler.threadUdp(ch, v)
		} else {
			go rtspHandler.threadTCP(ch, v)
		}
		chList.PushBack(ch)
	}

	for v := chList.Front(); v != nil; v = v.Next() {
		ch := v.Value.(chan int)
		logger.LOGT(<-ch)
	}
	logger.LOGT("all ch end")
}

func (rtspHandler *RTSPHandler) threadUdp(ch chan int, track *trackInfo) {
	waitRTP := new(sync.WaitGroup)
	waitRTCP := new(sync.WaitGroup)
	defer func() {
		waitRTP.Wait()
		waitRTCP.Wait()
		close(ch)
		logger.LOGD("thread udp end")
	}()
	//监听RTP和RTCP服务端端口，等待穿透数据
	waitRTP.Add(1)
	waitRTCP.Add(1)
	chRtpCli := make(chan *net.UDPAddr)
	chRtcpCli := make(chan *net.UDPAddr)
	defer func() {
		close(chRtpCli)
		close(chRtcpCli)
	}()
	go listenRTP(track, rtspHandler, waitRTP, chRtpCli)
	go listenRTCP(track, rtspHandler, waitRTCP, chRtcpCli)
	//rtp
	select {
	case addr := <-chRtpCli:
		var err error
		track.RTPCliConn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer track.RTPCliConn.Close()
	case <-time.After(5 * time.Second):
		logger.LOGE("no rtp net data recved")
		return
	}
	//rtcp
	select {
	case addr := <-chRtcpCli:
		var err error
		track.RTCPCliConn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer track.RTCPCliConn.Close()
	case <-time.After(5 * time.Second):
		logger.LOGE("no rtcp net data recved")
		return
	}
	logger.LOGT("get cli conn")
	defer func() {
		//关闭连接，以触发关闭监听
		logger.LOGD("close rtp rtcp server conn")
		track.RTPSvrConn.Close()
		track.RTCPSvrConn.Close()
	}()
	//发送数据
	beginSend := false
	if track.trackID == ctrl_track_video {

		//清空之前累计的亢余数据
		rtspHandler.mutexVideo.Lock()
		rtspHandler.videoCache = list.New()
		rtspHandler.mutexVideo.Unlock()
	} else if track.trackID == ctrl_track_audio {
		rtspHandler.mutexAudio.Lock()
		rtspHandler.audioCache = list.New()
		rtspHandler.mutexAudio.Unlock()
	}
	beginTime := uint32(0)
	audioBeginTime := uint32(0)
	logger.LOGT(track.trackID)
	track.seq = track.firstSeq
	lastRTCPTime := time.Now().Second()
	for rtspHandler.isPlaying {
		if time.Now().Second()-lastRTCPTime > 10 {
			lastRTCPTime = time.Now().Second()

			rtspHandler.mutexVideo.Lock()
			if rtspHandler.videoCache == nil || rtspHandler.videoCache.Len() == 0 {
				rtspHandler.mutexVideo.Unlock()
				time.Sleep(10 * time.Millisecond)
				continue
			}
			tag := rtspHandler.videoCache.Front().Value.(*flv.FlvTag).Copy()
			rtspHandler.mutexVideo.Unlock()
			timestamp := tag.Timestamp - beginTime
			if tag.Timestamp < beginTime {
				timestamp = 0
			}
			tmp64 := int64(track.clockRate / 1000)
			timestamp = uint32((tmp64 * int64(timestamp)) & 0xffffffff)
			rtspHandler.sendRTCP(track, timestamp)
		}
		//如果是视频 等到有关键帧时开始发送，如果是音频，直接发送
		if ctrl_track_video == track.trackID {
			if false == beginSend {
				//等待关键帧
				beginSend, beginTime = getH264Keyframe(rtspHandler.videoCache, rtspHandler.mutexVideo)
				if false == beginSend {
					time.Sleep(30 * time.Millisecond)
					continue
				}
				//把头加回去
				//				if rtspHandler.videoHeader != nil {
				//					rtspHandler.mutexVideo.Lock()
				//					rtspHandler.videoCache.PushFront(rtspHandler.videoHeader)
				//					rtspHandler.mutexVideo.Unlock()
				//				}
			}
			//发送数据
			rtspHandler.mutexVideo.Lock()
			if rtspHandler.videoCache == nil || rtspHandler.videoCache.Len() == 0 {
				rtspHandler.mutexVideo.Unlock()
				time.Sleep(10 * time.Millisecond)
				continue
			}
			tag := rtspHandler.videoCache.Front().Value.(*flv.FlvTag).Copy()
			rtspHandler.videoCache.Remove(rtspHandler.videoCache.Front())
			rtspHandler.mutexVideo.Unlock()
			err := rtspHandler.sendFlvH264(track, tag, beginTime)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		} else if ctrl_track_audio == track.trackID {
			//			logger.LOGT("audio not processed now")
			rtspHandler.mutexAudio.Lock()
			if rtspHandler.audioCache == nil || rtspHandler.audioCache.Len() == 0 {
				rtspHandler.mutexAudio.Unlock()
				time.Sleep(10 * time.Millisecond)
				continue
			}
			tag := rtspHandler.audioCache.Front().Value.(*flv.FlvTag).Copy()
			rtspHandler.audioCache.Remove(rtspHandler.audioCache.Front())
			rtspHandler.mutexAudio.Unlock()
			if audioBeginTime == 0 {
				audioBeginTime = tag.Timestamp
			}
			if false == beginSend {
				beginSend = true
				err := rtspHandler.sendFlvAudio(track, rtspHandler.audioHeader, audioBeginTime)
				if err != nil {
					logger.LOGE(err.Error())
					return
				}
			}
			err := rtspHandler.sendFlvAudio(track, tag, audioBeginTime)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		} else {
			logger.LOGE(track.trackID + " wrong")
			return
		}

	}
}

func (rtspHandler *RTSPHandler) threadTCP(ch chan int, track *trackInfo) {
	defer func() {
		close(ch)
	}()
	beginSend := true
	if track.trackID == ctrl_track_video {
		beginSend = false

		//清空之前累计的亢余数据
		rtspHandler.mutexVideo.Lock()
		rtspHandler.videoCache = list.New()
		rtspHandler.mutexVideo.Unlock()
	} else if track.trackID == ctrl_track_audio {
		rtspHandler.mutexAudio.Lock()
		rtspHandler.audioCache = list.New()
		rtspHandler.mutexAudio.Unlock()
	}
	beginTime := uint32(0)
	audioBeginTime := uint32(0)
	logger.LOGT(track.trackID)
	track.seq = track.firstSeq
	for rtspHandler.isPlaying {
		if ctrl_track_video == track.trackID {
			if false == beginSend {
				//等待关键帧
				beginSend, beginTime = getH264Keyframe(rtspHandler.videoCache, rtspHandler.mutexVideo)
				if false == beginSend {
					time.Sleep(30 * time.Millisecond)
					continue
				}
			}
			//发送数据
			rtspHandler.mutexVideo.Lock()
			if rtspHandler.videoCache == nil || rtspHandler.videoCache.Len() == 0 {
				rtspHandler.mutexVideo.Unlock()
				time.Sleep(10 * time.Millisecond)
				continue
			}
			tag := rtspHandler.videoCache.Front().Value.(*flv.FlvTag).Copy()
			rtspHandler.videoCache.Remove(rtspHandler.videoCache.Front())
			rtspHandler.mutexVideo.Unlock()
			err := rtspHandler.sendFlvH264(track, tag, beginTime)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		} else if ctrl_track_audio == track.trackID {
			//			logger.LOGT("audio not processed now")
			rtspHandler.mutexAudio.Lock()
			if rtspHandler.audioCache == nil || rtspHandler.audioCache.Len() == 0 {
				rtspHandler.mutexAudio.Unlock()
				time.Sleep(10 * time.Millisecond)
				continue
			}
			tag := rtspHandler.audioCache.Front().Value.(*flv.FlvTag).Copy()
			rtspHandler.audioCache.Remove(rtspHandler.audioCache.Front())
			rtspHandler.mutexAudio.Unlock()
			if audioBeginTime == 0 {
				audioBeginTime = tag.Timestamp
			}
			if false == beginSend {
				beginSend = true
				err := rtspHandler.sendFlvAudio(track, rtspHandler.audioHeader, audioBeginTime)
				if err != nil {
					logger.LOGE(err.Error())
					return
				}
			}
			err := rtspHandler.sendFlvAudio(track, tag, audioBeginTime)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}

		} else {
			logger.LOGE(track.trackID + " wrong")
			return
		}
	}
}

func (rtspHandler *RTSPHandler) sendFlvH264(track *trackInfo, tag *flv.FlvTag, beginSend uint32) (err error) {
	if "udp" == track.transPort {
		pkts := rtspHandler.generateH264RTPPackets(tag, beginSend, track)
		if nil == pkts {
			return
		}

		for v := pkts.Front(); v != nil; v = v.Next() {
			//data := v.Value.([]byte)
			//logger.LOGF(data)
			_, err = track.RTPCliConn.Write(v.Value.([]byte))
			track.pktSend++
			track.byteSend += int64(len(v.Value.([]byte)))
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		}
	} else {
		pkts := rtspHandler.generateH264RTPPackets(tag, beginSend, track)
		if nil == pkts {
			return
		}

		for v := pkts.Front(); v != nil; v = v.Next() {
			pktData := v.Value.([]byte)
			dataSize := len(pktData)
			{
				tcpHeader := make([]byte, 4)
				tcpHeader[0] = '$'
				tcpHeader[1] = byte(track.RTPChannel)
				tcpHeader[2] = byte(dataSize >> 8)
				tcpHeader[3] = byte(dataSize & 0xff)
				track.byteSend += 4
				rtspHandler.send(tcpHeader)
			}

			err = rtspHandler.send(pktData)
			track.pktSend++
			track.byteSend += int64(dataSize)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		}
	}
	return
}

func (rtspHandler *RTSPHandler) sendFlvAudio(track *trackInfo, tag *flv.FlvTag, beginSend uint32) (err error) {
	var pkts *list.List
	switch tag.Data[0] >> 4 {
	case flv.SoundFormatAAC:
		pkts = rtspHandler.generateAACRTPPackets(tag, beginSend, track)
	case flv.SoundFormatMP3:
		pkts = rtspHandler.generateMP3RTPPackets(tag, beginSend, track)
	default:
		logger.LOGW("audio type not support now")
		return
	}
	//send data
	if pkts == nil {
		return
	}

	for v := pkts.Front(); v != nil; v = v.Next() {

		if "udp" == track.transPort {
			_, err = track.RTPCliConn.Write(v.Value.([]byte))
			track.pktSend++
			track.byteSend += int64(len(v.Value.([]byte)))
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		} else {
			pktData := v.Value.([]byte)
			dataSize := len(pktData)
			{
				tcpHeader := make([]byte, 4)
				tcpHeader[0] = '$'
				tcpHeader[1] = byte(track.RTPChannel)
				tcpHeader[2] = byte(dataSize >> 8)
				tcpHeader[3] = byte(dataSize & 0xff)
				track.byteSend += 4
				rtspHandler.send(tcpHeader)
			}

			err = rtspHandler.send(pktData)
			track.pktSend++
			track.byteSend += int64(dataSize)
			if err != nil {
				logger.LOGE(err.Error())
				return
			}
		}
	}
	return
}

func (rtspHandler *RTSPHandler) sendRTCP(track *trackInfo, rtpTime uint32) {
	if "udp" == track.transPort {
		data := createSR(track.ssrc, rtpTime, uint32(track.pktSend), uint32(track.byteSend))
		_, err := track.RTCPCliConn.Write(data)
		if err != nil {
			logger.LOGE(err.Error())
		}
	} else {

	}
}

//packetization-mode 1
func (rtspHandler *RTSPHandler) generateH264RTPPackets(tag *flv.FlvTag, beginTime uint32, track *trackInfo) (rtpPkts *list.List) {
	payLoadSize := RTP_MTU
	if track.transPort == "tcp" {
		payLoadSize -= 4
	}
	//忽略AVC

	if tag.Data[0] == 0x17 && tag.Data[1] == 0x0 {
		return
	}
	//frame类型：1  avc type:1 compositionTime:3 nalsize:4
	//可能有多个nal
	cur := 5
	rtpPkts = list.New()
	for cur < len(tag.Data) {
		nalSize, _ := amf.AMF0DecodeInt32(tag.Data[cur:])
		cur += 4
		nalData := tag.Data[cur : cur+int(nalSize)]
		cur += int(nalSize)
		nalType := nalData[0] & 0xf
		//忽略sps pps
		if nalType == h264.NalType_sps || nalType == h264.NalType_pps {
			continue
		}

		//计算RTP时间
		timestamp := tag.Timestamp - beginTime
		if tag.Timestamp < beginTime {
			timestamp = 0
		}
		//rtptime/timestamp=rate/1000  rtptime=rate*timestamp/1000
		tmp64 := int64(track.clockRate / 1000)
		timestamp = uint32((tmp64 * int64(timestamp)) & 0xffffffff)
		//关键帧前面加sps pps
		if nalType == h264.NalType_idr {
			sps, pps := h264.GetSpsPpsFromAVC(rtspHandler.videoHeader.Data[5:])

			stapA := &amf.AMF0Encoder{}
			stapA.Init()
			{
				//header
				track.seq++
				if track.seq > 0xffff {
					track.seq = 0
				}
				headerData := createRTPHeader(Payload_h264, uint32(track.seq), timestamp, track.ssrc)
				stapA.AppendByteArray(headerData)
				//start bit
				Nri := ((sps[0] & 0x60) >> 5)
				Type := byte(NalType_STAP_A)
				stapA.AppendByte(((Nri << 5) | Type))
				//sps size
				stapA.EncodeInt16(int16(len(sps)))
				//sps
				stapA.AppendByteArray(sps)
				//pps size
				stapA.EncodeInt16(int16(len(pps)))
				//pps
				stapA.AppendByteArray(pps)
				pktData, _ := stapA.GetData()
				rtpPkts.PushBack(pktData)
			}
		}
		//帧数据
		{
			payLoadSize -= 13 //12 rtp header,1 f nri type
			//单一包
			if nalSize < uint32(payLoadSize) {
				single := &amf.AMF0Encoder{}
				single.Init()
				track.seq++
				if track.seq > 0xffff {
					track.seq = 0
				}
				headerData := createRTPHeader(Payload_h264, uint32(track.seq), timestamp, track.ssrc)
				single.AppendByteArray(headerData)
				single.AppendByteArray(nalData)
				pktData, _ := single.GetData()
				rtpPkts.PushBack(pktData)
			} else {
				//分片包,使用FU_A包
				payLoadSize-- //FU header
				var FU_S, FU_E, FU_R, FU_Type, Nri, Type byte
				Nri = ((nalData[0] & 0x60) >> 5)
				Type = NalType_FU_A
				FU_Type = nalType
				count := int(nalSize) / payLoadSize
				if count*payLoadSize < int(nalSize) {
					count++
				}
				//first frame
				curFrame := 0
				curNalData := 1 //nal 的第一个字节的帧类型信息放到fh_header 里面
				{
					FU_S = 1
					FU_E = 0
					FU_R = 0
					fua := amf.AMF0Encoder{}
					fua.Init()
					track.seq++
					if track.seq > 0xffff {
						track.seq = 0
					}
					headerData := createRTPHeader(Payload_h264, uint32(track.seq), timestamp, track.ssrc)
					fua.AppendByteArray(headerData)
					fua.AppendByte((Nri << 5) | Type)
					fua.AppendByte((FU_S << 7) | (FU_E << 6) | (FU_R << 5) | FU_Type)
					fua.AppendByteArray(nalData[curNalData : payLoadSize+curNalData])
					curNalData += payLoadSize
					pktData, _ := fua.GetData()
					rtpPkts.PushBack(pktData)
					curFrame++
				}

				//mid frame
				{
					FU_S = 0
					FU_E = 0
					FU_R = 0
					for curFrame+1 < count {
						track.seq++
						if track.seq > 0xffff {
							track.seq = 0
						}
						fua := amf.AMF0Encoder{}
						fua.Init()
						headerData := createRTPHeader(Payload_h264, uint32(track.seq), timestamp, track.ssrc)
						fua.AppendByteArray(headerData)
						fua.AppendByte((Nri << 5) | Type)
						fua.AppendByte((FU_S << 7) | (FU_E << 6) | (FU_R << 5) | FU_Type)
						fua.AppendByteArray(nalData[curNalData : payLoadSize+curNalData])
						curNalData += payLoadSize
						pktData, _ := fua.GetData()
						rtpPkts.PushBack(pktData)
						curFrame++
					}
				}
				//last frame
				lastFrameSize := int(nalSize) - curNalData
				if lastFrameSize > 0 {
					FU_S = 0
					FU_E = 1
					FU_R = 0
					track.seq++
					if track.seq > 0xffff {
						track.seq = 0
					}
					fua := amf.AMF0Encoder{}
					fua.Init()
					headerData := createRTPHeader(Payload_h264, uint32(track.seq), timestamp, track.ssrc)
					fua.AppendByteArray(headerData)
					fua.AppendByte((Nri << 5) | Type)
					fua.AppendByte((FU_S << 7) | (FU_E << 6) | (FU_R << 5) | FU_Type)
					fua.AppendByteArray(nalData[curNalData:])
					curNalData += payLoadSize
					pktData, _ := fua.GetData()
					rtpPkts.PushBack(pktData)
					curFrame++
				}
			}
		}
	}

	return
}

func (rtspHandler *RTSPHandler) stopPlayThread() {
	rtspHandler.isPlaying = false
	rtspHandler.waitPlaying.Wait()
}

/*
2byte au-header-length

*/
func (rtspHandler *RTSPHandler) generateAACRTPPackets(tag *flv.FlvTag, beginTime uint32, track *trackInfo) (rtpPkts *list.List) {
	payloadSize := RTP_MTU - 4
	if "tcp" == track.transPort {
		payloadSize -= 4
	}

	dataSize := len(tag.Data) - 2
	if dataSize < 4 {
		return
	}

	aHeader := make([]byte, 4)
	aHeader[0] = 0x00
	aHeader[1] = 0x10
	aHeader[2] = byte((dataSize & 0x1fe0) >> 5)
	aHeader[3] = byte((dataSize & 0x1f) << 3)
	timestamp := tag.Timestamp - beginTime
	if tag.Timestamp < beginTime {
		timestamp = 0
	}
	//rtptime/timestamp=rate/1000  rtptime=rate*timestamp/1000
	tmp64 := int64(track.clockRate / 1000)
	timestamp = uint32((tmp64 * int64(timestamp)) & 0xffffffff)
	//12 rtp header
	payloadSize -= 12
	cur := 2

	rtpPkts = list.New()
	for dataSize > payloadSize {
		logger.LOGF("big aac")
		tmp := &amf.AMF0Encoder{}
		tmp.Init()
		headerData := createRTPHeaderAAC(Payload_h264, uint32(track.seq), timestamp, track.ssrc)

		track.seq++
		if track.seq > 0xffff {
			track.seq = 0
		}
		tmp.AppendByteArray(headerData)
		tmp.AppendByteArray(aHeader)
		tmp.AppendByteArray(tag.Data[cur : cur+payloadSize])
		pktData, _ := tmp.GetData()
		rtpPkts.PushBack(pktData)
		dataSize -= payloadSize
		cur += payloadSize
	}
	//剩下的数据
	if dataSize > 0 {
		single := &amf.AMF0Encoder{}
		single.Init()
		headerData := createRTPHeaderAAC(Payload_h264, uint32(track.seq), timestamp, track.ssrc)
		track.seq++
		if track.seq > 0xffff {
			track.seq = 0
		}
		single.AppendByteArray(headerData)
		single.AppendByteArray(aHeader)
		single.AppendByteArray(tag.Data[cur:])
		pktData, _ := single.GetData()
		rtpPkts.PushBack(pktData)
	}
	return
}
