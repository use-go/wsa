package mp4

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/aac"
	"github.com/use-go/websocket-streamserver/mediaTypes/amf"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/mediaTypes/h264"
	"github.com/use-go/websocket-streamserver/mediaTypes/mp3"
	"github.com/use-go/websocket-streamserver/utils"
)

const saveToFile = false
const (
	videoTrack = 1
	audioTrack = 2
)

// audio Type
const (
	PCMPlatformEndian       = 0
	ADPCM                   = 1
	MP3                     = 2
	PCMlittleEndian         = 3
	Nellymoser16Mono        = 4
	Nellymoser8Mono         = 5
	Nellymoser              = 6
	G711ALawLogarithmicPCM  = 7
	G711MuLawLogarithmicPCM = 8
	AAC                     = 10
	Speex                   = 11
	Mp38                    = 14
	DeviceSpecificSound     = 15
)

//useragent
const (
	UserAgentFireFox = "firefox"
	UserAgentAndroid = "android"
	UserAgentIOS     = "ios"
	UserAgentChrome  = "chrome"
)

//FMP4Slice struct
type FMP4Slice struct {
	Data []byte
	Idx  int //-1 for init,0 base
	Type int //8 audio,9 video
}

// FMP4Creater struct
type FMP4Creater struct {
	videoIdx      int
	videoInited   bool
	videoLastTime uint32
	audioIdx      int
	audioInited   bool
	audioLastTime uint32
	audioCodecID  int

	width               int
	height              int
	fps                 int
	audioSampleSize     uint32
	audioSampleRate     uint32
	audioSampleDuration uint32
	ascData             []byte
	audioType           int
	firstNoZeroTime     uint32
	keyframeGeted       bool
}

//FMP4Flags struct
type FMP4Flags struct {
	IsLeading           uint32
	SampleDependsOn     uint32
	SampleIsDependedOn  uint32
	SampleHasRedundancy uint32
	IsAsync             uint32
}

//AddFlvTag for Data Package
func (fmp4Creater *FMP4Creater) AddFlvTag(tag *flv.FlvTag) (slice *FMP4Slice) {

	if 0 == fmp4Creater.firstNoZeroTime && tag.Timestamp != 0 {
		fmp4Creater.firstNoZeroTime = tag.Timestamp
	}
	tmpTag := tag.Copy()
	if 0 != fmp4Creater.firstNoZeroTime {
		if tmpTag.Timestamp >= fmp4Creater.firstNoZeroTime {
			tmpTag.Timestamp -= fmp4Creater.firstNoZeroTime
		}
	}
	switch tag.TagType {
	case flv.FlvTagAudio:
		return fmp4Creater.handleAudioTag(tmpTag)
	case flv.FlvTagVideo:
		return fmp4Creater.handleVideoTag(tmpTag)
	default:
		logger.LOGW(fmt.Sprintf("flv type:%d not processed", tag.TagType))
	}
	return
}

func (fmp4Creater *FMP4Creater) handleAudioTag(tag *flv.FlvTag) (slice *FMP4Slice) {
	if fmp4Creater.audioInited == false {
		fmp4Creater.audioInited = true
		return fmp4Creater.createAudioInitSeg(tag)
	}
	return fmp4Creater.createAudioSeg(tag)

}

func (fmp4Creater *FMP4Creater) handleVideoTag(tag *flv.FlvTag) (slice *FMP4Slice) {
	if tag.Data[0] != 0x17 && tag.Data[0] != 0x27 {
		logger.LOGW(fmt.Sprintf("%d not support now", int(tag.Data[0])))
		return
	}
	if fmp4Creater.videoInited == false {
		pktType := tag.Data[1]
		if pktType != 0 {
			logger.LOGE("AVC pkt not find")
			return
		}
		fmp4Creater.videoInited = true
		return fmp4Creater.createVideoInitSeg(tag)
	}
	if fmp4Creater.keyframeGeted {
		return fmp4Creater.createVideoSeg(tag)
	}
	if tag.Data[0] == 0x17 && tag.Data[1] == 0x1 {
		fmp4Creater.keyframeGeted = true
		return fmp4Creater.createVideoSeg(tag)
	}

	return
}

func (fmp4Creater *FMP4Creater) createAudioInitSeg(tag *flv.FlvTag) (slice *FMP4Slice) {
	fmp4Creater.audioType = int(tag.Data[0] >> 4)
	//logger.LOGT(tag.Data)
	switch fmp4Creater.audioType {
	case MP3:
		fmp4Creater.audioSampleSize = 1152
		mp3Header, _ := mp3.ParseMP3Header(tag.Data[1:])
		if mp3Header != nil {
			fmp4Creater.audioSampleRate = uint32(mp3Header.SampleRate)
			fmp4Creater.audioSampleDuration = fmp4Creater.audioSampleSize * 1000 / fmp4Creater.audioSampleRate
		}
		//mp3Header.Bitrate
		switch mp3Header.Version {
		case mp3.MPEG_2_5:
			fmp4Creater.audioCodecID = CodecIDMp3MPEG2
		case mp3.MPEG_2:
			fmp4Creater.audioCodecID = CodecIDMp3MPEG2
		case mp3.MPEG_1:
			fmp4Creater.audioCodecID = CodecIDMp3MPEG1
		}
	case AAC:
		fmp4Creater.audioSampleSize = 1024

		//asc:=aac.MP4AudioGetConfig(tag.Data[2:])
		//logger.LOGD(tag.Data[2:])
		//logger.LOGD(asc.Sample_rate)
		//fmp4Creater.audioSampleRate = uint32(asc.Sample_rate)
		//logger.LOGT(asc.Object_type)
		asc := aac.GenerateAudioSpecificConfig(tag.Data[2:])
		logger.LOGD(tag.Data[2:])
		logger.LOGD(asc.SamplingFrequency)
		fmp4Creater.audioSampleRate = uint32(asc.SamplingFrequency)
		logger.LOGT(asc.AudioObjectType)
		logger.LOGT(fmp4Creater.audioSampleRate)
		//		soundRate := ((tag.Data[0] & 0xC) >> 2)
		mpeg4Asc := aac.MP4AudioGetConfig(tag.Data[2:])
		logger.LOGD(mpeg4Asc.Ext_object_type)
		logger.LOGD(mpeg4Asc.Sample_rate)
		if mpeg4Asc.Ext_object_type != 0 && mpeg4Asc.Ext_sample_rate != 0 {
			fmp4Creater.audioSampleRate = uint32(mpeg4Asc.Ext_sample_rate)
		} else {
			fmp4Creater.audioSampleRate = uint32(mpeg4Asc.Sample_rate)
		}

		fmp4Creater.audioSampleDuration = fmp4Creater.audioSampleSize * 1000 / fmp4Creater.audioSampleRate
		logger.LOGT(fmp4Creater.audioSampleDuration)
		logger.LOGT(fmp4Creater.audioSampleRate)
		if mpeg4Asc.Ext_object_type == 0 {
			fmp4Creater.ascData = tag.Data[2:]
			switch mpeg4Asc.Object_type {
			case aac.AAC_Main:
				fmp4Creater.audioCodecID = CodecIDAACMAIN
			case aac.AAC_LC:
				fmp4Creater.audioCodecID = CodecIDAACLC
			case aac.AAC_SSR:
				fmp4Creater.audioCodecID = CodecIDAACSSR
			default:
				fmp4Creater.audioCodecID = CodecIDAAC
			}
			fmp4Creater.audioCodecID = CodecIDAAC
		} else {
			fmp4Creater.ascData = fmp4Creater.aacForHTTP(tag, "")
			objType := (tag.Data[2] & 0xf8) >> 3
			switch objType {
			case aac.AAC_Main:
				fmp4Creater.audioCodecID = CodecIDAACMAIN
			case aac.AAC_LC:
				fmp4Creater.audioCodecID = CodecIDAACLC
			case aac.AAC_SSR:
				fmp4Creater.audioCodecID = CodecIDAACSSR
			default:
				fmp4Creater.audioCodecID = CodecIDAAC
			}
			fmp4Creater.audioCodecID = CodecIDAAC
		}

	default:
		logger.LOGE("unknown audio type")
	}
	logger.LOGT(fmp4Creater.audioSampleDuration)
	slice = &FMP4Slice{}
	slice.Type = flv.FlvTagAudio
	slice.Idx = -1
	segEncoder := &amf.AMF0Encoder{}
	segEncoder.Init()
	//ftyp
	ftyp := &Box{}
	ftyp.Push([]byte("ftyp"))
	ftyp.PushBytes([]byte("isom"))
	ftyp.Push4Bytes(1)
	ftyp.PushBytes([]byte("isom"))
	ftyp.PushBytes([]byte("avc1"))
	ftyp.Pop()
	err := segEncoder.AppendByteArray(ftyp.Flush())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	duration := uint32(0)
	//moov
	moovBox := &Box{}
	moovBox.Push([]byte("moov"))
	//mvhd
	moovBox.Push([]byte("mvhd"))
	moovBox.Push4Bytes(0)        //version
	moovBox.Push4Bytes(0)        //creation_time
	moovBox.Push4Bytes(0)        //modification_time
	moovBox.Push4Bytes(1000)     //time_scale
	moovBox.Push4Bytes(duration) //duration 1s
	//log.Println("duration 0 now")
	moovBox.Push4Bytes(0x00010000) //rate
	moovBox.Push2Bytes(0x0100)     //volume
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push8Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000)
	moovBox.Push4Bytes(0x0) //pre_defined
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	//nextrack id
	moovBox.Push4Bytes(0xffffffff)
	//!mvhd
	moovBox.Pop()
	//trak
	moovBox.Push([]byte("trak"))
	//tkhd
	moovBox.Push([]byte("tkhd"))
	moovBox.Push4Bytes(0x07) //version and flag
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(audioTrack) //track id
	moovBox.Push4Bytes(0)          //reserved
	moovBox.Push4Bytes(duration)   //duration
	//log.Println("duration 0xffffffff")
	moovBox.Push8Bytes(0) //reserved
	moovBox.Push2Bytes(0) //layer
	moovBox.Push2Bytes(0) //alternate_group
	//moovBox.Push2Bytes(0x0100)     //volume
	moovBox.Push2Bytes(0)          //??
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000) //matrix
	moovBox.Push4Bytes(0)          //width
	moovBox.Push4Bytes(0)          //height
	//!tkhd
	moovBox.Pop()
	//mdia
	moovBox.Push([]byte("mdia"))
	//mdhd
	moovBox.Push([]byte("mdhd"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push4Bytes(0) //creation_time
	moovBox.Push4Bytes(0) //modification_time
	//log.Println("maybe to audio sample hz,now use Video time")
	moovBox.Push4Bytes(1000)     //time scale
	moovBox.Push4Bytes(duration) //duration
	//log.Println("duration 0xffffffff")
	if fmp4Creater.audioType == MP3 {
		moovBox.Push4Bytes(0x55c40000)
	} else {
		moovBox.Push4Bytes(0x55c40000) //language und
	}
	//!mdhd
	moovBox.Pop()
	//hdlr
	moovBox.Push([]byte("hdlr"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("soun"))
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("SoundHandler"))
	moovBox.PushByte(0)
	//!hdlr
	moovBox.Pop()
	//minf
	moovBox.Push([]byte("minf"))
	//smhd
	moovBox.Push([]byte("smhd"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push2Bytes(0) //balance
	moovBox.Push2Bytes(0) //reserved
	//!smhd
	moovBox.Pop()
	//dinf
	moovBox.Push([]byte("dinf"))
	//dref
	moovBox.Push([]byte("dref"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(1) //entry_count
	//url
	moovBox.Push([]byte("url "))
	moovBox.Push4Bytes(1)
	//!url
	moovBox.Pop()
	//!dref
	moovBox.Pop()
	//!dinf
	moovBox.Pop()
	//stbl
	moovBox.Push([]byte("stbl"))
	fmp4Creater.stsdA(moovBox, tag) //stsd
	//stts
	moovBox.Push([]byte("stts"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(0) //count
	//!stts
	moovBox.Pop()
	//stsc
	moovBox.Push([]byte("stsc"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsc
	moovBox.Pop()
	//stsz
	moovBox.Push([]byte("stsz"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsz
	moovBox.Pop()
	//stco
	moovBox.Push([]byte("stco"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stco
	moovBox.Pop()
	//!stbl
	moovBox.Pop()
	//!minf
	moovBox.Pop()
	//!mdia
	moovBox.Pop()
	//!trak
	moovBox.Pop()
	//mvex
	moovBox.Push([]byte("mvex"))
	//trex
	moovBox.Push([]byte("trex"))
	moovBox.Push4Bytes(0)          //version and flag
	moovBox.Push4Bytes(audioTrack) //track id
	moovBox.Push4Bytes(1)          //default_sample_description_index
	moovBox.Push4Bytes(0)          //default_sample_duration
	moovBox.Push4Bytes(0)          //default_sample_size
	moovBox.Push4Bytes(0x00010001) //default_sample_flags
	//!trex
	moovBox.Pop()
	//!mvex
	moovBox.Pop()
	//!moov
	moovBox.Pop()

	err = segEncoder.AppendByteArray(moovBox.Flush())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	if saveToFile {

		utils.CreateDirectory("audio")
		fp, err := os.Create("audio/init.mp4")
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer fp.Close()
		fp.Write(slice.Data)
	}

	return
}

func (fmp4Creater *FMP4Creater) createAudioSeg(tag *flv.FlvTag) (slice *FMP4Slice) {
	slice = &FMP4Slice{}
	slice.Type = flv.FlvTagAudio
	slice.Idx = fmp4Creater.audioIdx
	fmp4Creater.audioIdx++
	segEncoder := amf.AMF0Encoder{}
	segEncoder.Init()

	sounBox := &Box{}
	//moof
	sounBox.Push([]byte("moof"))
	//mfhd
	sounBox.Push([]byte("mfhd"))
	sounBox.Push4Bytes(0) //version and flags
	sounBox.Push4Bytes(uint32(fmp4Creater.audioIdx))
	//mfhd
	sounBox.Pop()
	//traf
	sounBox.Push([]byte("traf"))
	//tfhd
	sounBox.Push([]byte("tfhd"))
	sounBox.Push4Bytes(0)          //version and flags,no default-base-is-moof
	sounBox.Push4Bytes(audioTrack) //track
	//!tfhd
	sounBox.Pop()
	//tfdt
	sounBox.Push([]byte("tfdt"))
	sounBox.Push4Bytes(0)
	sounBox.Push4Bytes(fmp4Creater.audioLastTime)
	//!tfdt
	sounBox.Pop()
	//trun

	dataPrefixLength := 1
	if fmp4Creater.audioType == AAC {
		dataPrefixLength = 2
	} else if fmp4Creater.audioType == MP3 {
		dataPrefixLength = 1
	} else {
		logger.LOGE("wth")
	}
	sounBox.Push([]byte("trun"))
	sounBox.Push4Bytes(0xf01) //offset,duration,samplesize,composition
	sounBox.Push4Bytes(1)     //1 sample
	sounBox.Push4Bytes(0x79)  //offset:if base-is-moof ,data offset,from moov begin to mdat data,so now base is first byte

	if tag.Timestamp-fmp4Creater.audioLastTime == 0 {
		//no duration,just a first frame
		sounBox.Push4Bytes(fmp4Creater.audioSampleDuration) //duration
	} else {
		//sounBox.Push4Bytes(fmp4Creater.audioSampleDuration)
		sounBox.Push4Bytes(tag.Timestamp - fmp4Creater.audioLastTime)
	}
	//log.Println(fmt.Sprintf("%d %d", fmp4Creater.audioLastTime, tag.Timestamp))
	//fmp4Creater.audioLastTime += fmp4Creater.audioSampleDuration
	fmp4Creater.audioLastTime = tag.Timestamp

	sounBox.Push4Bytes(uint32(len(tag.Data) - dataPrefixLength)) //sample size
	flags := &FMP4Flags{}
	flags.SampleDependsOn = 1
	sounBox.PushByte(uint8((flags.IsLeading << 2) | flags.SampleDependsOn))
	sounBox.PushByte(uint8((flags.SampleIsDependedOn << 6) | (flags.SampleHasRedundancy << 4) | flags.IsAsync))
	sounBox.Push2Bytes(0)
	sounBox.Push4Bytes(0) //sample_composition_time                                     //sample_composition_time
	//!trun
	sounBox.Pop()
	//sdtp
	sounBox.Push([]byte("sdtp"))
	sounBox.Push4Bytes(0)
	sounBox.PushByte(uint8((flags.IsLeading << 6) | (flags.SampleDependsOn << 4) | (flags.SampleIsDependedOn << 2) | (flags.SampleHasRedundancy)))
	//!sdtp
	sounBox.Pop()
	//!traf
	sounBox.Pop()
	//!moof
	sounBox.Pop()
	err := segEncoder.AppendByteArray(sounBox.Flush())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	//mdat
	err = segEncoder.EncodeInt32(int32(len(tag.Data) - dataPrefixLength + 8))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	err = segEncoder.AppendByteArray([]byte("mdat"))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	err = segEncoder.AppendByteArray(tag.Data[dataPrefixLength:])
	//!mdat
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	if saveToFile {
		fileName := "audio/segment_" + strconv.Itoa(slice.Idx) + ".m4s"
		fp, err := os.Create(fileName)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer fp.Close()
		fp.Write(slice.Data)

	}
	return
}

func (fmp4Creater *FMP4Creater) createVideoInitSeg(tag *flv.FlvTag) (slice *FMP4Slice) {
	slice = &FMP4Slice{}
	slice.Type = flv.FlvTagVideo
	slice.Idx = -1
	segEncoder := amf.AMF0Encoder{}
	segEncoder.Init()
	//ftyp
	ftyp := &Box{}
	ftyp.Push([]byte("ftyp"))
	ftyp.PushBytes([]byte("isom"))
	ftyp.Push4Bytes(1)
	ftyp.PushBytes([]byte("isom"))
	ftyp.PushBytes([]byte("avc1"))
	ftyp.Pop()
	err := segEncoder.AppendByteArray(ftyp.Flush())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	//moov
	moovBox := &Box{}
	moovBox.Push([]byte("moov"))
	//mvhd
	duration := uint32(0)
	moovBox.Push([]byte("mvhd"))
	moovBox.Push4Bytes(0)        //version
	moovBox.Push4Bytes(0)        //creation_time
	moovBox.Push4Bytes(0)        //modification_time
	moovBox.Push4Bytes(1000)     //time_scale
	moovBox.Push4Bytes(duration) //duration 1s
	//log.Println("duration 0xffffffff now")
	moovBox.Push4Bytes(0x00010000) //rate
	moovBox.Push2Bytes(0x0100)     //volume
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push8Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000)
	moovBox.Push4Bytes(0x0) //pre_defined
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	//nextrack id
	moovBox.Push4Bytes(0xffffffff)
	//!mvhd
	moovBox.Pop()
	//trak
	moovBox.Push([]byte("trak"))
	//tkhd
	moovBox.Push([]byte("tkhd"))
	moovBox.Push4Bytes(0x07) //version and flag
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(videoTrack) //track id
	moovBox.Push4Bytes(0)          //reserved
	moovBox.Push4Bytes(duration)   //duration
	//log.Println("duration 0xffffffff")
	moovBox.Push8Bytes(0)          //reserved
	moovBox.Push2Bytes(0)          //layer
	moovBox.Push2Bytes(0)          //alternate_group
	moovBox.Push2Bytes(0)          //volume
	moovBox.Push2Bytes(0)          //reserved
	moovBox.Push4Bytes(0x00010000) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x00010000)
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x0) //matrix
	moovBox.Push4Bytes(0x0)
	moovBox.Push4Bytes(0x40000000) //matrix
	//parse sps ,get w h fps
	tmpTagData := make([]byte, len(tag.Data))
	copy(tmpTagData, tag.Data)
	fmp4Creater.width, fmp4Creater.height, fmp4Creater.fps = h264.ParseSPS(tmpTagData[13:])
	moovBox.Push4Bytes(uint32(fmp4Creater.width << 16))  //width
	moovBox.Push4Bytes(uint32(fmp4Creater.height << 16)) //height
	//!tkhd
	moovBox.Pop()
	//mdia
	moovBox.Push([]byte("mdia"))
	//mdhd
	moovBox.Push([]byte("mdhd"))
	moovBox.Push4Bytes(0)        //version and flag
	moovBox.Push4Bytes(0)        //creation_time
	moovBox.Push4Bytes(0)        //modification_time
	moovBox.Push4Bytes(1000)     //time scale
	moovBox.Push4Bytes(duration) //duration
	//log.Println("duration 0xffffffff")
	moovBox.Push4Bytes(0x55c40000) //language und
	//!mdhd
	moovBox.Pop()
	//hdlr
	moovBox.Push([]byte("hdlr"))
	moovBox.Push4Bytes(0) //version and flag
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("vide"))
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.Push4Bytes(0) //reserved
	moovBox.PushBytes([]byte("VideoHandler"))
	moovBox.PushByte(0)
	//!hdlr
	moovBox.Pop()
	//minf
	moovBox.Push([]byte("minf"))
	//vmhd
	moovBox.Push([]byte("vmhd"))
	moovBox.Push4Bytes(1) //
	moovBox.Push2Bytes(0) //copy
	moovBox.Push2Bytes(0) //opcolor
	moovBox.Push2Bytes(0) //opcolor
	moovBox.Push2Bytes(0) //opcolor
	//!vmhd
	moovBox.Pop()
	//dinf
	moovBox.Push([]byte("dinf"))
	//dref
	moovBox.Push([]byte("dref"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(1) //entry_count
	//url
	moovBox.Push([]byte("url "))
	moovBox.Push4Bytes(1)
	//!url
	moovBox.Pop()
	//!dref
	moovBox.Pop()
	//!dinf
	moovBox.Pop()
	//stbl
	moovBox.Push([]byte("stbl"))
	fmp4Creater.stsdV(moovBox, tag) //stsd
	//stts
	moovBox.Push([]byte("stts"))
	moovBox.Push4Bytes(0) //version
	moovBox.Push4Bytes(0) //count
	//!stts
	moovBox.Pop()
	//stsc
	moovBox.Push([]byte("stsc"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsc
	moovBox.Pop()
	//stsz
	moovBox.Push([]byte("stsz"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stsz
	moovBox.Pop()
	//stco
	moovBox.Push([]byte("stco"))
	moovBox.Push4Bytes(0)
	moovBox.Push4Bytes(0)
	//!stco
	moovBox.Pop()
	//!stbl
	moovBox.Pop()
	//!minf
	moovBox.Pop()
	//!mdia
	moovBox.Pop()
	//!trak
	moovBox.Pop()
	//mvex
	moovBox.Push([]byte("mvex"))
	//trex
	moovBox.Push([]byte("trex"))
	moovBox.Push4Bytes(0)          //version and flag
	moovBox.Push4Bytes(videoTrack) //track id
	moovBox.Push4Bytes(1)          //default_sample_description_index
	moovBox.Push4Bytes(0)          //default_sample_duration
	moovBox.Push4Bytes(0)          //default_sample_size
	moovBox.Push4Bytes(0x00010001) //default_sample_flags
	//!trex
	moovBox.Pop()
	//!mvex
	moovBox.Pop()
	//!moov
	moovBox.Pop()

	err = segEncoder.AppendByteArray(moovBox.Flush())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	if saveToFile {

		utils.CreateDirectory("video")
		fp, err := os.OpenFile("video/init.mp4", os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer fp.Close()
		fp.Write(slice.Data)
	}
	return
}

func (fmp4Creater *FMP4Creater) createVideoSeg(tag *flv.FlvTag) (slice *FMP4Slice) {
	slice = &FMP4Slice{}
	slice.Type = flv.FlvTagVideo
	slice.Idx = fmp4Creater.videoIdx
	fmp4Creater.videoIdx++
	segEncoder := amf.AMF0Encoder{}
	segEncoder.Init()

	flags := &FMP4Flags{}
	flags.IsLeading = 0
	flags.SampleHasRedundancy = 0

	if tag.Data[0] == 0x17 {
		flags.SampleDependsOn = 2
		flags.SampleIsDependedOn = 1
		flags.IsAsync = 0
	} else if tag.Data[0] == 0x27 {
		flags.SampleDependsOn = 1
		flags.SampleIsDependedOn = 0
		flags.IsAsync = 1
	} else {
		logger.LOGE("invalid video")
		return
	}

	videBox := &Box{}
	//moof
	videBox.Push([]byte("moof"))
	//mfhd
	videBox.Push([]byte("mfhd"))
	videBox.Push4Bytes(0) //version and flags
	videBox.Push4Bytes(uint32(fmp4Creater.videoIdx))
	//mfhd
	videBox.Pop()
	//traf
	videBox.Push([]byte("traf"))
	//tfhd
	videBox.Push([]byte("tfhd"))
	videBox.Push4Bytes(0)          //version and flags,no default-base-is-moof
	videBox.Push4Bytes(videoTrack) //track
	//!tfhd
	videBox.Pop()
	//tfdt
	videBox.Push([]byte("tfdt"))
	videBox.Push4Bytes(0)
	videBox.Push4Bytes(tag.Timestamp)
	//!tfdt
	videBox.Pop()
	//trun
	videBox.Push([]byte("trun"))
	videBox.Push4Bytes(0xf01) //offset,duration,samplesize,composition
	videBox.Push4Bytes(1)     //1 sample
	videBox.Push4Bytes(0x79)  //offset:if base-is-moof ,data offset,from moov begin to mdat data,so now base is first byte
	if tag.Timestamp-fmp4Creater.videoLastTime == 0 {
		//no duration,just a first frame
		videBox.Push4Bytes(uint32(1000 / fmp4Creater.fps)) //duration
		//log.Println(uint32(1000 / fmp4Creater.fps))
	} else {
		videBox.Push4Bytes(tag.Timestamp - fmp4Creater.videoLastTime) //duration
		//log.Println(tag.Timestamp - fmp4Creater.videoLastTime)
		//log.Println(fmp4Creater.videoLastTime)
	}
	composition := (uint32(tag.Data[2]) << 16) | (uint32(tag.Data[3]) << 8) | (uint32(tag.Data[4]) << 0)
	//log.Println(fmt.Sprintf("timestame:%d  composition:%d duration:%d", tag.Timestamp, composition, tag.Timestamp-fmp4Creater.videoLastTime))
	fmp4Creater.videoLastTime = tag.Timestamp
	videBox.Push4Bytes(uint32(len(tag.Data) - (5))) //sample size,mdat data size
	videBox.PushByte(uint8((flags.IsLeading << 2) | flags.SampleDependsOn))
	videBox.PushByte(uint8((flags.SampleIsDependedOn << 6) | (flags.SampleHasRedundancy << 4) | flags.IsAsync))
	videBox.Push2Bytes(0)
	videBox.Push4Bytes(composition) //sample_composition_time
	//!trun
	videBox.Pop()
	//sdtp
	videBox.Push([]byte("sdtp"))
	videBox.Push4Bytes(0)
	videBox.PushByte(uint8((flags.IsLeading << 6) | (flags.SampleDependsOn << 4) | (flags.SampleIsDependedOn << 2) | (flags.SampleHasRedundancy)))
	//!sdtp
	videBox.Pop()
	//!traf
	videBox.Pop()
	//!moof
	videBox.Pop()
	err := segEncoder.AppendByteArray(videBox.Flush())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	//mdat
	err = segEncoder.EncodeInt32(int32(len(tag.Data) - (5) + 8))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	err = segEncoder.AppendByteArray([]byte("mdat"))
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	err = segEncoder.AppendByteArray(tag.Data[5:])
	//!mdat
	slice.Data, err = segEncoder.GetData()
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	if saveToFile {
		fileName := "video/segment_" + strconv.Itoa(slice.Idx) + ".m4s"
		fp, err := os.Create(fileName)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer fp.Close()
		fp.Write(slice.Data)
	}
	return
}

func (fmp4Creater *FMP4Creater) stsdV(box *Box, tag *flv.FlvTag) {
	//stsd
	box.Push([]byte("stsd"))
	box.Push4Bytes(0)
	box.Push4Bytes(1)
	//avc1
	box.Push([]byte("avc1"))
	box.Push4Bytes(0)
	box.Push2Bytes(0)
	box.Push2Bytes(1)
	box.Push2Bytes(0)
	box.Push2Bytes(0)
	box.Push4Bytes(0)
	box.Push4Bytes(0)
	box.Push4Bytes(0)
	box.Push2Bytes(uint16(fmp4Creater.width))
	box.Push2Bytes(uint16(fmp4Creater.height))
	box.Push4Bytes(0x00480000)
	box.Push4Bytes(0x00480000)
	box.Push4Bytes(0)
	box.Push2Bytes(1)
	box.PushByte(uint8(len("fmp4 coding")))
	box.PushBytes([]byte("fmp4 coding"))
	spaceEnd := make([]byte, 32-1-len("fmp4 coding"))
	box.PushBytes(spaceEnd)
	box.Push2Bytes(0x18)
	box.Push2Bytes(0xffff)
	//avcC
	box.Push([]byte("avcC"))
	box.PushBytes(tag.Data[5:])
	//!avcC
	box.Pop()
	//!avc1
	box.Pop()
	//!stsd
	box.Pop()
	return
}

func (fmp4Creater *FMP4Creater) stsdA(box *Box, tag *flv.FlvTag) {
	//stsd
	box.Push([]byte("stsd"))
	box.Push4Bytes(0)
	box.Push4Bytes(1)
	//mp4a
	box.Push([]byte("mp4a"))
	box.Push4Bytes(0)  //reserved
	box.Push2Bytes(0)  //reserved
	box.Push2Bytes(1)  //data reference index
	box.Push8Bytes(0)  //reserved int32[2]
	box.Push2Bytes(2)  //channel count
	box.Push2Bytes(16) //sample size
	box.Push2Bytes(0)  //pre defined
	box.Push2Bytes(0)  //reserved
	logger.LOGT(fmp4Creater.audioSampleRate)
	box.Push4Bytes(fmp4Creater.audioSampleRate << 16) //samplerate
	//esds
	box.Push([]byte("esds"))
	box.Push4Bytes(0)           //version and flag
	box.PushByte(MP4ESDescrTag) //tag MP4ESDescrTag
	esd := &Box{}
	esd.Push2Bytes(1) //ES ID
	esd.PushByte(0)   // stream priority (0-32)

	esd.PushByte(MP4DecConfigDescrTag) //MP4DecConfigDescrTag tag
	esdDesc := &Box{}
	switch fmp4Creater.audioType { //object type indication
	case MP3:
		esdDesc.PushByte(byte(fmp4Creater.audioCodecID))
	case AAC:
		esdDesc.PushByte(byte(fmp4Creater.audioCodecID))
	default:
		esdDesc.PushByte(0x40)
		logger.LOGT(fmt.Sprintf("audio type %d not support", fmp4Creater.audioType))
	}
	esdDesc.PushByte(0x15) //固定15  streamType upstream reserved
	esdDesc.PushByte(0)    //24位buffer size db
	esdDesc.Push2Bytes(0)  //24位补充
	esdDesc.Push4Bytes(0)  //max bitrate
	esdDesc.Push4Bytes(0)  //avg bitrate
	if fmp4Creater.audioType == AAC {
		esdDesc.PushByte(MP4DecSpecificDescrTag) //MP4DecSpecificDescrTag
		if len(tag.Data) >= 2 {
			//esdDesc.PushByte(byte(len(tag.Data) - 2))
			//esdDesc.PushBytes(tag.Data[2:])
			//ascData := fmp4Creater.aacForHTTP(tag, "")
			//log.Println(ascData)
			//esdDesc.PushByte(byte(len(ascData)))
			//esdDesc.PushBytes(ascData)
			esdDesc.PushByte(byte(len(fmp4Creater.ascData)))
			esdDesc.PushBytes(fmp4Creater.ascData)
		}

	}
	esdDescData := esdDesc.Flush()
	esd.PushByte(byte(len(esdDescData)))
	esd.PushBytes(esdDescData)
	//0x 06 01 02 or 0x06 0x80 0x80 0x80 01 02
	esd.PushByte(0x06) //SLConfigDescrTag
	esd.PushByte(0x01) //length field
	esd.PushByte(0x02) //predefined 0x02 reserved for use int mp4 faile
	esdData := esd.Flush()
	box.PushByte(byte(len(esdData)))
	box.PushBytes(esdData)
	//!esds
	box.Pop()
	//!mp4a
	box.Pop()
	//!stsd
	box.Pop()
	return
}

func (fmp4Creater *FMP4Creater) aacForHTTP(tag *flv.FlvTag, useragent string) (cfg []byte) {
	asc := aac.GenerateAudioSpecificConfig(tag.Data[2:])

	if len(useragent) > 0 {
		useragent = strings.ToLower(useragent)
	}
	switch useragent {
	case UserAgentFireFox:
		if asc.SamplingFrequencyIndex >= aac.AAC_SCALABLE {
			asc.AudioObjectType = aac.AAC_HE_OR_SBR
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex - 3
			cfg = make([]byte, 4)
		} else {
			asc.AudioObjectType = aac.AAC_LC
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
			cfg = make([]byte, 2)
		}
	case UserAgentAndroid:
		asc.AudioObjectType = aac.AAC_LC
		asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
		cfg = make([]byte, 2)
	default:
		asc.AudioObjectType = aac.AAC_HE_OR_SBR
		asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
		cfg = make([]byte, 4)
		if asc.SamplingFrequencyIndex >= aac.AAC_SCALABLE {
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex - 3
		} else if asc.ChannelConfiguration == 1 {
			asc.AudioObjectType = aac.AAC_LC
			asc.ExtensionSamplingIndex = asc.SamplingFrequencyIndex
			cfg = make([]byte, 2)
		}
	}
	cfg[0] = (asc.AudioObjectType << 3)
	cfg[0] |= ((asc.SamplingFrequencyIndex & 0xf) >> 1)
	cfg[1] = ((asc.SamplingFrequencyIndex & 0xf) << 7)
	cfg[1] |= ((asc.ChannelConfiguration & 0xf) << 3)
	if asc.AudioObjectType == aac.AAC_HE_OR_SBR {
		cfg[1] |= ((asc.ExtensionSamplingIndex & 0xf) >> 1)
		cfg[2] = ((asc.ExtensionSamplingIndex & 1) << 7)
		cfg[2] |= (2 << 2)
		cfg[3] = 0
	}
	return
}

func (fmp4Creater *FMP4Creater) stsdA1(box *Box, tag *flv.FlvTag) {
	//stsd
	box.Push([]byte("stsd"))
	box.Push4Bytes(0)
	box.Push4Bytes(1)
	//mp4a
	box.Push([]byte("mp4a"))
	box.Push4Bytes(0)  //reserved
	box.Push2Bytes(0)  //reserved
	box.Push2Bytes(1)  //data reference index
	box.Push8Bytes(0)  //reserved int32[2]
	box.Push2Bytes(2)  //channel count
	box.Push2Bytes(16) //sample size
	box.Push2Bytes(0)  //pre defined
	box.Push2Bytes(0)  //reserved
	logger.LOGT(fmp4Creater.audioSampleRate)
	box.Push4Bytes(fmp4Creater.audioSampleRate << 16) //samplerate
	//esds
	box.Push([]byte("esds"))
	box.Push4Bytes(0) //version and flag
	box.PushByte(3)   //tag
	box.PushByte(0x80)
	box.PushByte(0x80)
	box.PushByte(0x80)
	esd := &Box{}
	esd.Push2Bytes(0x02) //ES ID
	esd.PushByte(0)      //1:streamDependenceFlag=0  1:URL_Flag=0 1:OCRstreamFlag=0 5:streamPrority=0
	esd.PushByte(4)      //DecoderConfigDescriptor tag
	box.PushByte(0x80)
	box.PushByte(0x80)
	box.PushByte(0x80)
	esdDesc := &Box{}
	switch fmp4Creater.audioType { //object type indication
	case MP3:
		esdDesc.PushByte(0x6b)
	case AAC:
		esdDesc.PushByte(0x40)
	default:
		esdDesc.PushByte(0x40)
		logger.LOGT(fmt.Sprintf("audio type %d not support", fmp4Creater.audioType))
	}
	esdDesc.PushByte(0x15)      //固定15  streamType upstream reserved
	esdDesc.PushByte(0)         //24位buffer size db
	esdDesc.Push2Bytes(0)       //24位补充
	esdDesc.Push4Bytes(0x20000) //max bitrate
	esdDesc.Push4Bytes(0)       //avg bitrate
	if fmp4Creater.audioType == AAC {
		esdDesc.PushByte(0x05)
		if len(tag.Data) >= 2 {
			//esdDesc.PushByte(byte(len(tag.Data) - 2))
			//esdDesc.PushBytes(tag.Data[2:])
			//ascData := fmp4Creater.aacForHTTP(tag, "")
			//log.Println(ascData)
			//esdDesc.PushByte(byte(len(ascData)))
			//esdDesc.PushBytes(ascData)
			esdDesc.PushByte(byte(len(fmp4Creater.ascData)))
			esdDesc.PushBytes(fmp4Creater.ascData)
		}

	}
	esdDescData := esdDesc.Flush()
	esd.PushByte(byte(len(esdDescData)))
	esd.PushBytes(esdDescData)
	esd.PushByte(0x06) //SLConfigDescrTag
	box.PushByte(0x80)
	box.PushByte(0x80)
	box.PushByte(0x80)
	esd.PushByte(0x01) //length field
	esd.PushByte(0x02) //predefined 0x02 reserved for use int mp4 faile
	esdData := esd.Flush()
	box.PushByte(byte(len(esdData)))
	box.PushBytes(esdData)
	//!esds
	box.Pop()
	//!mp4a
	box.Pop()
	//!stsd
	box.Pop()
	return
}
