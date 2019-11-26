package rtmp

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediatype/flv"
	"github.com/use-go/websocket-streamserver/wssapi"
)

const (
	play_idle = iota
	play_playing
	play_paused
)

type rtmpPlayer struct {
	parent         wssapi.MsgHandler
	playStatus     int
	mutexStatus    sync.RWMutex
	playing        bool //true for thread send playing data
	waitPlaying    *sync.WaitGroup
	mutexCache     sync.RWMutex
	cache          *list.List
	audioHeader    *flv.FlvTag
	videoHeader    *flv.FlvTag
	metadata       *flv.FlvTag
	keyFrameWrited bool
	beginTime      uint32
	startTime      float32
	duration       float32
	reset          bool
	rtmp           *RTMP
}

func (rtmpplayer *rtmpPlayer) Init(msg *wssapi.Msg) (err error) {
	rtmpplayer.playStatus = play_idle
	rtmpplayer.waitPlaying = new(sync.WaitGroup)
	rtmpplayer.rtmp = msg.Param1.(*RTMP)
	rtmpplayer.resetCache()
	return
}

func (rtmpplayer *rtmpPlayer) Start(msg *wssapi.Msg) (err error) {
	rtmpplayer.startPlay()
	return
}

func (rtmpplayer *rtmpPlayer) Stop(msg *wssapi.Msg) (err error) {
	rtmpplayer.stopPlay()
	return
}

func (rtmpplayer *rtmpPlayer) GetType() string {
	return ""
}

func (rtmpplayer *rtmpPlayer) HandleTask(task *wssapi.Task) (err error) {
	return
}

func (rtmpplayer *rtmpPlayer) ProcessMessage(msg *wssapi.Msg) (err error) {
	return
}

func (rtmpplayer *rtmpPlayer) startPlay() {
	rtmpplayer.mutexStatus.Lock()
	defer rtmpplayer.mutexStatus.Unlock()
	switch rtmpplayer.playStatus {
	case play_idle:
		go rtmpplayer.threadPlay()
		rtmpplayer.playStatus = play_playing
	case play_paused:
		logger.LOGE("pause not processed")
		return
	case play_playing:
		return
	}

}

func (rtmpplayer *rtmpPlayer) stopPlay() {
	rtmpplayer.mutexStatus.Lock()
	defer rtmpplayer.mutexStatus.Unlock()
	switch rtmpplayer.playStatus {
	case play_idle:
		return
	case play_paused:
		logger.LOGE("pause not processed")
		return
	case play_playing:
		//stop play thread
		//reset
		rtmpplayer.playing = false
		rtmpplayer.waitPlaying.Wait()
		rtmpplayer.playStatus = play_idle
		rtmpplayer.resetCache()
	}

}

func (rtmpplayer *rtmpPlayer) pause() {

}

func (rtmpplayer *rtmpPlayer) IsPlaying() bool {
	return rtmpplayer.playStatus == play_playing
}

func (rtmpplayer *rtmpPlayer) appendFlvTag(tag *flv.FlvTag) (err error) {
	rtmpplayer.mutexStatus.RLock()
	defer rtmpplayer.mutexStatus.RUnlock()
	if rtmpplayer.playStatus != play_playing {
		err = errors.New("not playing ,can not recv mediaData")
		logger.LOGE(err.Error())
		return
	}
	tag = tag.Copy()
	//	if rtmpplayer.beginTime == 0 && tag.Timestamp > 0 {
	//		rtmpplayer.beginTime = tag.Timestamp
	//	}
	//	tag.Timestamp -= rtmpplayer.beginTime
	if false == rtmpplayer.keyFrameWrited && tag.TagType == flv.FlvTagVideo {
		if rtmpplayer.videoHeader == nil {
			rtmpplayer.videoHeader = tag
		} else {
			if (tag.Data[0] >> 4) == 1 {
				rtmpplayer.keyFrameWrited = true
			} else {
				return
			}
		}

	}
	rtmpplayer.mutexCache.Lock()
	defer rtmpplayer.mutexCache.Unlock()
	rtmpplayer.cache.PushBack(tag)
	return
}

func (rtmpplayer *rtmpPlayer) setPlayParams(path string, startTime, duration int, reset bool) bool {
	rtmpplayer.mutexStatus.RLock()
	defer rtmpplayer.mutexStatus.RUnlock()
	if rtmpplayer.playStatus != play_idle {
		return false
	}
	return true
}

func (rtmpplayer *rtmpPlayer) resetCache() {
	rtmpplayer.audioHeader = nil
	rtmpplayer.videoHeader = nil
	rtmpplayer.metadata = nil
	rtmpplayer.beginTime = 0
	rtmpplayer.keyFrameWrited = false
	rtmpplayer.mutexCache.Lock()
	defer rtmpplayer.mutexCache.Unlock()
	rtmpplayer.cache = list.New()
}

func (rtmpplayer *rtmpPlayer) threadPlay() {
	rtmpplayer.playing = true
	rtmpplayer.waitPlaying.Add(1)
	defer func() {
		rtmpplayer.waitPlaying.Done()
		rtmpplayer.sendPlayEnds()
		rtmpplayer.playStatus = play_idle
	}()
	rtmpplayer.sendPlayStarts()

	for rtmpplayer.playing == true {
		rtmpplayer.mutexCache.Lock()
		if rtmpplayer.cache == nil || rtmpplayer.cache.Len() == 0 {
			rtmpplayer.mutexCache.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if rtmpplayer.cache.Len() > serviceConfig.CacheCount {
			rtmpplayer.mutexCache.Unlock()
			//bw not enough
			rtmpplayer.rtmp.CmdStatus("warning", "NetStream.Play.InsufficientBW",
				"instufficient bw", rtmpplayer.rtmp.Link.Path, 0, RTMP_channel_Invoke)
			//shutdown
			return
		}
		tag := rtmpplayer.cache.Front().Value.(*flv.FlvTag)
		rtmpplayer.cache.Remove(rtmpplayer.cache.Front())
		rtmpplayer.mutexCache.Unlock()
		//时间错误

		err := rtmpplayer.rtmp.SendPacket(FlvTagToRTMPPacket(tag), false)
		if err != nil {
			logger.LOGE("send rtmp packet failed in play")
			return
		}
	}
}

func (rtmpplayer *rtmpPlayer) sendPlayEnds() {

	err := rtmpplayer.rtmp.SendCtrl(RTMP_CTRL_streamEof, 1, 0)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	err = rtmpplayer.rtmp.FCUnpublish()
	if err != nil {
		logger.LOGE("FCUnpublish failed:" + err.Error())
		return
	}

	err = rtmpplayer.rtmp.CmdStatus("status", "NetStream.Play.UnpublishNotify",
		fmt.Sprintf("%s is unpublished", rtmpplayer.rtmp.Link.Path),
		rtmpplayer.rtmp.Link.Path, 0, RTMP_channel_Invoke)

	if err != nil {
		logger.LOGE(err.Error())
		return
	}
}

func (rtmpplayer *rtmpPlayer) sendPlayStarts() {
	err := rtmpplayer.rtmp.CmdStatus("status", "NetStream.Play.PublishNotify",
		fmt.Sprintf("%s is now unpublished", rtmpplayer.rtmp.Link.Path),
		rtmpplayer.rtmp.Link.Path,
		0, RTMP_channel_Invoke)
	if err != nil {
		logger.LOGE(err.Error())
	}
	err = rtmpplayer.rtmp.SendCtrl(RTMP_CTRL_streamBegin, 1, 0)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	if true == rtmpplayer.reset {
		err = rtmpplayer.rtmp.CmdStatus("status", "NetStream.Play.Reset",
			fmt.Sprintf("Playing and resetting %s", rtmpplayer.rtmp.Link.Path),
			rtmpplayer.rtmp.Link.Path, 0, RTMP_channel_Invoke)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
	}

	err = rtmpplayer.rtmp.CmdStatus("status", "NetStream.Play.Start",
		fmt.Sprintf("Started playing %s", rtmpplayer.rtmp.Link.Path), rtmpplayer.rtmp.Link.Path, 0, RTMP_channel_Invoke)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	logger.LOGT("start playing")
}

func (rtmpplayer *rtmpPlayer) SetParent(parent wssapi.MsgHandler) {
	rtmpplayer.parent = parent
}
