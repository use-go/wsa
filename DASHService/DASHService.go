package DASHService

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/use-go/websocketStreamServer/HTTPMUX"
	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

//http://addr/DASH/streamName/req
const (
	MpdPREFIX   = "mpd"
	VideoPREFIX = "video"
	AudioPREFIX = "audio"
)

//DASHService struct
type DASHService struct {
	sources   map[string]*DASHSource
	muxSource sync.RWMutex
}

//DASHConfig struc
type DASHConfig struct {
	Port  int    `json:"Port"`
	Route string `json:"Route"`
}

var service *DASHService
var serviceConfig DASHConfig

func (dashService *DASHService) Init(msg *wssAPI.Msg) (err error) {
	defer func() {
		if nil != err {
			logger.LOGE(err.Error())
		}
	}()
	if nil == msg || nil == msg.Param1 {
		err = errors.New("invalid param")
		return
	}
	fileName := msg.Param1.(string)
	err = dashService.loadConfigFile(fileName)
	if nil != err {
		return
	}
	strPort := ":" + strconv.Itoa(serviceConfig.Port)
	HTTPMUX.AddRoute(strPort, serviceConfig.Route, dashService.ServeHTTP)
	dashService.sources = make(map[string]*DASHSource)
	service = dashService
	return
}

func (dashService *DASHService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	streamName, reqType, param, err := dashService.parseURL(req.URL.String())
	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	dashService.muxSource.RLock()
	source, exist := dashService.sources[streamName]
	dashService.muxSource.RUnlock()
	if false == exist {

		dashService.createSource(streamName)

		dashService.muxSource.RLock()
		source, exist = dashService.sources[streamName]
		dashService.muxSource.RUnlock()

		if false == exist {
			w.WriteHeader(400)
			w.Write([]byte("bad request"))
		}
	}
	source.serveHTTP(reqType, param, w, req)
}
func (dashService *DASHService) createSource(streamName string) {
	ch := make(chan bool, 1)
	msg := &wssAPI.Msg{Param1: streamName, Param2: ch}
	source := &DASHSource{}
	err := source.Init(msg)
	if nil != err {
		logger.LOGE(err.Error())
		return
	}
	select {
	case ret, ok := <-ch:
		if false == ok || false == ret {
			source.Stop(nil)
		} else {
			dashService.muxSource.Lock()
			defer dashService.muxSource.Unlock()
			_, exist := dashService.sources[streamName]
			if exist {
				logger.LOGD("competition:" + streamName)
				return
			} else {
				dashService.sources[streamName] = source
			}
		}
	case <-time.After(time.Minute):
		source.Stop(nil)
		return
	}
}

func (dashService *DASHService) loadConfigFile(fileName string) (err error) {
	buf, err := wssAPI.ReadFileAll(fileName)
	if err != nil {
		return
	}
	err = json.Unmarshal(buf, &serviceConfig)
	return err
}

func (dashService *DASHService) Start(msg *wssAPI.Msg) (err error) {
	return
}

func (dashService *DASHService) Stop(msg *wssAPI.Msg) (err error) {
	return
}

func (dashService *DASHService) GetType() string {
	return wssAPI.OBJDASHServer
}

func (dashService *DASHService) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (dashService *DASHService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

func (dashService *DASHService) parseURL(url string) (streamName, reqType, param string, err error) {
	url = strings.TrimPrefix(url, serviceConfig.Route)
	url = strings.TrimSuffix(url, "/")
	subs := strings.Split(url, "/")
	if len(subs) < 2 {
		err = errors.New("invalid request :" + url)
		return
	}
	streamName = strings.TrimSuffix(url, subs[len(subs)-1])
	streamName = strings.TrimSuffix(streamName, "/")
	param = subs[len(subs)-1]
	if strings.HasPrefix(param, MpdPREFIX) || strings.HasSuffix(param, MpdPREFIX) {
		reqType = MpdPREFIX
	} else if strings.HasPrefix(param, VideoPREFIX) {
		reqType = VideoPREFIX
	} else if strings.HasPrefix(param, AudioPREFIX) {
		reqType = AudioPREFIX
	} else {
		err = errors.New("invalid req ")
	}
	return
}

func (dashService *DASHService) Add(name string, src *DASHSource) (err error) {
	dashService.muxSource.Lock()
	defer dashService.muxSource.Unlock()
	_, exist := dashService.sources[name]
	if exist {
		err = errors.New("source existed")
		return
	}
	dashService.sources[name] = src
	return
}

func (dashService *DASHService) Del(name, id string) {
	dashService.muxSource.Lock()
	defer dashService.muxSource.Unlock()
	src, exist := dashService.sources[name]
	if exist && src.clientId == id {
		delete(dashService.sources, name)
	}
}
