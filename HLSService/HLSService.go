package HLSService

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/use-go/websocketStreamServer/logger"

	"github.com/use-go/websocketStreamServer/HTTPMUX"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

type HLSService struct {
	sources   map[string]*HLSSource
	muxSource sync.RWMutex
	icoData   []byte
}

type HLSConfig struct {
	Port  int    `json:"Port"`
	Route string `json:"Route"`
	ICO   string `json:"ico"`
}

var service *HLSService
var serviceConfig HLSConfig

func (hlsService *HLSService) Init(msg *wssAPI.Msg) (err error) {
	defer func() {
		if nil != err {
			logger.LOGE(err.Error())
		}
	}()
	if nil == msg || nil == msg.Param1 {
		err = errors.New("invalid param")
		return
	}
	hlsService.sources = make(map[string]*HLSSource)
	fileName := msg.Param1.(string)
	err = hlsService.loadConfigFile(fileName)
	if err != nil {
		return
	}
	service = hlsService

	strPort := ":" + strconv.Itoa(serviceConfig.Port)
	HTTPMUX.AddRoute(strPort, serviceConfig.Route, hlsService.ServeHTTP)

	if len(serviceConfig.ICO) > 0 {
		hlsService.icoData, err = wssAPI.ReadFileAll(serviceConfig.ICO)
		if err != nil {
			logger.LOGW(err.Error())
			err = nil
			return
		}
	}

	if strings.HasPrefix(serviceConfig.Route, "/") {
		serviceConfig.Route = strings.TrimPrefix(serviceConfig.Route, "/")
	}
	if strings.HasSuffix(serviceConfig.Route, "/") {
		serviceConfig.Route = strings.TrimSuffix(serviceConfig.Route, "/")
	}
	return
}

func (hlsService *HLSService) loadConfigFile(fileName string) (err error) {
	buf, err := wssAPI.ReadFileAll(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &serviceConfig)
	if err != nil {
		return err
	}
	return
}

func (hlsService *HLSService) Start(msg *wssAPI.Msg) (err error) {

	go func() {
		//strPort := ":" + strconv.Itoa(serviceConfig.Port)
		//mux := http.NewServeMux()
		//mux.Handle(serviceConfig.Route, hlsService)
		//err = http.ListenAndServe(strPort, mux)
		//if err != nil {
		//	logger.LOGE("start websocket failed:" + err.Error())
		//}
		//HTTPMUX.AddRoute(strPort,serviceConfig.Route,hlsService.ServeHTTP)
	}()
	return
}

func (hlsService *HLSService) Stop(msg *wssAPI.Msg) (err error) {
	return
}

func (hlsService *HLSService) GetType() string {
	return wssAPI.OBJHLSServer
}

func (hlsService *HLSService) HandleTask(task wssAPI.Task) (err error) {
	return
}

func (hlsService *HLSService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

func (hlsService *HLSService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//url path:/prefex/app/streamname/level/master.m3u8 or a.m3u8 or v.m3u8
	//first v only
	defer func() {
		//logger.LOGT("serve end")
	}()
	if "/hls/ts.html" == req.URL.Path {
		data, err := wssAPI.ReadFileAll("ts.html")
		if err != nil {
			logger.LOGE(err.Error())
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", http.DetectContentType(data))
		w.Write(data)
		return
	}
	url := req.URL.Path
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")
	if strings.HasPrefix(url, serviceConfig.Route) {
		//logger.LOGD(url)
		url = strings.TrimPrefix(url, serviceConfig.Route)
		if strings.HasPrefix(url, "/") {
			//streamName := strings.TrimPrefix(url, "/")
			//
			//param:=streamName
			//if strings.HasSuffix(url, ".ts") {
			//	subs := strings.Split(url, "/")
			//	streamName = strings.TrimSuffix(streamName, subs[len(subs)-1])
			//	streamName=strings.TrimSuffix(streamName,"/")
			//}else if strings.HasSuffix(url,".m3u8"){
			//	streamName=strings.TrimSuffix(streamName,".m3u8")
			//}
			url = strings.TrimPrefix(url, "/")
			substrs := strings.Split(url, "/")
			if len(substrs) < 2 {
				w.WriteHeader(404)
				return
			}
			param := substrs[len(substrs)-1]
			streamName := strings.TrimSuffix(url, param)
			streamName = strings.TrimSuffix(streamName, "/")
			//
			//logger.LOGD(streamName)
			hlsService.muxSource.RLock()
			source, exist := hlsService.sources[streamName]
			hlsService.muxSource.RUnlock()
			if exist == false {
				source = hlsService.createSource(streamName)
				if wssAPI.InterfaceIsNil(source) {
					logger.LOGE("add hls source " + streamName + " failed")
					w.WriteHeader(404)
					return
				} else {
					source.ServeHTTP(w, req, param)
					return
				}
			}
			source.ServeHTTP(w, req, param)
		} else {
			w.WriteHeader(404)
			return
		}
	} else {
		//ico or invalid
		if "favicon.ico" == url {
			if len(hlsService.icoData) > 0 {
				contentType := http.DetectContentType(hlsService.icoData)
				w.Header().Set("Content-type", contentType)
				w.Write(hlsService.icoData)
			}
		} else {
			w.WriteHeader(404)
		}
	}
}

func (hlsService *HLSService) Add(key string, v *HLSSource) (err error) {
	hlsService.muxSource.Lock()
	defer hlsService.muxSource.Unlock()
	_, ok := hlsService.sources[key]
	if true == ok {
		err = errors.New("source existed")
		return
	}
	hlsService.sources[key] = v
	return
}

func (hlsService *HLSService) DelSource(key, id string) {
	hlsService.muxSource.Lock()
	defer hlsService.muxSource.Unlock()
	src, exist := hlsService.sources[key]
	if exist && src.clientId == id {
		delete(hlsService.sources, key)
	}
}

func (hlsService *HLSService) createSource(streamName string) (source *HLSSource) {
	chSvr := make(chan bool, 1)
	msg := &wssAPI.Msg{
		Param1: streamName,
		Param2: chSvr}
	source = &HLSSource{}
	logger.LOGD("create source")
	err := source.Init(msg)
	if err != nil {
		logger.LOGE(err.Error())
		return nil
	}

	logger.LOGD("create source2")
	select {
	case ret, ok := <-chSvr:
		if false == ok || false == ret {
			source.Stop(nil)
		} else {
			hlsService.muxSource.Lock()
			defer hlsService.muxSource.Unlock()
			old, exist := hlsService.sources[streamName]
			if exist == true {
				source.Stop(nil)
				return old
			} else {
				hlsService.sources[streamName] = source
				return source
			}
		}
	case <-time.After(time.Minute):
		source.Stop(nil)
		return nil
	}
	return
}
