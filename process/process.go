// Copyright 2017-2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package process

import (
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/use-go/websocket-streamserver/rtspsrv"
	"github.com/use-go/websocket-streamserver/backend"
	"github.com/use-go/websocket-streamserver/dash"
	"github.com/use-go/websocket-streamserver/hls"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/rtmp"
	"github.com/use-go/websocket-streamserver/streamer"
	"github.com/use-go/websocket-streamserver/utils"
	"github.com/use-go/websocket-streamserver/websocket"
	"github.com/use-go/websocket-streamserver/wssapi"
)

/*
Package process , this is the main process to control all the services
1、read process config from configuration
2、create and init each service
3、start each configed service
*/

type contextConfiguration struct {
	RTMPConfigName          string `json:"RTMP"`
	WebSocketConfigName     string `json:"WebSocket"`
	BackendConfigName       string `json:"Backend"`
	LogPath                 string `json:"LogPath"`
	StreamManagerConfigName string `json:"Streamer"`
	HLSConfigName           string `json:"HLS"`
	DASHConfigName          string `json:"DASH,omitempty"`
	RTSPConfigName          string `json:"RTSP,omitempty"`
}

//context : context holding all the Service that will be launched in Process
type context struct {
	servicesRWMutex sync.RWMutex //service sync operation
	services        map[string]wssapi.MsgHandler
}

var processContext *context
var processConfig *contextConfiguration

func init() {
	processContext = &context{}
	wssapi.SetHandler(processContext)
}

// Run all Services Of current processing
func Run() {
	processContext.Init(nil)
	processContext.createAllService(nil)
	processContext.Start(nil)
}

// Init the service Configuration from json file such as hls/dash/rtsp
func (processCtx *context) Init(msg *wssapi.Msg) (err error) {
	processCtx.services = make(map[string]wssapi.MsgHandler)
	err = processCtx.loadConfig()
	if err != nil {
		logger.LOGE("process load config failed")
		return
	}
	return
}

//Create the needed Service Instance And Save it to HttpMux
func (processCtx *context) createAllService(msg *wssapi.Msg) (err error) {

	//Cretate Streamer Service
	if true {
		livingSvr := &streamer.StreamerService{}
		msg := &wssapi.Msg{}
		if len(processConfig.StreamManagerConfigName) > 0 {
			msg.Param1 = processConfig.StreamManagerConfigName
		}
		err = livingSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[livingSvr.GetType()] = livingSvr
			processCtx.servicesRWMutex.Unlock()
		}
	}

	//create RTMP Service
	if len(processConfig.RTMPConfigName) > 0 {
		rtmpSvr := &rtmp.RTMPService{}
		msg := &wssapi.Msg{}
		msg.Param1 = processConfig.RTMPConfigName
		err = rtmpSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[rtmpSvr.GetType()] = rtmpSvr
			processCtx.servicesRWMutex.Unlock()
		}
	}

	//create WebSocket Service
	if len(processConfig.WebSocketConfigName) > 0 {
		webSocketSvr := &websocket.WebSocketService{}
		msg := &wssapi.Msg{}
		msg.Param1 = processConfig.WebSocketConfigName
		err = webSocketSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[webSocketSvr.GetType()] = webSocketSvr
			processCtx.servicesRWMutex.Unlock()
		}
	}

	//create backend
	if len(processConfig.BackendConfigName) > 0 {
		backendSvr := &backend.BackendService{}
		msg := &wssapi.Msg{}
		msg.Param1 = processConfig.BackendConfigName
		err = backendSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[backendSvr.GetType()] = backendSvr
			processCtx.servicesRWMutex.Unlock()
		}
	}

	//create RTSP Service
	if len(processConfig.RTSPConfigName) > 0 {
		rtspSvr := &rtspsrv.RTSPService{}
		msg := &wssapi.Msg{}
		msg.Param1 = processConfig.RTSPConfigName
		err = rtspSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[rtspSvr.GetType()] = rtspSvr
			processCtx.servicesRWMutex.Unlock()
		}
	}
	//create HLS Service
	if len(processConfig.HLSConfigName) > 0 {
		hls := &hls.HLSService{}
		msg := &wssapi.Msg{Param1: processConfig.HLSConfigName}
		err = hls.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[hls.GetType()] = hls
			processCtx.servicesRWMutex.Unlock()
		}
	}
	//create DASH Service
	if len(processConfig.DASHConfigName) > 0 {
		dash := &dash.DASHService{}
		msg := &wssapi.Msg{Param1: processConfig.DASHConfigName}
		err = dash.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.servicesRWMutex.Lock()
			processCtx.services[dash.GetType()] = dash
			processCtx.servicesRWMutex.Unlock()
		}
	}

	return
}

//Load Main configuration file config.json to init Services ，next
func (processCtx *context) loadConfig() (err error) {
	configName := ""
	if len(os.Args) > 1 {
		configName = os.Args[1]
	} else {
		logger.LOGW("use default :config.json")
		configName = "config.json"
	}
	data, err := utils.ReadFileAll(configName)
	if err != nil {
		logger.LOGE("process load config file failed:" + err.Error())
		return
	}
	processConfig = &contextConfiguration{}
	err = json.Unmarshal(data, processConfig)

	if err != nil {
		logger.LOGE(err.Error())
		return
	}

	if len(processConfig.LogPath) > 0 {
		processCtx.createLogFile(processConfig.LogPath)
	}

	return
}

//Log info
func (processCtx *context) createLogFile(logPath string) {
	if strings.HasSuffix(logPath, "/") {
		logPath = strings.TrimSuffix(logPath, "/")
	}
	dir := logPath + time.Now().Format("/2006/01/02/")
	bResult, _ := utils.CheckDirectory(dir)

	if false == bResult {
		_, err := utils.CreateDirectory(dir)
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
func (processCtx *context) Start(msg *wssapi.Msg) (err error) {
	//if false {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//}
	//httpmux.Start() //remove Logic Design to Start All Service Once
	processCtx.servicesRWMutex.RLock()
	defer processCtx.servicesRWMutex.RUnlock()

	if len(processCtx.services) < 1 {
		logger.LOGI("no service avaiable")
		return errors.New("no service avaiable")
	}

	for k, v := range processCtx.services {
		//v.SetParent(processCtx)
		err = v.Start(nil)
		if err != nil {
			logger.LOGE("start " + k + " failed:" + err.Error())
			continue
		}
		logger.LOGI("start " + k + " successed ")
	}

	return
}

//Stop all the launched services
func (processCtx *context) Stop(msg *wssapi.Msg) (err error) {
	processCtx.servicesRWMutex.RLock()
	defer processCtx.servicesRWMutex.RUnlock()
	for _, v := range processCtx.services {
		err = v.Stop(nil)
	}
	return
}

//GetType for process
func (processCtx *context) GetType() string {
	return wssapi.OBJProcess
}

//HandleTask for process
func (processCtx *context) HandleTask(task wssapi.Task) (err error) {
	processCtx.servicesRWMutex.RLock()
	defer processCtx.servicesRWMutex.RUnlock()
	handler, exist := processCtx.services[task.Receiver()]
	if exist == false {
		return errors.New("invalid task")
	}
	return handler.HandleTask(task)
}

//ProcessMessage for process
func (processCtx *context) ProcessMessage(msg *wssapi.Msg) (err error) {
	return nil
}

//SetParent for process
func (processCtx *context) SetParent(arent wssapi.MsgHandler) {

}

//AddSvr for process
func AddSvr(svr wssapi.MsgHandler) {
	logger.LOGE("add svr")
}
