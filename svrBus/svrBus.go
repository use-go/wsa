package svrBus

import (
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/use-go/websocketStreamServer/DASH"
	"github.com/use-go/websocketStreamServer/HLSService"
	"github.com/use-go/websocketStreamServer/HTTPMUX"
	"github.com/use-go/websocketStreamServer/RTMPService"
	"github.com/use-go/websocketStreamServer/RTSPService"
	"github.com/use-go/websocketStreamServer/backend"
	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/streamer"
	"github.com/use-go/websocketStreamServer/webSocketService"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

type busConfig struct {
	RTMPConfigName          string `json:"RTMP"`
	WebSocketConfigName     string `json:"WebSocket"`
	BackendConfigName       string `json:"Backend"`
	LogPath                 string `json:"LogPath"`
	StreamManagerConfigName string `json:"Streamer"`
	HLSConfigName           string `json:"HLS"`
	DASHConfigName          string `json:"DASH,omitempty"`
	RTSPConfigName          string `json:"RTSP,omitempty"`
}

//ServiceBus : ServiceBus holding all the Service that will be launched in Process
type ServiceBus struct {
	mutexServices sync.RWMutex
	services      map[string]wssAPI.Obj
}

var serviceBus *ServiceBus

func init() {
	serviceBus = &ServiceBus{}
	wssAPI.SetBus(serviceBus)
}

// Start init the server processing
func Start() {
	serviceBus.Init(nil)
	serviceBus.Start(nil)
}

// Init the configed service such as hls/dash/rtsp
func (srvBus *ServiceBus) Init(msg *wssAPI.Msg) (err error) {
	srvBus.services = make(map[string]wssAPI.Obj)
	err = srvBus.loadConfig()
	if err != nil {
		logger.LOGE("svr bus load config failed")
		return
	}
	return
}

func (srvBus *ServiceBus) loadConfig() (err error) {
	configName := ""
	if len(os.Args) > 1 {
		configName = os.Args[1]
	} else {
		logger.LOGW("use default :config.json")
		configName = "config.json"
	}
	data, err := wssAPI.ReadFileAll(configName)
	if err != nil {
		logger.LOGE("load config file failed:" + err.Error())
		return
	}
	cfg := &busConfig{}
	err = json.Unmarshal(data, cfg)

	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	if len(cfg.LogPath) > 0 {
		srvBus.createLogFile(cfg.LogPath)
	}

	if true {
		livingSvr := &streamer.StreamerService{}
		msg := &wssAPI.Msg{}
		if len(cfg.StreamManagerConfigName) > 0 {
			msg.Param1 = cfg.StreamManagerConfigName
		}
		err = livingSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[livingSvr.GetType()] = livingSvr
			srvBus.mutexServices.Unlock()
		}
	}

	//start RTMP Service
	if len(cfg.RTMPConfigName) > 0 {
		rtmpSvr := &RTMPService.RTMPService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = cfg.RTMPConfigName
		err = rtmpSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[rtmpSvr.GetType()] = rtmpSvr
			srvBus.mutexServices.Unlock()
		}
	}

	//start WebSocket Service
	if len(cfg.WebSocketConfigName) > 0 {
		webSocketSvr := &webSocketService.WebSocketService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = cfg.WebSocketConfigName
		err = webSocketSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[webSocketSvr.GetType()] = webSocketSvr
			srvBus.mutexServices.Unlock()
		}
	}

	//start backendService
	if len(cfg.BackendConfigName) > 0 {
		backendSvr := &backend.BackendService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = cfg.BackendConfigName
		err = backendSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[backendSvr.GetType()] = backendSvr
			srvBus.mutexServices.Unlock()
		}
	}

	//start RTSP Service
	if len(cfg.RTSPConfigName) > 0 {
		rtspSvr := &RTSPService.RTSPService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = cfg.RTSPConfigName
		err = rtspSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[rtspSvr.GetType()] = rtspSvr
			srvBus.mutexServices.Unlock()
		}
	}
	//start HLS Service
	if len(cfg.HLSConfigName) > 0 {
		hls := &HLSService.HLSService{}
		msg := &wssAPI.Msg{Param1: cfg.HLSConfigName}
		err = hls.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[hls.GetType()] = hls
			srvBus.mutexServices.Unlock()
		}
	}
	//start DASH Service
	if len(cfg.DASHConfigName) > 0 {
		dash := &DASH.DASHService{}
		msg := &wssAPI.Msg{Param1: cfg.DASHConfigName}
		err = dash.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			srvBus.mutexServices.Lock()
			srvBus.services[dash.GetType()] = dash
			srvBus.mutexServices.Unlock()
		}
	}
	return
}

func (srvBus *ServiceBus) createLogFile(logPath string) {
	if strings.HasSuffix(logPath, "/") {
		logPath = strings.TrimSuffix(logPath, "/")
	}
	dir := logPath + time.Now().Format("/2006/01/02/")
	bResult, _ := wssAPI.CheckDirectory(dir)

	if false == bResult {
		_, err := wssAPI.CreateDirectory(dir)
		if err != nil {
			logger.LOGE("create log file failed:", err.Error())
			return
		}
	}
	fullName := dir + time.Now().Format("2006-01-02_15.04") + ".log"
	fp, err := os.OpenFile(fullName, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_TRUNC, os.ModePerm)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	logger.SetOutput(fp)
	//start a go routine in backgroud to avoid one log file grouthing too big
	go func() {
		logFileTick := time.Tick(time.Hour * 72)
		for {
			select {
			case <-logFileTick:
				fullName := dir + time.Now().Format("2006-01-02_15:04") + ".log"
				newLogFile, _ := os.OpenFile(fullName, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_TRUNC, os.ModePerm)
				if newLogFile != nil {
					logger.SetOutput(newLogFile)
					fp.Close()
					fp = newLogFile
				}
			}
		}
	}()
}

//Start all the configed service ,such as rtsp ,websocket ,backend
func (srvBus *ServiceBus) Start(msg *wssAPI.Msg) (err error) {
	//if false {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//}
	HTTPMUX.Start()
	srvBus.mutexServices.RLock()
	defer srvBus.mutexServices.RUnlock()
	for k, v := range srvBus.services {
		//v.SetParent(srvBus)
		err = v.Start(nil)
		if err != nil {
			logger.LOGE("start " + k + " failed:" + err.Error())
			continue
		}
		logger.LOGI("start " + k + " successed")
	}

	return
}

//Stop all the launched services
func (srvBus *ServiceBus) Stop(msg *wssAPI.Msg) (err error) {
	srvBus.mutexServices.RLock()
	defer srvBus.mutexServices.RUnlock()
	for _, v := range srvBus.services {
		err = v.Stop(nil)
	}
	return
}

func (srvBus *ServiceBus) GetType() string {
	return wssAPI.OBJ_ServerBus
}

func (srvBus *ServiceBus) HandleTask(task wssAPI.Task) (err error) {
	srvBus.mutexServices.RLock()
	defer srvBus.mutexServices.RUnlock()
	handler, exist := srvBus.services[task.Receiver()]
	if exist == false {
		return errors.New("invalid task")
	}
	return handler.HandleTask(task)
}

func (srvBus *ServiceBus) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return nil
}

func (srvBus *ServiceBus) SetParent(arent wssAPI.Obj) {

}

func AddSvr(svr wssAPI.Obj) {
	logger.LOGE("add svr")
}
