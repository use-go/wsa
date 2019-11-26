package hls

import (
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediatype/aac"
	"github.com/use-go/websocket-streamserver/mediatype/flv"
)

//may be aac or mp3 or other
//first support aac
type audioCache struct {
	audioType int
	aacCache  *aac.AACCreater
}

func (audio *audioCache) Init(tag *flv.FlvTag) {
	audio.audioType = int((tag.Data[0] >> 4) & 0xf)
	if audio.audioType == flv.SoundFormatAAC {
		audio.aacCache = &aac.AACCreater{}
		audio.aacCache.Init(tag.Data[2:])
	} else {
		logger.LOGW("sound fmt not processed")
	}
}

func (audio *audioCache) AddTag(tag *flv.FlvTag) {
	if audio.audioType == flv.SoundFormatAAC {
		audio.aacCache.Add(tag.Data[2:])
	} else {
		//		logger.LOGW("sound fmt not processed")
	}
}

func (audio *audioCache) Flush() (data []byte) {
	if audio.audioType == flv.SoundFormatAAC {
		data = audio.aacCache.Flush()
		return data
	} else {
		return nil
	}
	return nil
}
