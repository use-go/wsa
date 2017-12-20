package DASHService

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/use-go/websocketStreamServer/logger"

	"github.com/use-go/websocketStreamServer/mediaTypes/h264"

	"github.com/use-go/websocketStreamServer/mediaTypes/flv"

	"github.com/panda-media/muxer-fmp4/codec/H264"
	"github.com/panda-media/muxer-fmp4/dashSlicer"
	"github.com/use-go/websocketStreamServer/events/eStreamerEvent"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

//DASHSource description
type DASHSource struct {
	clientID   string
	chSvr      chan bool
	chValid    bool
	streamName string
	sinkAdded  bool
	inSvr      bool

	slicer            *dashSlicer.DASHSlicer
	mediaReceiver     *FMP4Cache
	appendedAACHeader bool
	appendedKeyFrame  bool
	audioHeader       *flv.FlvTag
	videoHeader       *flv.FlvTag
}

func (dashSource *DASHSource) serveHTTP(reqType, param string, w http.ResponseWriter, req *http.Request) {
	switch reqType {
	case MpdPREFIX:
		dashSource.serveMPD(param, w, req)
	case VideoPREFIX:
		//logger.LOGD(param)
		dashSource.serveVideo(param, w, req)
	case AudioPREFIX:
		//logger.LOGD(param)
		dashSource.serveAudio(param, w, req)
	}
}

func (dashSource *DASHSource) serveMPD(param string, w http.ResponseWriter, req *http.Request) {

	if nil == dashSource.slicer {
		w.WriteHeader(404)
		return
	}
	mpd, err := dashSource.slicer.GetMPD()
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	//mpd,err=wssAPI.ReadFileAll("mpd/taotao.mpd")
	//if err!=nil{
	//	logger.LOGE(err.Error())
	//	return
	//}
	w.Header().Set("Content-Type", "application/dash+xml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(mpd)

}

func (dashSource *DASHSource) serveVideo(param string, w http.ResponseWriter, req *http.Request) {
	var data []byte
	var err error
	if strings.Contains(param, "_init_") {
		data, err = dashSource.mediaReceiver.GetVideoHeader()
	} else {
		id := int64(0)
		fmt.Sscanf(param, "video_video0_%d_mp4.m4s", &id)
		data, err = dashSource.mediaReceiver.GetVideoSegment(id)
	}
	if err != nil {
		w.WriteHeader(404)
		return
	}

	w.Write(data)
}

func (dashSource *DASHSource) serveAudio(param string, w http.ResponseWriter, req *http.Request) {
	var data []byte
	var err error
	if strings.Contains(param, "_init_") {
		data, err = dashSource.mediaReceiver.GetAudioHeader()
	} else {
		id := int64(0)
		fmt.Sscanf(param, "audio_audio0_%d_mp4.m4s", &id)
		data, err = dashSource.mediaReceiver.GetAudioSegment(id)
	}
	if err != nil {
		w.WriteHeader(404)
		return
	}
	w.Write(data)
}

func (dashSource *DASHSource) Init(msg *wssAPI.Msg) (err error) {

	//dashSource.slicer=dashSlicer.NEWSlicer(true,2000,10000,5)

	var ok bool
	dashSource.streamName, ok = msg.Param1.(string)
	if false == ok {
		return errors.New("invalid param init DASH source")
	}
	dashSource.chSvr, ok = msg.Param2.(chan bool)
	if false == ok {
		dashSource.chValid = false
		return errors.New("invalid param init hls source")
	}
	dashSource.chValid = true
	dashSource.clientID = wssAPI.GenerateGUID()
	taskAddSink := &eStreamerEvent.EveAddSink{
		StreamName: dashSource.streamName,
		SinkId:     dashSource.clientID,
		Sinker:     dashSource}

	wssAPI.HandleTask(taskAddSink)

	return
}

// Start action
func (dashSource *DASHSource) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (dashSource *DASHSource) Stop(msg *wssAPI.Msg) (err error) {
	if dashSource.sinkAdded {
		logger.LOGD("del sink:" + dashSource.clientID)
		taskDelSink := &eStreamerEvent.EveDelSink{}
		taskDelSink.StreamName = dashSource.streamName
		taskDelSink.SinkId = dashSource.clientID
		go wssAPI.HandleTask(taskDelSink)
		dashSource.sinkAdded = false
		logger.LOGT("del sinker:" + dashSource.clientID)
	}
	if dashSource.inSvr {
		service.Del(dashSource.streamName, dashSource.clientID)
	}
	if dashSource.chValid {
		close(dashSource.chSvr)
		dashSource.chValid = false
	}
	return
}

func (dashSource *DASHSource) GetType() string {
	return ""
}

func (dashSource *DASHSource) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (dashSource *DASHSource) ProcessMessage(msg *wssAPI.Msg) (err error) {
	switch msg.Type {
	case wssAPI.MSG_GetSource_NOTIFY:
		if dashSource.chValid {
			dashSource.chSvr <- true
			dashSource.inSvr = true
			close(dashSource.chSvr)
			dashSource.chValid = false
		}
		dashSource.sinkAdded = true
	case wssAPI.MSG_GetSource_Failed:
		dashSource.Stop(nil)
	case wssAPI.MSG_PLAY_START:
		dashSource.sinkAdded = true
	case wssAPI.MSG_PLAY_STOP:
		dashSource.Stop(nil)
	case wssAPI.MSG_FLV_TAG:
		dashSource.addFlvTag(msg.Param1.(*flv.FlvTag))
	}
	return
}

func (dashSource *DASHSource) createSlicer() (err error) {
	var fps int
	if nil != dashSource.videoHeader {
		if dashSource.videoHeader.Data[0] == 0x17 && dashSource.videoHeader.Data[1] == 0 {
			avc, err := H264.DecodeAVC(dashSource.videoHeader.Data[5:])
			if err != nil {
				logger.LOGE(err.Error())
				return err
			}
			for e := avc.SPS.Front(); e != nil; e = e.Next() {
				_, _, fps = h264.ParseSPS(e.Value.([]byte))
				break
			}
		}
		dashSource.mediaReceiver = NewFMP4Cache(5)
		dashSource.slicer, err = dashSlicer.NEWSlicer(fps, 1000, 1000, 1000, 9000, 5, dashSource.mediaReceiver)
		if err != nil {
			logger.LOGE(err.Error())
			return err
		}
	} else {
		err = errors.New("invalid video  header")
		return
	}

	if nil != dashSource.audioHeader {
		dashSource.slicer.AddAACFrame(dashSource.audioHeader.Data[2:], int64(dashSource.audioHeader.Timestamp))
	}
	tag := dashSource.videoHeader.Copy()
	avc, err := H264.DecodeAVC(tag.Data[5:])
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	for e := avc.SPS.Front(); e != nil; e = e.Next() {
		nal := make([]byte, 3+len(e.Value.([]byte)))
		nal[0] = 0
		nal[1] = 0
		nal[2] = 1
		copy(nal[3:], e.Value.([]byte))
		dashSource.slicer.AddH264Nals(nal, int64(tag.Timestamp))
	}
	for e := avc.PPS.Front(); e != nil; e = e.Next() {
		nal := make([]byte, 3+len(e.Value.([]byte)))
		nal[0] = 0
		nal[1] = 0
		nal[2] = 1
		copy(nal[3:], e.Value.([]byte))
		dashSource.slicer.AddH264Nals(nal, int64(tag.Timestamp))
	}
	return
}

func (dashSource *DASHSource) addFlvTag(tag *flv.FlvTag) {
	switch tag.TagType {
	case flv.FLV_TAG_Audio:
		if nil == dashSource.audioHeader {
			dashSource.audioHeader = tag.Copy()
			return
		} else if dashSource.slicer == nil {
			dashSource.createSlicer()
		}
		if false == dashSource.appendedAACHeader {
			logger.LOGD("AAC")
			dashSource.slicer.AddAACFrame(tag.Data[2:], int64(tag.Timestamp))
			dashSource.appendedAACHeader = true
		} else {
			if dashSource.appendedKeyFrame {
				dashSource.slicer.AddAACFrame(tag.Data[2:], int64(tag.Timestamp))
			}
		}
	case flv.FLV_TAG_Video:
		if nil == dashSource.videoHeader {
			dashSource.videoHeader = tag.Copy()
			return
		} else if nil == dashSource.slicer {
			dashSource.createSlicer()
		}
		cur := 5
		for cur < len(tag.Data) {
			size := int(tag.Data[cur]) << 24
			size |= int(tag.Data[cur+1]) << 16
			size |= int(tag.Data[cur+2]) << 8
			size |= int(tag.Data[cur+3]) << 0
			cur += 4
			nal := make([]byte, 3+size)
			nal[0] = 0
			nal[1] = 0
			nal[2] = 1
			copy(nal[3:], tag.Data[cur:cur+size])
			if false == dashSource.appendedKeyFrame {
				if tag.Data[cur]&0x1f == H264.NAL_IDR_SLICE {
					dashSource.appendedKeyFrame = true
				} else {
					cur += size
					continue
				}
			}
			//dashSource.slicer.AddH264Nals(nal,int64(tag.Timestamp))
			cur += size
		}
		compositionTime := int(tag.Data[2]) << 16
		compositionTime |= int(tag.Data[3]) << 8
		compositionTime |= int(tag.Data[4]) << 0
		dashSource.slicer.AddH264Frame(tag.Data[5:], int64(tag.Timestamp), compositionTime)
	}
}
