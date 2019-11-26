package rtspsrv

import (
	"errors"
	"fmt"
	"time"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediatype/amf"
	"github.com/use-go/websocket-streamserver/wssapi"
)

//RTCP Cmd Type
const (
	RTCPSR   = 200
	RTCPRR   = 201
	RTCPSDES = 202
	RTCPBYE  = 203
	RTCPAPP  = 204
)

//RTCPPacket Type
type RTCPPacket struct {
	version              byte  //2
	padding              bool  //1
	receptionReportCount byte  //5
	packetType           byte  //16
	length               int16 //16 以四字节为单位
	body                 interface{}
}

//RTCPHeaderSR struct
type RTCPHeaderSR struct {
	ssrc            uint32
	ntpTimestampMSW uint32 //从1900到现在的秒数 不是1970
	ntpTimestampLSW uint32 //秒剩下的 1s/2的32次方等于  单位232.8皮秒
	rtpTimestamp    uint32 //RTP时间
	pktCount        uint32 //已发送RTP包个数
	octetCount      uint32 //已发送总字节数
	reportBlock     []RTCPHeaderReportBlock
}

//RTCPHeaderReportBlock struct
type RTCPHeaderReportBlock struct {
	ssrc           uint32
	fractLost      byte   //8
	cumulativeLost uint32 //24
	hSeqNo         uint32
	jitter         uint32
	lastSR         uint32
	delayLastSR    uint32
}

//RTCPHeaderRR Description
type RTCPHeaderRR struct {
	reportBlock []RTCPHeaderReportBlock
}

//RTCPHeaderSDES Description
type RTCPHeaderSDES struct {
}

//RTCPHeaderBYE  Description
type RTCPHeaderBYE struct {
}

func parseRTCP(data []byte) (pkt *RTCPPacket, err error) {
	pkt = &RTCPPacket{}
	if len(data) < 4 {
		return nil, errors.New("data not enough")
	}
	reader := &wssapi.BitReader{}
	reader.Init(data)
	pkt.version = byte(reader.ReadBits(2))
	pkt.padding = (reader.ReadBit() == 1)
	pkt.receptionReportCount = byte(reader.ReadBits(5))
	pkt.packetType = byte(reader.ReadBits(8))
	pkt.length = int16(reader.ReadBits(16))
	if pkt.receptionReportCount < 0 {
		return
	}
	switch pkt.packetType {
	case RTCPSR:
		sr := &RTCPHeaderSR{}
		pkt.body = sr
		sr.ssrc = uint32(reader.Read32Bits())
		sr.ntpTimestampMSW = uint32(reader.Read32Bits())
		sr.ntpTimestampLSW = uint32(reader.Read32Bits())
		sr.rtpTimestamp = uint32(reader.Read32Bits())
		sr.pktCount = uint32(reader.Read32Bits())
		sr.octetCount = uint32(reader.Read32Bits())
		if pkt.receptionReportCount > 0 {
			sr.reportBlock = make([]RTCPHeaderReportBlock, pkt.receptionReportCount)
			for i := 0; i < int(pkt.receptionReportCount); i++ {
				sr.reportBlock[i].ssrc = uint32(reader.Read32Bits())
				sr.reportBlock[i].fractLost = byte(reader.ReadBits(8))
				sr.reportBlock[i].cumulativeLost = uint32(reader.ReadBits(24))
				sr.reportBlock[i].hSeqNo = uint32(reader.Read32Bits())
				sr.reportBlock[i].jitter = uint32(reader.Read32Bits())
				sr.reportBlock[i].lastSR = uint32(reader.Read32Bits())
				sr.reportBlock[i].delayLastSR = uint32(reader.Read32Bits())
			}
		}
	case RTCPRR:
		rr := &RTCPHeaderRR{}
		if pkt.receptionReportCount > 0 {
			rr.reportBlock = make([]RTCPHeaderReportBlock, pkt.receptionReportCount)
			for i := 0; i < int(pkt.receptionReportCount); i++ {
				rr.reportBlock[i].ssrc = uint32(reader.Read32Bits())
				rr.reportBlock[i].fractLost = byte(reader.ReadBits(8))
				rr.reportBlock[i].cumulativeLost = uint32(reader.ReadBits(24))
				rr.reportBlock[i].hSeqNo = uint32(reader.Read32Bits())
				rr.reportBlock[i].jitter = uint32(reader.Read32Bits())
				rr.reportBlock[i].lastSR = uint32(reader.Read32Bits())
				rr.reportBlock[i].delayLastSR = uint32(reader.Read32Bits())
			}
		}
	case RTCPSDES:
	case RTCPAPP:
	case RTCPBYE:
	default:
		logger.LOGW(fmt.Sprintf("rtcp packet type :%d nor supproted", pkt.packetType))
	}
	return
}

func createSR(ssrc, rtpTime, rtpCount, sendCount uint32) (data []byte) {
	enc := &amf.AMF0Encoder{}
	enc.Init()
	var tmp32 uint32
	version := uint32(2)
	padding := uint32(1)
	receptionReportCount := uint32(0)
	length := uint32(6)
	tmp32 = ((version << 30) | (padding << 29) | (receptionReportCount << 24) | (uint32(RTCPSR) << 16) | (length))

	enc.EncodeUint32(tmp32) //header

	enc.EncodeUint32(ssrc) //ssrc
	//msw
	tmp64 := time.Now().Unix() + 0x83AA7E80
	tmp32 = uint32(tmp64 & 0xffffffff)
	enc.EncodeUint32(tmp32)
	//lsw
	tmp64 = time.Now().Unix() - time.Now().UnixNano()
	tmp32 = uint32(((tmp64 << 32) / 1000000000) & 0xffffffff)
	enc.EncodeUint32(tmp32)
	//rtp time
	enc.EncodeUint32(rtpTime)
	//pkt count
	enc.EncodeUint32(rtpCount)
	//data size
	enc.EncodeUint32(sendCount)

	data, _ = enc.GetData()
	return
}
