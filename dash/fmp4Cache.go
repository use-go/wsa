package dash

import (
	"container/list"
	"errors"
	"sync"
)

//FMP4Cache description Struct
type FMP4Cache struct {
	audioHeader []byte
	videoHeader []byte
	cacheSize   int
	muxAudio    sync.RWMutex
	audioCache  map[int64][]byte
	audioKeys   *list.List
	muxVideo    sync.RWMutex
	videoCache  map[int64][]byte
	videoKeys   *list.List
}

//NewFMP4Cache to create new Cache
func NewFMP4Cache(cacheSize int) (cache *FMP4Cache) {
	cache = &FMP4Cache{}
	cache.cacheSize = cacheSize
	cache.audioCache = make(map[int64][]byte)
	cache.videoCache = make(map[int64][]byte)
	cache.audioKeys = list.New()
	cache.videoKeys = list.New()
	return
}

func (fmp4Cache *FMP4Cache) VideoHeaderGenerated(videoHeader []byte) {
	fmp4Cache.muxVideo.Lock()
	defer fmp4Cache.muxVideo.Unlock()
	fmp4Cache.videoHeader = make([]byte, len(videoHeader))
	copy(fmp4Cache.videoHeader, videoHeader)
}
func (fmp4Cache *FMP4Cache) VideoSegmentGenerated(videoSegment []byte, timestamp int64, duration int) {
	fmp4Cache.muxVideo.Lock()
	defer fmp4Cache.muxVideo.Unlock()
	fmp4Cache.videoCache[timestamp] = videoSegment
	fmp4Cache.videoKeys.PushBack(timestamp)
	if fmp4Cache.videoKeys.Len() > fmp4Cache.cacheSize {
		key := fmp4Cache.videoKeys.Front().Value.(int64)
		delete(fmp4Cache.videoCache, key)
		fmp4Cache.videoKeys.Remove(fmp4Cache.videoKeys.Front())
	}
}
func (fmp4Cache *FMP4Cache) AudioHeaderGenerated(audioHeader []byte) {
	fmp4Cache.muxAudio.Lock()
	defer fmp4Cache.muxAudio.Unlock()
	fmp4Cache.audioHeader = make([]byte, len(audioHeader))
	copy(fmp4Cache.audioHeader, audioHeader)
}
func (fmp4Cache *FMP4Cache) AudioSegmentGenerated(audioSegment []byte, timestamp int64, duration int) {
	fmp4Cache.muxAudio.Lock()
	defer fmp4Cache.muxAudio.Unlock()
	fmp4Cache.audioCache[timestamp] = audioSegment
	fmp4Cache.audioKeys.PushBack(timestamp)
	if fmp4Cache.audioKeys.Len() > fmp4Cache.cacheSize {
		key := fmp4Cache.audioKeys.Front().Value.(int64)
		delete(fmp4Cache.audioCache, key)
		fmp4Cache.audioKeys.Remove(fmp4Cache.audioKeys.Front())
	}
}

func (fmp4Cache *FMP4Cache) GetAudioHeader() (data []byte, err error) {
	fmp4Cache.muxAudio.RLock()
	defer fmp4Cache.muxAudio.RUnlock()
	data = fmp4Cache.audioHeader
	if nil == data || len(data) == 0 {
		err = errors.New("no audio header now")
	}
	return
}

func (fmp4Cache *FMP4Cache) GetAudioSegment(timestamp int64) (seg []byte, err error) {
	fmp4Cache.muxAudio.RLock()
	defer fmp4Cache.muxAudio.RUnlock()
	seg = fmp4Cache.audioCache[timestamp]
	if nil == seg {
		err = errors.New("audio segment not found")
	}
	return
}

func (fmp4Cache *FMP4Cache) GetVideoHeader() (data []byte, err error) {
	fmp4Cache.muxVideo.RLock()
	defer fmp4Cache.muxVideo.RUnlock()
	data = fmp4Cache.videoHeader
	if nil == data || len(data) == 0 {
		err = errors.New("no video header")
	}
	return
}

func (fmp4Cache *FMP4Cache) GetVideoSegment(timestamp int64) (seg []byte, err error) {
	fmp4Cache.muxVideo.RLock()
	defer fmp4Cache.muxVideo.RUnlock()
	seg = fmp4Cache.videoCache[timestamp]
	if nil == seg {
		err = errors.New("video segment not found")
	}
	return
}
