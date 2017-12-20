package RTMPService

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"

	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/mediaTypes/flv"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

//RTMP Protocol Setting
const (
	RTMP_protocol_rtmp      = "rtmp"
	RTMP_default_chunk_size = 128
	RTMP_default_buff_ms    = 500
	RTMP_better_chunk_size  = 128

	RTMP_channel_control      = 0x02
	RTMP_channel_Invoke       = 0x03
	RTMP_channel_SendLive     = 0x04
	RTMP_channel_SendPlayback = 0x05
	RTMP_channel_AV           = 0x08
)

var bwCheckCounts float64

const (
	RTMP_PACKET_TYPE_CHUNK_SIZE         = 0x01
	RTMP_PACKET_TYPE_ABORT              = 0x02
	RTMP_PACKET_TYPE_BYTES_READ_REPORT  = 0x03
	RTMP_PACKET_TYPE_CONTROL            = 0x04
	RTMP_PACKET_TYPE_SERVER_BW          = 0x05
	RTMP_PACKET_TYPE_CLIENT_BW          = 0x06
	RTMP_PACKET_TYPE_AUDIO              = 0x08
	RTMP_PACKET_TYPE_VIDEO              = 0x09
	RTMP_PACKET_TYPE_FLEX_STREAM_SEND   = 0x0f //amf3
	RTMP_PACKET_TYPE_FLEX_SHARED_OBJECT = 0x10
	RTMP_PACKET_TYPE_FLEX_MESSAGE       = 0x11
	RTMP_PACKET_TYPE_INFO               = 0x12 //amf0
	RTMP_PACKET_TYPE_SHARED_OBJECT      = 0x13
	RTMP_PACKET_TYPE_INVOKE             = 0x14
	RTMP_PACKET_TYPE_FLASH_VIDEO        = 0x16
)

const (
	RTMP_CTRL_streamBegin       = 0
	RTMP_CTRL_streamEof         = 1
	RTMP_CTRL_streamDry         = 2
	RTMP_CTRL_setBufferLength   = 3
	RTMP_CTRL_streamIsRecorded  = 4
	RTMP_CTRL_pingRequest       = 6
	RTMP_CTRL_pingResponse      = 7
	RTMP_CTRL_streamBufferEmpty = 31
	RTMP_CTRL_streamBufferReady = 32
)

// RTMP HEADER TYPE
const (
	RTMP_HEADER_TYPE_0 = 0
	RTMP_HEADER_TYPE_1 = 1
	RTMP_HEADER_TYPE_2 = 2
	RTMP_HEADER_TYPE_3 = 3
)

//RTMP_LINK struct
type RTMP_LINK struct {
	Protocol    string
	App         string
	Flashver    string
	SwfUrl      string
	TcUrl       string
	Fpad        string
	AudioCodecs string
	VideoCodecs string
	Path        string
	Address     string
	SeekTime    int
	StopTime    int
}

//RTMPPacket struct
type RTMPPacket struct {
	ChunkStreamID   int32
	Fmt             byte
	TimeStamp       uint32 //fmt 0为绝对时间，1或2为相对时间
	MessageLength   uint32
	MessageTypeID   byte
	MessageStreamID uint32
	Body            []byte
	BodyReaded      int32
}

//RTMP Description
type RTMP struct {
	Conn                      net.Conn
	SendChunkSize             uint32
	RecvChunkSize             uint32
	NumInvokes                int32
	StreamID                  uint32
	Link                      RTMP_LINK
	AudioCodecs               int32
	VideoCodecs               int32
	TargetBW                  uint32
	AcknowledgementWindowSize uint32
	SelfBW                    uint32
	LimitType                 uint32
	BytesIn                   int64
	BytesInLast               int64
	buffMS                    uint32
	recvCache                 map[int32]*RTMPPacket
	methodCache               map[int32]string
	mutexMethod               sync.RWMutex
}

func (rtmp *RTMPPacket) Copy() (dst *RTMPPacket) {
	dst = &RTMPPacket{}
	dst.BodyReaded = rtmp.BodyReaded
	dst.Body = make([]byte, dst.BodyReaded)
	copy(dst.Body, rtmp.Body)
	dst.ChunkStreamID = rtmp.ChunkStreamID
	dst.Fmt = rtmp.Fmt
	dst.MessageLength = rtmp.MessageLength
	dst.MessageStreamID = rtmp.MessageStreamID
	dst.TimeStamp = rtmp.TimeStamp
	dst.MessageTypeID = rtmp.MessageTypeID
	return
}

func (rtmp *RTMPPacket) ToFLVTag() (dst *flv.FlvTag) {
	dst = &flv.FlvTag{}
	dst.TagType = rtmp.MessageTypeID
	dst.StreamID = 0
	dst.Timestamp = rtmp.TimeStamp
	//dst.Data = make([]byte, len(rtmp.Body))
	//copy(dst.Data, rtmp.Body)
	dst.Data = rtmp.Body
	return
}

func FlvTagToRTMPPacket(ta *flv.FlvTag) (dst *RTMPPacket) {
	dst = &RTMPPacket{}
	dst.ChunkStreamID = RTMP_channel_SendLive
	dst.Fmt = 0
	dst.TimeStamp = ta.Timestamp
	dst.MessageStreamID = 1
	dst.MessageTypeID = ta.TagType
	dst.MessageLength = uint32(len(ta.Data))
	//dst.Body = make([]byte, dst.MessageLength)
	//copy(dst.Body, ta.Data)
	dst.Body = ta.Data
	return
}

func (rtmp *RTMP) Init(conn net.Conn) {
	rtmp.Conn = conn
	rtmp.SendChunkSize = RTMP_default_chunk_size
	rtmp.RecvChunkSize = RTMP_default_chunk_size
	rtmp.AudioCodecs = 3191
	rtmp.VideoCodecs = 252
	rtmp.TargetBW = 2500000
	rtmp.AcknowledgementWindowSize = 0
	rtmp.SelfBW = 2500000
	rtmp.LimitType = 2
	rtmp.buffMS = RTMP_default_buff_ms
	rtmp.recvCache = make(map[int32]*RTMPPacket)
	rtmp.methodCache = make(map[int32]string)
}

func (rtmp *RTMP) ReadPacket() (packet *RTMPPacket, err error) {
	for {
		packet, err = rtmp.ReadChunk()
		if err != nil {
			logger.LOGT("read rtmp chunk failed")
			return
		}
		if nil != packet && packet.BodyReaded > 0 &&
			packet.BodyReaded == int32(len(packet.Body)) {
			return
		}
	}
	return
}

func timeAdd(src, delta uint32) (ret uint32) {
	ret = src + delta
	return
}

func (rtmp *RTMP) ReadChunk() (packet *RTMPPacket, err error) {
	//接收basic header
	buf, err := rtmp.rtmpSocketRead(1)
	if err != nil {
		return
	}
	var chunkfmt byte
	var chunkID int32
	chunkfmt = (buf[0] & 0xc0) >> 6
	chunkID = int32(buf[0] & 0x3f)
	if chunkID == 0 {
		buf, err = rtmp.rtmpSocketRead(1)
		if err != nil {
			return
		}
		chunkID = int32(buf[0]) + 64
	} else if chunkID == 1 {
		buf, err = rtmp.rtmpSocketRead(2)
		if err != nil {
			return
		}
		tmp16 := int32(buf[0]) + int32(buf[1])*256 + 64
		if err != nil {
			return nil, err
		}
		chunkID = int32(tmp16)
	}
	if rtmp.recvCache[chunkID] == nil {
		logger.LOGT("new chunkid " + strconv.Itoa(int(chunkID)))
		rtmp.recvCache[chunkID] = &RTMPPacket{}
		rtmp.recvCache[chunkID].Fmt = 0
	}
	rtmp.recvCache[chunkID].ChunkStreamID = chunkID
	//接收message header
	switch chunkfmt {
	case 0:
		buf, err := rtmp.rtmpSocketRead(11)
		if err != nil {
			return nil, err
		}
		tmpPkt := rtmp.recvCache[chunkID]
		tmpPkt.TimeStamp, _ = AMF0DecodeInt24(buf)
		tmpPkt.MessageLength, _ = AMF0DecodeInt24(buf[3:])
		tmpPkt.MessageTypeID = buf[6]
		tmpPkt.MessageStreamID, _ = AMF0DecodeInt32LE(buf[7:])
		if tmpPkt.TimeStamp == 0xffffff {
			buf, err = rtmp.rtmpSocketRead(4)
			if err != nil {
				return nil, err
			}
			tmpPkt.TimeStamp, _ = AMF0DecodeInt32(buf)
		}
	case 1:
		buf, err := rtmp.rtmpSocketRead(7)
		if err != nil {
			return nil, err
		}
		tmpPkt := rtmp.recvCache[chunkID]
		timeDelta, _ := AMF0DecodeInt24(buf)
		tmpPkt.MessageLength, _ = AMF0DecodeInt24(buf[3:])
		tmpPkt.MessageTypeID = buf[6]
		if timeDelta == 0xffffff {
			buf, err = rtmp.rtmpSocketRead(4)
			if err != nil {
				return nil, err
			}
			timeDelta, _ = AMF0DecodeInt32(buf)
		}
		tmpPkt.TimeStamp = timeAdd(tmpPkt.TimeStamp, timeDelta)
	case 2:
		buf, err := rtmp.rtmpSocketRead(3)

		if err != nil {
			return nil, err
		}

		tmpPkt := rtmp.recvCache[chunkID]
		timeDelta, _ := AMF0DecodeInt24(buf)
		if timeDelta == 0xffffff {
			buf, err = rtmp.rtmpSocketRead(4)
			if err != nil {
				return nil, err
			}
			timeDelta, _ = AMF0DecodeInt32(buf)
		}
		tmpPkt.TimeStamp = timeAdd(tmpPkt.TimeStamp, timeDelta)

	case 3:

	}
	//接收chunk data
	tmpPkt, ok := rtmp.recvCache[chunkID]
	if false == ok {
		logger.LOGF("ok")
	}
	if tmpPkt.MessageLength == 0 {
		logger.LOGE("why 0....message length")
		return
	}
	if chunkfmt != 3 {
		tmpPkt.Body = make([]byte, tmpPkt.MessageLength)
		tmpPkt.BodyReaded = 0
	} else {
		if tmpPkt.BodyReaded == int32(tmpPkt.MessageLength) {
			tmpPkt.BodyReaded = 0
		}
	}
	//接收小于等于一个chunksize的数据
	recvsize := tmpPkt.MessageLength - uint32(tmpPkt.BodyReaded)
	if recvsize > rtmp.RecvChunkSize {
		recvsize = rtmp.RecvChunkSize
	}
	tmpBody, err := rtmp.rtmpSocketRead(int(recvsize))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	if chunkfmt == 3 {
		if tmpPkt.BodyReaded == int32(tmpPkt.MessageLength) {
			logger.LOGT(fmt.Sprintf("%d %d", tmpPkt.BodyReaded, tmpPkt.MessageLength))
			logger.LOGE(tmpBody)
		}
	}
	copy(tmpPkt.Body[tmpPkt.BodyReaded:], tmpBody)
	tmpPkt.BodyReaded += int32(recvsize)
	//判断是否收到一个完整的包
	if tmpPkt.BodyReaded == int32(tmpPkt.MessageLength) {
		packet = tmpPkt.Copy()
	}
	return
}

func (rtmp *RTMP) rtmpSocketRead(size int) (data []byte, err error) {
	data, err = wssAPI.TCPRead(rtmp.Conn, size)
	if err != nil {
		return
	}
	if rtmp.AcknowledgementWindowSize > 0 {
		rtmp.BytesIn += int64(len(data))
		if rtmp.BytesIn > int64(rtmp.AcknowledgementWindowSize/10)+rtmp.BytesInLast {
			rtmp.BytesInLast = rtmp.BytesIn
			rtmp.sendAcknowledgement()
		}
	}
	return
}

func (rtmp *RTMP) SendPacket(packet *RTMPPacket, queue bool) (err error) {
	//基本头
	encoder := &AMF0Encoder{}
	encoder.Init()
	var tmp8 byte
	if packet.ChunkStreamID < 63 {
		tmp8 = (packet.Fmt << 6) | byte(packet.ChunkStreamID)
		encoder.AppendByte(tmp8)
	} else if packet.ChunkStreamID < 319 {
		tmp8 = (packet.Fmt << 6)
		encoder.AppendByte(tmp8)
		encoder.AppendByte(byte(packet.ChunkStreamID - 64))
	} else if packet.ChunkStreamID < 65599 {
		tmp8 = (packet.Fmt << 6) | byte(1)
		encoder.AppendByte(tmp8)
		encoder.AppendByte(byte(packet.ChunkStreamID-64) & 0xff)
		encoder.AppendByte(byte(packet.ChunkStreamID-64) >> 8)
	} else {
		return fmt.Errorf("chunk stream id:%d invalid", packet.ChunkStreamID)
	}
	//消息头
	if packet.Fmt <= 2 {
		//只有时间
		if packet.TimeStamp >= 0xffffff {
			encoder.EncodeInt24(0xffffff)
		} else {
			encoder.EncodeInt24(int32(packet.TimeStamp))
		}
	}
	if packet.Fmt <= 1 {
		//有长度，类型
		encoder.EncodeInt24(int32(packet.MessageLength))
		encoder.AppendByte(byte(packet.MessageTypeID))
	}
	if packet.Fmt == 0 {
		//有流id
		encoder.EncodeInt32LittleEndian(int32(packet.MessageStreamID))
	}
	//时间扩展
	if packet.TimeStamp >= 0xffffff {
		encoder.EncodeInt32(int32(packet.TimeStamp))
	}
	//消息
	var bodySended uint32
	bodySended = 0
	sendSize := packet.MessageLength
	if sendSize > rtmp.SendChunkSize {
		sendSize = rtmp.SendChunkSize
	}
	encoder.AppendByteArray(packet.Body[:sendSize])
	buf, err := encoder.GetData()
	if err != nil {
		return
	}

	_, err = wssAPI.TCPWrite(rtmp.Conn, buf)
	if err != nil {
		return
	}
	bodySended += sendSize
	//剩下的消息
	packet.Fmt = 3
	for bodySended < packet.MessageLength {
		encoder3 := &AMF0Encoder{}
		encoder3.Init()
		if packet.ChunkStreamID < 63 {
			tmp8 = (packet.Fmt << 6) | byte(packet.ChunkStreamID)
			encoder3.AppendByte(tmp8)
		} else if packet.ChunkStreamID < 319 {
			tmp8 = (packet.Fmt << 6)
			encoder3.AppendByte(tmp8)
			encoder3.AppendByte(byte(packet.ChunkStreamID - 64))
		} else if packet.ChunkStreamID < 65599 {
			tmp8 = (packet.Fmt << 6) | byte(1)
			encoder3.AppendByte(tmp8)
			encoder3.AppendByte(byte(packet.ChunkStreamID-64) & 0xff)
			encoder3.AppendByte(byte(packet.ChunkStreamID-64) >> 8)
		}
		sendSize = packet.MessageLength - bodySended
		if sendSize > rtmp.SendChunkSize {
			sendSize = rtmp.SendChunkSize
		}
		encoder3.AppendByteArray(packet.Body[bodySended : bodySended+sendSize])
		buf, err := encoder3.GetData()
		if err != nil {
			return err
		}
		_, err = wssAPI.TCPWrite(rtmp.Conn, buf)
		if err != nil {
			return err
		}
		bodySended += sendSize
	}
	if RTMP_PACKET_TYPE_INVOKE == packet.MessageTypeID && queue {
		cmdName, err := AMF0DecodeString(packet.Body[1:])
		if err != nil {
			return err
		}
		transactionID, err := AMF0DecodeNumber(packet.Body[4+len(cmdName):])
		if err != nil {
			return err
		}
		rtmp.mutexMethod.Lock()
		rtmp.methodCache[int32(transactionID)] = cmdName
		rtmp.mutexMethod.Unlock()
	}
	return
}

func (rtmp *RTMP) HandleControl(pkt *RTMPPacket) (err error) {
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
	case RTMP_CTRL_streamDry:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGT(fmt.Sprintf("stream dry:%d", streamID))
	case RTMP_CTRL_setBufferLength:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		buffMS, _ := AMF0DecodeInt32(pkt.Body[6:])
		rtmp.buffMS = uint32(buffMS)
		rtmp.StreamID = uint32(streamID)
		//logger.LOGI(fmt.Sprintf("set buffer length --streamid:%d--buffer length:%d", rtmp.StreamID, rtmp.buffMS))
	case RTMP_CTRL_streamIsRecorded:
		streamID, _ := AMF0DecodeInt32(pkt.Body[2:])
		logger.LOGT(fmt.Sprintf("stream %d is recorded", streamID))
	case RTMP_CTRL_pingRequest:
		timestamp, _ := AMF0DecodeInt32(pkt.Body[2:])
		rtmp.pingResponse(timestamp)
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

func (rtmp *RTMP) pingResponse(timestamp uint32) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_control
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_CONTROL
	pkt.TimeStamp = 0
	pkt.MessageStreamID = 0

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeInt16(RTMP_CTRL_pingResponse)
	encoder.EncodeInt32(int32(timestamp))
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))

	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) sendAcknowledgement() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_control
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_BYTES_READ_REPORT
	pkt.TimeStamp = 0
	pkt.MessageStreamID = 0
	encoder := &AMF0Encoder{}
	encoder.Init()
	param := int32(rtmp.BytesIn & 0xffffffff)
	encoder.EncodeInt32(param)
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	//logger.LOGT(pkt.Body)
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) AcknowledgementBW() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_control
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_SERVER_BW
	pkt.TimeStamp = 0
	pkt.MessageStreamID = 0

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeInt32(int32(rtmp.TargetBW))
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))

	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) SetPeerBW() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_control
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_CLIENT_BW

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeInt32(int32(rtmp.SelfBW))
	encoder.AppendByte(byte(rtmp.LimitType))
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))

	err = rtmp.SendPacket(pkt, false)

	return
}

func (rtmp *RTMP) SetChunkSize(chunkSize uint32) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_control
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_CHUNK_SIZE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeInt32(int32(chunkSize))
	rtmp.SendChunkSize = chunkSize
	logger.LOGT(fmt.Sprintf("set chunk size %d", rtmp.SendChunkSize))
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) ConnectResult(obj *AMF0Object) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_result")
	idx := obj.AMF0GetPropByIndex(1).Value.NumValue
	encoder.EncodeNumber(idx)

	encoder.AppendByte(AMF0_object)
	encoder.EncodeNamedString("fmsVer", "FMS/5,0,3,3029")
	encoder.EncodeNamedNumber("capabilities", 255)
	encoder.EncodeNamedNumber("mode", 1)
	encoder.EncodeInt24(AMF0_object_end)

	encoder.AppendByte(AMF0_object)
	encoder.EncodeNamedString("level", "status")
	encoder.EncodeNamedString("code", "NetConnection.Connect.Success")
	encoder.EncodeNamedString("description", "Connection succeeded.")
	objEncodeNumber := 3.0
	if obj.Props.Len() == 3 {
		cmdObj := obj.AMF0GetPropByIndex(2)
		objEncode := cmdObj.Value.ObjValue.AMF0GetPropByName("objectEncoding")
		if objEncode != nil {
			objEncodeNumber = objEncode.Value.NumValue
			logger.LOGI(objEncodeNumber)
		}
	}
	encoder.EncodeNamedNumber("objectEncoding", objEncodeNumber)
	encoder.EncodeInt16(4)
	encoder.AppendByte('d')
	encoder.AppendByte('a')
	encoder.AppendByte('t')
	encoder.AppendByte('a')
	encoder.AppendByte(AMF0_ecma_array)
	encoder.EncodeInt32(0)
	encoder.EncodeNamedString("version", "5,0,3,3029")
	encoder.EncodeInt24(AMF0_object_end)
	encoder.EncodeInt24(AMF0_object_end)

	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) CmdError(level string, code string, description string, idx float64) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_error")
	encoder.EncodeNumber(idx)
	encoder.AppendByte(AMF0_null)
	encoder.AppendByte(AMF0_object)
	encoder.EncodeNamedString("level", level)
	encoder.EncodeNamedString("code", code)
	encoder.EncodeNamedString("description", description)
	encoder.EncodeInt24(AMF0_object_end)

	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) CmdStatus(level, code, description, details string, clientID float64, channel int) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = int32(channel)
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	pkt.MessageStreamID = 1
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("onStatus")
	encoder.EncodeNumber(0.0)
	encoder.AppendByte(AMF0_null)
	encoder.AppendByte(AMF0_object)
	encoder.EncodeNamedString("level", level)
	encoder.EncodeNamedString("code", code)
	if len(details) > 0 {
		encoder.EncodeNamedString("details", details)
	}
	encoder.EncodeNamedNumber("clientID", clientID)
	encoder.EncodeInt24(AMF0_object_end)
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) CmdNumberResult(idx float64, numValue float64) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_result")
	encoder.EncodeNumber(idx)
	encoder.AppendByte(AMF0_null)
	encoder.EncodeNumber(numValue)
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) SendCtrl(ctype int, obj uint32, time uint32) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_control
	pkt.MessageTypeID = RTMP_PACKET_TYPE_CONTROL
	encoder := &AMF0Encoder{}
	encoder.Init()
	if ctype == RTMP_CTRL_setBufferLength {
		encoder.EncodeInt16(int16(ctype))
		encoder.EncodeInt32(int32(obj))
		encoder.EncodeInt32(int32(time))
	} else {

		encoder.EncodeInt16(int16(ctype))
		encoder.EncodeInt32(int32(obj))
	}
	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) OnBWDone() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("onBWDone")
	encoder.EncodeNumber(0.0)
	encoder.AppendByte(AMF0_null)

	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) OnBWCheck() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_onbwcheck")
	bwCheckCounts += 1.0
	encoder.EncodeNumber(bwCheckCounts)
	//encoder.AppendByteArray() //18384 char
	strByte := make([]byte, 16384)
	for i := 0; i < 16384; i++ {
		strByte[i] = byte(rand.Int()%95) + 32
	}
	encoder.EncodeString(string(strByte))
	encoder.EncodeNumber(0)
	encoder.AppendByte(AMF0_null)

	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, true)
	return
}

func (rtmp *RTMP) _OnBWDone() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_onbwdone")
	encoder.EncodeNumber(0) //??
	encoder.EncodeNumber(0) //??
	encoder.EncodeNumber(0) //??
	encoder.AppendByte(AMF0_null)

	pkt.Body, err = encoder.GetData()
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) FCUnpublish() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE

	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("FCUnpbulish")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.EncodeString(rtmp.Link.Path)
	encoder.AppendByte(AMF0_null)
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)
	return
}

func (rtmp *RTMP) OnMetadata(data []byte) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INFO
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("onMetaData")
	encoder.AppendByteArray(data)
	pkt.Body, _ = encoder.GetData()

	pkt.MessageLength = uint32(len(pkt.Body))
	rtmp.SendPacket(pkt, false)
}

func (rtmp *RTMP) Connect(publish bool) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("connect")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.AppendByte(AMF0_object)
	encoder.EncodeNamedString("app", rtmp.Link.App)
	encoder.EncodeNamedString("flashver", "WIN 18,0,0,232")
	encoder.EncodeNamedString("tcUrl", rtmp.Link.TcUrl)

	if false == publish {
		encoder.EncodeNamedBool("fpad", false)
		encoder.EncodeNamedNumber("capabilities", 239)
		encoder.EncodeNamedNumber("audioCodecs", float64(rtmp.AudioCodecs))
		encoder.EncodeNamedNumber("videoCodecs", float64(rtmp.VideoCodecs))
		encoder.EncodeNamedNumber("videoFunction", 1.0)
		encoder.EncodeNamedNumber("objectEncoding", 3.0)
	}

	encoder.EncodeInt24(AMF0_object_end)

	pkt.Body, err = encoder.GetData()
	logger.LOGD(pkt.Body)
	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, true)
	return
}

func (rtmp *RTMP) CreateStream() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("createStream")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.AppendByte(AMF0_null)
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, true)

	return
}

func (rtmp *RTMP) SendCheckBW() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_checkbw")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.AppendByte(AMF0_null)
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)

	return
}

func (rtmp *RTMP) SendReleaseStream() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("releaseStream")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.AppendByte(AMF0_null)
	encoder.EncodeString(rtmp.Link.Path)
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, true)

	return
}

func (rtmp *RTMP) SendFCPublish() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("FCPublish")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.AppendByte(AMF0_null)
	encoder.EncodeString(rtmp.Link.Path)
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, true)

	return
}

func (rtmp *RTMP) SendPlay() (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_AV
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	pkt.MessageStreamID = rtmp.StreamID
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("play")
	rtmp.NumInvokes++
	encoder.EncodeNumber(float64(rtmp.NumInvokes))
	encoder.AppendByte(AMF0_null)
	encoder.EncodeString(rtmp.Link.Path)
	if rtmp.Link.SeekTime > 0 {
		encoder.EncodeNumber(float64(rtmp.Link.SeekTime))
	} else {
		encoder.EncodeNumber(-2.0)
	}
	if rtmp.Link.StopTime != 0 {
		encoder.EncodeNumber(float64(rtmp.Link.StopTime - rtmp.Link.SeekTime))
	}
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, true)

	return
}

func (rtmp *RTMP) SendCheckBWResult(transactionID float64) (err error) {
	pkt := &RTMPPacket{}
	pkt.ChunkStreamID = RTMP_channel_Invoke
	pkt.Fmt = 0
	pkt.MessageTypeID = RTMP_PACKET_TYPE_INVOKE
	encoder := &AMF0Encoder{}
	encoder.Init()
	encoder.EncodeString("_result")
	encoder.EncodeNumber(transactionID)
	encoder.AppendByte(AMF0_null)
	encoder.EncodeNumber(0.0)
	pkt.Body, err = encoder.GetData()

	if err != nil {
		return
	}
	pkt.MessageLength = uint32(len(pkt.Body))
	err = rtmp.SendPacket(pkt, false)

	return
}
