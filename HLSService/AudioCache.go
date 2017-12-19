package HLSService

import (
	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/mediaTypes/aac"
	"github.com/use-go/websocketStreamServer/mediaTypes/flv"
)

//may be aac or mp3 or other
//first support aac
type audioCache struct {
	audioType int
	aacCache  *aac.AACCreater
}

func (audio *audioCache) Init(tag *flv.FlvTag) {
	audio.audioType = int((tag.Data[0] >> 4) & 0xf)
	if audio.audioType == flv.SoundFormat_AAC {
		audio.aacCache = &aac.AACCreater{}
		audio.aacCache.Init(tag.Data[2:])
	} else {
		logger.LOGW("sound fmt not processed")
	}
}

func (audio *audioCache) AddTag(tag *flv.FlvTag) {
	if audio.audioType == flv.SoundFormat_AAC {
		audio.aacCache.Add(tag.Data[2:])
	} else {
		//		logger.LOGW("sound fmt not processed")
	}
}

func (audio *audioCache) Flush() (data []byte) {
	if audio.audioType == flv.SoundFormat_AAC {
		data = audio.aacCache.Flush()
		return data
	} else {
		return nil
	}
	return nil
}
