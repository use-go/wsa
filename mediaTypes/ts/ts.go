package ts

import (
	"container/list"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/aac"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/mediaTypes/h264"
	"github.com/use-go/websocket-streamserver/mediaTypes/mp3"
)

var crc32Table []uint32

const (
	PMT_ID        = 0x100
	Video_Id      = 0x101
	Audio_Id      = 0x102
	TS_length     = 188
	PCR_HZ        = 27000000
	TS_VIDEO_ONLY = false
)

func init() {
	crc32Table = make([]uint32, 256)
	for i := uint32(0); i < 256; i++ {
		k := uint32(0)
		for j := (i << 24) | 0x800000; j != 0x80000000; j <<= 1 {
			tmp := ((k ^ j) & 0x80000000)
			if tmp != 0 {
				k = (k << 1) ^ 0x04c11db7
			} else {
				k = (k << 1) ^ 0
			}
		}
		crc32Table[i] = k
	}
}

func Crc32Calculate(buffer []uint8) (crc32reg uint32) {
	crc32reg = 0xFFFFFFFF
	for _, v := range buffer {
		crc32reg = (crc32reg << 8) ^ crc32Table[((crc32reg>>24)^uint32(v))&0xFF]
	}

	return crc32reg
}

type TsCreater struct {
	tsVcount    int16
	tsAcount    int16
	audioHeader []byte
	//asc                      aac.AudioSpecificConfig
	asc                      *aac.MP4AACAudioSpecificConfig
	videoHeader              []byte
	sps                      []byte
	pps                      []byte
	sei                      []byte
	videoTypeId              int
	audioTypeId              int
	audioFrameSize           int
	audioSampleHz            int
	audioPts                 int64
	beginTime                uint32
	nowTime                  uint32
	tsCache                  *list.List
	keyframeWrited           bool
	encodeAudio              bool
	data_alignment_indicator bool
}

func (tsCreater *TsCreater) Reset() {
	tsCreater.tsCache = list.New()
	tsCreater.beginTime = 0xffffffff
	tsCreater.nowTime = 0
	tsCreater.data_alignment_indicator = false
	tsCreater.audioPts = 0
}

func (tsCreater *TsCreater) writePCR(buf []byte, pcr, pcrExt uint64) {
	cur := 0
	buf[cur] = byte((pcr >> 25) & 0xff)
	cur++
	buf[cur] = byte((pcr >> 17) & 0xff)
	cur++
	buf[cur] = byte((pcr >> 9) & 0xff)
	cur++
	buf[cur] = byte((pcr >> 1) & 0xff)
	cur++
	buf[cur] = byte((pcr << 7) & 0x80)
	buf[cur] |= byte((pcrExt >> 8) & 0x01)
	cur++
	buf[cur] = byte(pcrExt & 0xff)
	cur++
}

func (tsCreater *TsCreater) AddTag(tag *flv.FlvTag) {
	if flv.FlvTagScriptData == tag.TagType {
		return
	}
	if tsCreater.tsCache == nil {
		tsCreater.keyframeWrited = false
		tsCreater.encodeAudio = TS_VIDEO_ONLY
		tsCreater.tsCache = list.New()
	}
	tsCreater.nowTime = tag.Timestamp
	if true == tsCreater.avHeaderAdded(tag) {
		if 0xffffffff == tsCreater.beginTime {
			tsCreater.beginTime = tag.Timestamp
			if tsCreater.audioHeader == nil || TS_VIDEO_ONLY {
				tsCreater.encodeAudio = false
			} else {
				tsCreater.encodeAudio = true
			}
			tsCreater.addPatPmt()
		}
		var addDts, addPCR bool
		if flv.FlvTagAudio == tag.TagType {
			addDts = false
			addPCR = false
			if TS_VIDEO_ONLY {
				return
			}
		} else {
			addDts = true
			addPCR = true
		}

		var tsCount, padSize int
		var tmp16 uint16

		var dataPayload []byte
		var payloadSize int

		if flv.FlvTagAudio == tag.TagType {
			dataPayload, payloadSize = tsCreater.audioPayload(tag)
			if 0 == payloadSize || nil == dataPayload {
				return
			}
		} else if flv.FlvTagVideo == tag.TagType {
			dataPayload = tsCreater.videoPayload(tag)
			if nil == dataPayload {
				logger.LOGE(dataPayload)
				return
			}
			payloadSize = len(dataPayload)
		}

		tsCount, padSize = tsCreater.getTsCount(payloadSize, addPCR, addDts)
		if tag.TagType == flv.FlvTagAudio {
			tsCreater.calAudioTime(tag)

		}
		tsBuf := make([]byte, TS_length)
		cur := 0

		pcr, pcrExt, pcrPts, pcrDts := tsCreater.calPcrPtsDts(tag)

		//logger.LOGD(pcr,pcrPts,pcrDts-pcrPts)
		if 1 == tsCount {
			for idx := 0; idx < TS_length; idx++ {
				tsBuf[idx] = 0xff
			}
			cur = 0
			tsBuf[cur] = 0x47
			cur++
			if flv.FlvTagAudio == tag.TagType {
				tmp16 = uint16(0x4000 | Audio_Id)
			} else {
				tmp16 = uint16(0x4000 | Video_Id)
			}
			tsBuf[cur] = byte(tmp16 >> 8)
			cur++
			tsBuf[cur] = byte(tmp16 & 0xff)
			cur++
			if addPCR || padSize > 0 {
				if flv.FlvTagAudio == tag.TagType {
					tsBuf[cur] = byte(0x30 | tsCreater.tsAcount)
					cur++
				} else {
					tsBuf[cur] = byte(0x30 | tsCreater.tsVcount)
					cur++
				}
			} else {
				if flv.FlvTagAudio == tag.TagType {
					tsBuf[cur] = byte(0x10 | tsCreater.tsAcount)
					cur++
				} else {
					tsBuf[cur] = byte(0x10 | tsCreater.tsVcount)
					cur++
				}
			}
			if flv.FlvTagAudio == tag.TagType {
				tsCreater.tsAcount++
				if tsCreater.tsAcount == 16 {
					tsCreater.tsAcount = 0
				}
			} else {
				tsCreater.tsVcount++
				if tsCreater.tsVcount == 16 {
					tsCreater.tsVcount = 0
				}
			}

			//!四字节头
			//PCR、PAD
			//timeMS := uint64(tag.Timestamp - tsCreater.beginTime)
			//pcr := uint64(((timeMS * (PCR_HZ / 1000)) / 300) % 0x200000000)
			if addPCR {
				adpLength := 7 + padSize
				tsBuf[cur] = byte(adpLength)
				cur++
				tsBuf[cur] = 0x10
				cur++
				tsCreater.writePCR(tsBuf[cur:], pcr, pcrExt)
				cur += 6
				cur += padSize
			} else if false == addPCR && padSize > 0 {
				adpLength := padSize - 1
				tsBuf[cur] = byte(adpLength)
				cur++
				if padSize > 1 {
					tsBuf[cur] = 0
					cur += padSize - 1
				}
			}
			//!PCR PAD
			//PES
			tsBuf[cur] = 0x00
			cur++
			tsBuf[cur] = 0x00
			cur++
			tsBuf[cur] = 0x01
			cur++
			if flv.FlvTagAudio == tag.TagType {
				tsBuf[cur] = 0xc0
				cur++
				tmp16 = uint16(payloadSize + 8)
				tsBuf[cur] = byte(tmp16 >> 8)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++
				tsBuf[cur] = 0x80
				cur++
				tsBuf[cur] = 0x80
				cur++
				tsBuf[cur] = 0x05
				cur++

				tsBuf[cur] = byte((0x20) | ((tsCreater.audioPts & 0x1c0000000) >> 29) | 1)
				cur++
				tmp16 = uint16(((tsCreater.audioPts & 0x3fff8000) >> 14) | 1)
				tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++
				tmp16 = uint16((tsCreater.audioPts&0x7fff)<<1) | 1
				tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++

				copy(tsBuf[cur:], dataPayload)
				cur += payloadSize
			} else {
				tsBuf[cur] = 0xe0
				cur++
				tsBuf[cur] = 0x00
				cur++
				tsBuf[cur] = 0x00
				cur++
				if tsCreater.data_alignment_indicator {
					tsBuf[cur] = 0x80
				} else {
					tsBuf[cur] = 0x84
					tsCreater.data_alignment_indicator = true
				}
				cur++
				tsBuf[cur] = 0xc0
				cur++
				tsBuf[cur] = 0x0a
				cur++

				tsBuf[cur] = byte((3 << 4) | ((pcrPts & 0x1c0000000) >> 29) | 1)
				cur++
				tmp16 = uint16(((pcrPts & 0x3fff8000) >> 14) | 1)
				tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++
				tmp16 = uint16(((pcrPts & 0x7fff) << 1) | 1)
				tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++
				tsBuf[cur] = byte((1 << 4) | ((pcrDts & 0x1c0000000) >> 29) | 1)
				cur++
				tmp16 = uint16(((pcrDts & 0x3fff8000) >> 14) | 1)
				tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++
				tmp16 = uint16(((pcrDts & 0x7fff) << 1) | 1)
				tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
				cur++
				tsBuf[cur] = byte(tmp16 & 0xff)
				cur++
				copy(tsBuf[cur:], dataPayload)
				cur += payloadSize
			}
			//!PES
			tsCreater.appendTsPkt(tsBuf)

		} else {
			//不止一个包的情况
			payloadCur := 0
			for i := 0; i < tsCount; i++ {
				for idx := 0; idx < len(tsBuf); idx++ {
					tsBuf[idx] = 0xff
				}
				cur = 0
				//第一帧
				if 0 == i {
					tsBuf[cur] = 0x47
					cur++
					if flv.FlvTagAudio == tag.TagType {
						tmp16 = uint16(0x4000 | Audio_Id)
					} else {
						tmp16 = uint16(0x4000 | Video_Id)
					}
					tsBuf[cur] = byte(tmp16 >> 8)
					cur++
					tsBuf[cur] = byte(tmp16 & 0xff)
					cur++
					if addPCR {
						if flv.FlvTagAudio == tag.TagType {
							tsBuf[cur] = byte(0x30 | tsCreater.tsAcount)
							cur++
						} else {
							tsBuf[cur] = byte(0x30 | tsCreater.tsVcount)
							cur++
						}
					} else {
						if flv.FlvTagAudio == tag.TagType {
							tsBuf[cur] = byte(0x10 | tsCreater.tsAcount)
							cur++
						} else {
							tsBuf[cur] = byte(0x10 | tsCreater.tsVcount)
							cur++
						}
					}

					if flv.FlvTagAudio == tag.TagType {
						tsCreater.tsAcount++
						if tsCreater.tsAcount == 16 {
							tsCreater.tsAcount = 0
						}
					} else {
						tsCreater.tsVcount++
						if tsCreater.tsVcount == 16 {
							tsCreater.tsVcount = 0
						}
					}

					//!四字节头
					//PCR
					//timeMS := uint64(tag.Timestamp - tsCreater.beginTime)
					//pcr := uint64(((timeMS * (PCR_HZ / 1000)) / 300) % 0x200000000)
					if addPCR {
						adpLength := 7
						tsBuf[cur] = byte(adpLength)
						cur++
						tsBuf[cur] = 0x10
						cur++
						tsCreater.writePCR(tsBuf[cur:], pcr, pcrExt)
						cur += 6
					}
					//!PCR
					//PES头
					tsBuf[cur] = 0x00
					cur++
					tsBuf[cur] = 0x00
					cur++
					tsBuf[cur] = 0x01
					cur++
					if flv.FlvTagAudio == tag.TagType {
						tsBuf[cur] = 0xc0
						cur++
						tmp16 = uint16(payloadSize + 8)
						tsBuf[cur] = byte(tmp16 >> 8)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
						tsBuf[cur] = 0x80
						cur++
						tsBuf[cur] = 0x80
						cur++
						tsBuf[cur] = 0x05
						cur++

						tsBuf[cur] = byte((0x20) | ((tsCreater.audioPts & 0x1c0000000) >> 29) | 1)
						cur++
						tmp16 = uint16(((tsCreater.audioPts & 0x3fff8000) >> 14) | 1)
						tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
						tmp16 = uint16((tsCreater.audioPts&0x7fff)<<1) | 1
						tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
					} else {

						tsBuf[cur] = 0xe0 //stream id:ITU-T h262| ITU-T h.264
						cur++
						tsBuf[cur] = 0x00 //stream id
						cur++
						tsBuf[cur] = 0x00 //stream id
						cur++
						if tsCreater.data_alignment_indicator {
							tsBuf[cur] = 0x80
						} else {
							tsBuf[cur] = 0x84
							tsCreater.data_alignment_indicator = true
						}
						cur++
						tsBuf[cur] = 0xc0
						cur++
						tsBuf[cur] = 0x0a
						cur++

						tsBuf[cur] = byte((3 << 4) | ((pcrPts & 0x1c0000000) >> 29) | 1)
						cur++
						tmp16 = uint16(((pcrPts & 0x3fff8000) >> 14) | 1)
						tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
						tmp16 = uint16(((pcrPts & 0x7fff) << 1) | 1)
						tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
						tsBuf[cur] = byte((1 << 4) | ((pcrDts & 0x1c0000000) >> 29) | 1)
						cur++
						tmp16 = uint16(((pcrDts & 0x3fff8000) >> 14) | 1)
						tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
						tmp16 = uint16(((pcrDts & 0x7fff) << 1) | 1)
						tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
						cur++
						tsBuf[cur] = byte(tmp16 & 0xff)
						cur++
					}
					//!PES头
					copy(tsBuf[cur:], dataPayload[payloadCur:TS_length-cur])
					payloadCur += TS_length - cur
					tsCreater.appendTsPkt(tsBuf)
				} else {
					//四字节头
					tsBuf[cur] = 0x47
					cur++
					if flv.FlvTagAudio == tag.TagType {
						tmp16 = uint16(Audio_Id)
					} else {
						tmp16 = uint16(Video_Id)
					}
					tsBuf[cur] = byte(tmp16 >> 8)
					cur++
					tsBuf[cur] = byte(tmp16 & 0xff)
					cur++
					//!3字节头
					if i == tsCount-1 && padSize != 0 {
						//最后一帧，且有pad
						if flv.FlvTagAudio == tag.TagType {
							tsBuf[cur] = byte(0x30 | tsCreater.tsAcount)
							cur++
						} else {
							tsBuf[cur] = byte(0x30 | tsCreater.tsVcount)
							cur++
						}
						tsBuf[cur] = byte(padSize - 1)
						cur++
						if padSize != 1 {
							tsBuf[cur] = 0
							cur++
						}
						copy(tsBuf[4+padSize:], dataPayload[payloadCur:payloadCur+TS_length-4-padSize])
						payloadCur += TS_length - 4 - padSize
					} else {
						//普通添加数据
						if flv.FlvTagAudio == tag.TagType {
							tsBuf[cur] = byte(0x10 | tsCreater.tsAcount)
							cur++
						} else {
							tsBuf[cur] = byte(0x10 | tsCreater.tsVcount)
							cur++
						}

						tmps := dataPayload[payloadCur : payloadCur+TS_length-cur]
						copy(tsBuf[cur:], tmps)
						payloadCur += TS_length - cur
					}
					if flv.FlvTagAudio == tag.TagType {
						tsCreater.tsAcount++
						if tsCreater.tsAcount == 16 {
							tsCreater.tsAcount = 0
						}
					} else {
						tsCreater.tsVcount++
						if tsCreater.tsVcount == 16 {
							tsCreater.tsVcount = 0
						}
					}
					tsCreater.appendTsPkt(tsBuf)
				}
			}
		}
	}
}

func (tsCreater *TsCreater) GetDuration() (sec int) {
	return int(tsCreater.nowTime - tsCreater.beginTime)
}

func (tsCreater *TsCreater) FlushTsList() (tsList *list.List) {
	tsList = tsCreater.tsCache
	tsCreater.tsCache = list.New()
	tsCreater.keyframeWrited = false
	tsCreater.addPatPmt()
	return tsList
}

func (tsCreater *TsCreater) avHeaderAdded(tag *flv.FlvTag) (headerGeted bool) {
	if TS_VIDEO_ONLY && tsCreater.videoHeader != nil {
		return true
	}
	if tsCreater.audioHeader != nil && tsCreater.videoHeader != nil {
		return true
	}
	tsCreater.beginTime = 0xffffffff
	if tag.TagType == flv.FlvTagAudio {
		if tsCreater.audioHeader != nil {
			//防止只有音频的情况
			return true
		}
		tsCreater.audioHeader = make([]byte, len(tag.Data))
		copy(tsCreater.audioHeader, tag.Data)
		tsCreater.parseAudioType(tsCreater.audioHeader)
		return false
	}
	if tag.TagType == flv.FlvTagVideo {
		if tsCreater.videoHeader != nil {
			//防止没有音频的情况
			return true
		}
		tsCreater.videoHeader = make([]byte, len(tag.Data))
		copy(tsCreater.videoHeader, tag.Data)
		tsCreater.videoTypeId = 0x1b
		tsCreater.parseAVC(tsCreater.videoHeader)
		return false
	}
	return false
}

func (tsCreater *TsCreater) parseAudioType(data []byte) {
	audioCodec := data[0] >> 4
	switch audioCodec {
	case flv.SoundFormatAAC:
		tsCreater.audioFrameSize = 1024
		//tsCreater.asc = aac.GenerateAudioSpecificConfig(data[2:])
		tsCreater.asc = aac.MP4AudioGetConfig(data[2:])
		tsCreater.audioSampleHz = int(tsCreater.asc.Sample_rate)
		tsCreater.audioTypeId = 0x0f
	case flv.SoundFormatMP3:
		tsCreater.audioFrameSize = 1152
		mp3Header, err := mp3.ParseMP3Header(data[1:])
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		tsCreater.audioSampleHz = mp3Header.SampleRate
		if mp3Header.Version == 3 {
			tsCreater.audioTypeId = 0x03
		} else {
			tsCreater.audioTypeId = 0x04
		}
	default:
		logger.LOGE("ts audio type not supported", audioCodec)
		return
	}

}

func (tsCreater *TsCreater) parseAVC(data []byte) {
	if data[0] == 0x17 && data[1] == 0 {
		//avc
		tsCreater.sps, tsCreater.pps = h264.GetSpsPpsFromAVC(data[5:])
	}
}

func (tsCreater *TsCreater) addPatPmt() {
	cur := 0
	var tmp16 uint16
	var tmp32 uint32
	tsBuf := make([]byte, TS_length)
	for idx := 0; idx < TS_length; idx++ {
		tsBuf[idx] = 0xff
	}
	//pat
	tsBuf[cur] = 0x47
	cur++
	tsBuf[cur] = 0x40
	cur++
	tsBuf[cur] = 0x00
	cur++
	tsBuf[cur] = 0x10
	cur++

	tsBuf[cur] = 0x00 //0个补充字节
	cur++

	tsBuf[cur] = 0x00 //table id
	cur++
	tmp16 = (((0xb0) << 8) | 0xd) //section length
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	tsBuf[cur] = 0x00 //transport stream id
	cur++
	tsBuf[cur] = 0x01
	cur++
	tsBuf[cur] = 0xc1 //vesion 0,current valid
	cur++
	tsBuf[cur] = 0x00 //section num
	cur++
	tsBuf[cur] = 0x00 //last section num
	cur++
	tsBuf[cur] = 0x00 //program num
	cur++
	tsBuf[cur] = 0x01
	cur++
	tmp16 = (0xe000 | PMT_ID) //PMT id
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	tmp32 = Crc32Calculate(tsBuf[5:cur]) //CRC
	tsBuf[cur] = byte((tmp32 >> 24) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp32 >> 16) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp32 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp32 >> 0) & 0xff)
	cur++

	tsCreater.appendTsPkt(tsBuf)
	//pmt
	tsBuf = make([]byte, TS_length)
	for idx := 0; idx < TS_length; idx++ {
		tsBuf[idx] = 0xff
	}
	cur = 0

	tsBuf[cur] = 0x47
	cur++
	tmp16 = ((0x40 << 8) | PMT_ID)
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	tsBuf[cur] = 0x10
	cur++

	tsBuf[cur] = 0x00 //0个补充字节
	cur++

	tsBuf[cur] = 0x02 //table id
	cur++

	if tsCreater.encodeAudio {
		tmp16 = ((0xb0 << 8) | 0x17) //section length
	} else {
		tmp16 = ((0xb0 << 8) | 0x12)
	}
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	tsBuf[cur] = 0x00 //transport stream id
	cur++
	tsBuf[cur] = 0x01
	cur++
	tsBuf[cur] = 0xc1 //vesion 0,current valid
	cur++
	tsBuf[cur] = 0x00 //section num
	cur++
	tsBuf[cur] = 0x00 //last section num
	cur++
	tmp16 = (0xe000 | Video_Id) //pcr pid
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	tmp16 = 0xf000 //program info length = 0
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	//video
	tsBuf[cur] = byte(tsCreater.videoTypeId)
	cur++
	tmp16 = (0xe000 | Video_Id)
	tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
	cur++
	tsBuf[cur] = 0xf0
	cur++
	tsBuf[cur] = 0x00
	cur++
	//audio
	if tsCreater.encodeAudio {
		tsBuf[cur] = byte(tsCreater.audioTypeId)
		cur++
		tmp16 = (0xe000 | Audio_Id)
		tsBuf[cur] = byte((tmp16 >> 8) & 0xff)
		cur++
		tsBuf[cur] = byte((tmp16 >> 0) & 0xff)
		cur++
		tsBuf[cur] = 0xf0
		cur++
		tsBuf[cur] = 0x00
		cur++

	}
	tmp32 = Crc32Calculate(tsBuf[5:cur]) //CRC
	tsBuf[cur] = byte((tmp32 >> 24) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp32 >> 16) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp32 >> 8) & 0xff)
	cur++
	tsBuf[cur] = byte((tmp32 >> 0) & 0xff)
	cur++

	tsCreater.appendTsPkt(tsBuf)
}

func (tsCreater *TsCreater) appendTsPkt(tsBuf []byte) {
	tmp := make([]byte, len(tsBuf))
	copy(tmp, tsBuf)
	tsCreater.tsCache.PushBack(tmp)
}

func (tsCreater *TsCreater) getTsCount(dataSize int, addPCR, addDts bool) (tsCount, padSize int) {
	firstValidSize := TS_length - 4
	if addPCR {
		firstValidSize -= 8
	}
	if addDts {
		firstValidSize -= 19
	} else {
		firstValidSize -= 14
	}
	validSize := TS_length - 4

	if dataSize <= firstValidSize {
		tsCount = 1
		padSize = firstValidSize - dataSize
		return tsCount, padSize
	} else {
		size := dataSize
		size -= firstValidSize
		tsCount = size/validSize + 1
		padSize = size % validSize
		if padSize != 0 {
			tsCount++
			padSize = validSize - padSize
		}
		return tsCount, padSize
	}

	return
}

func (tsCreater *TsCreater) videoPayload(tag *flv.FlvTag) (payload []byte) {
	if tag.Data[0] == 0x17 && tag.Data[1] == 0 {
		tsCreater.parseAVC(tag.Data)
		return nil
	}
	nalCur := 5
	getKeyframe := false
	nalList := list.New()
	totalNalSize := 0
	for nalCur < len(tag.Data) {
		nalSize := 0
		nalSizeSlice := tag.Data[nalCur : nalCur+4]
		nalSize = (int(nalSizeSlice[0]) << 24) | (int(nalSizeSlice[1]) << 16) |
			(int(nalSizeSlice[2]) << 8) | (int(nalSizeSlice[3]) << 0)
		nalCur += 4
		nalType := tag.Data[nalCur] & 0x1f

		switch nalType {
		case h264.NalType_sei:
			tsCreater.sei = make([]byte, nalSize)
			copy(tsCreater.sei, tag.Data[nalCur:nalCur+nalSize])
		case h264.NalType_sps:
			tsCreater.sps = make([]byte, nalSize)
			copy(tsCreater.sps, tag.Data[nalCur:nalCur+nalSize])
		case h264.NalType_pps:
			tsCreater.pps = make([]byte, nalSize)
			copy(tsCreater.pps, tag.Data[nalCur:nalCur+nalSize])
		case h264.NalType_idr:
			getKeyframe = true
			tsCreater.keyframeWrited = true
			totalNalSize += nalSize + 3
			tmp := make([]byte, nalSize)
			copy(tmp, tag.Data[nalCur:nalCur+nalSize])
			nalList.PushBack(tmp)
		case h264.NalType_aud:
			if /*0!=totalNalSize&&*/ nalSize != 2 {
				totalNalSize += nalSize + 3
				tmp := make([]byte, nalSize)
				copy(tmp, tag.Data[nalCur:nalCur+nalSize])
				nalList.PushBack(tmp)
			}
		default:
			totalNalSize += nalSize + 3
			tmp := make([]byte, nalSize)
			copy(tmp, tag.Data[nalCur:nalCur+nalSize])
			nalList.PushBack(tmp)
		}
		nalCur += nalSize
	}

	if false == getKeyframe && tsCreater.keyframeWrited == false {
		logger.LOGE("no keyframe")
		return nil
	}

	if nalList.Len() == 0 {
		logger.LOGE("no frame")
		return nil
	}

	if getKeyframe {
		payloadSize := totalNalSize + 5

		if len(tsCreater.sps) > 0 {
			payloadSize += len(tsCreater.sps) + 3
		}
		if len(tsCreater.pps) > 0 {
			payloadSize += len(tsCreater.pps) + 3
		}
		if len(tsCreater.sei) > 0 {
			payloadSize += len(tsCreater.sei) + 3
		}
		tmp32 := 0
		payload = make([]byte, payloadSize)
		payload[tmp32] = 0x00
		tmp32++
		payload[tmp32] = 0x00
		tmp32++
		payload[tmp32] = 0x01
		tmp32++
		payload[tmp32] = 0x09
		tmp32++
		payload[tmp32] = 0x10
		tmp32++

		if len(tsCreater.sps) > 0 {
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x01
			tmp32++

			copy(payload[tmp32:], tsCreater.sps)
			tmp32 += len(tsCreater.sps)
		}

		if len(tsCreater.pps) > 0 {
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x01
			tmp32++
			copy(payload[tmp32:], tsCreater.pps)
			tmp32 += len(tsCreater.pps)
		}

		if len(tsCreater.sei) > 0 {
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x01
			tmp32++
			copy(payload[tmp32:], tsCreater.sei)
			tmp32 += len(tsCreater.sei)
		}

		for e := nalList.Front(); e != nil; e = e.Next() {
			buf := e.Value.([]byte)
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x01
			tmp32++
			copy(payload[tmp32:], buf)
			tmp32 += len(buf)
		}
	} else {
		payloadSize := totalNalSize + 5
		payload = make([]byte, payloadSize)
		tmp32 := 0
		payload[tmp32] = 0x00
		tmp32++
		payload[tmp32] = 0x00
		tmp32++
		payload[tmp32] = 0x01
		tmp32++
		payload[tmp32] = 0x09
		tmp32++
		payload[tmp32] = 0x10
		tmp32++

		for e := nalList.Front(); e != nil; e = e.Next() {
			buf := e.Value.([]byte)
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x00
			tmp32++
			payload[tmp32] = 0x01
			tmp32++

			copy(payload[tmp32:], buf)
			tmp32 += len(buf)
		}
	}

	return payload
}
