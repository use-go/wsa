package ts

import (
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/aac"
	"github.com/use-go/websocket-streamserver/mediaTypes/amf"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
)

func (tsCreater *TsCreater) audioPayload(tag *flv.FlvTag) (payload []byte, size int) {
	if tsCreater.audioTypeId == 0xf {
		//adth:=aac.GenerateADTHeader(tsCreater.asc,len(tag.Data)-2)
		adth := aac.CreateAACADTHeader(tsCreater.asc, len(tag.Data)-2)
		size = len(adth) + len(tag.Data) - 2
		payload = make([]byte, size)
		copy(payload, adth)
		copy(payload[len(adth):], tag.Data[2:])
		return
	} else if tsCreater.audioTypeId == 0x03 || tsCreater.audioTypeId == 0x04 {
		size = len(tag.Data) - 1
		payload = make([]byte, size)
		copy(payload, tag.Data[1:])
		return
	} else {
		logger.LOGF(tsCreater.audioTypeId)
		logger.LOGE("invalid audio type")
	}
	return
}

func (tsCreater *TsCreater) calPcrPtsDts(tag *flv.FlvTag) (pcr, pcrExt, pts, dts uint64) {
	timeMS := uint64(tag.Timestamp)
	pcr = (timeMS * 90) & 0x1ffffffff
	pcrExt = (timeMS * PCR_HZ / 1000) & 0x1ff
	if len(tag.Data) < 5 {
		logger.LOGF("wtf")
	}
	compositionTime, _ := amf.AMF0DecodeInt24(tag.Data[2:])
	u64 := uint64(compositionTime)
	dts = timeMS*90 + 90
	pts = dts + u64*90
	return
}

func (tsCreater *TsCreater) calAudioTime(tag *flv.FlvTag) {
	//tmp:=int64(90*tsCreater.audioSampleHz*int(tag.Timestamp-tsCreater.beginTime))
	tmp := int64(90 * int(tag.Timestamp))
	//logger.LOGT(tmp,tag.Timestamp-tsCreater.beginTime)
	//audioPtsDelta := int64(90000 * int64(tsCreater.audioFrameSize) / int64(tsCreater.audioSampleHz))
	//tsCreater.audioPts += audioPtsDelta
	//logger.LOGD(tsCreater.audioPts)
	tsCreater.audioPts = tmp
}
