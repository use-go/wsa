package RTSPService

import (
	"errors"
	"fmt"
	"time"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/amf"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

//RTCP Cmd Type
const (
	RTCP_SR   = 200
	RTCP_RR   = 201
	RTCP_SDES = 202
	RTCP_BYE  = 203
	RTCP_APP  = 204
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
	ssrc            uint32
	fract_lost      byte   //8
	cumulative_lost uint32 //24
	h_seq_no        uint32
	jitter          uint32
	lastSR          uint32
	delay_last_SR   uint32
}

type RTCPHeaderRR struct {
	reportBlock []RTCPHeaderReportBlock
}

type RTCPHeaderSDES struct {
}

type RTCPHeaderBYE struct {
}

func parseRTCP(data []byte) (pkt *RTCPPacket, err error) {
	pkt = &RTCPPacket{}
	if len(data) < 4 {
		return nil, errors.New("data not enough")
	}
	reader := &wssAPI.BitReader{}
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
	case RTCP_SR:
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
				sr.reportBlock[i].fract_lost = byte(reader.ReadBits(8))
				sr.reportBlock[i].cumulative_lost = uint32(reader.ReadBits(24))
				sr.reportBlock[i].h_seq_no = uint32(reader.Read32Bits())
				sr.reportBlock[i].jitter = uint32(reader.Read32Bits())
				sr.reportBlock[i].lastSR = uint32(reader.Read32Bits())
				sr.reportBlock[i].delay_last_SR = uint32(reader.Read32Bits())
			}
		}
	case RTCP_RR:
		rr := &RTCPHeaderRR{}
		if pkt.receptionReportCount > 0 {
			rr.reportBlock = make([]RTCPHeaderReportBlock, pkt.receptionReportCount)
			for i := 0; i < int(pkt.receptionReportCount); i++ {
				rr.reportBlock[i].ssrc = uint32(reader.Read32Bits())
				rr.reportBlock[i].fract_lost = byte(reader.ReadBits(8))
				rr.reportBlock[i].cumulative_lost = uint32(reader.ReadBits(24))
				rr.reportBlock[i].h_seq_no = uint32(reader.Read32Bits())
				rr.reportBlock[i].jitter = uint32(reader.Read32Bits())
				rr.reportBlock[i].lastSR = uint32(reader.Read32Bits())
				rr.reportBlock[i].delay_last_SR = uint32(reader.Read32Bits())
			}
		}
	case RTCP_SDES:
	case RTCP_APP:
	case RTCP_BYE:
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
	reception_report_count := uint32(0)
	length := uint32(6)
	tmp32 = ((version << 30) | (padding << 29) | (reception_report_count << 24) | (uint32(RTCP_SR) << 16) | (length))

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
