// Copyright 2017 The use-go websocket-streamserver Authors. All rights reserved.
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

	"github.com/use-go/websocket-streamserver/DASHService"
	"github.com/use-go/websocket-streamserver/HLSService"
	"github.com/use-go/websocket-streamserver/HTTPMUX"
	"github.com/use-go/websocket-streamserver/RTMPService"
	"github.com/use-go/websocket-streamserver/RTSPService"
	"github.com/use-go/websocket-streamserver/backend"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/streamer"
	"github.com/use-go/websocket-streamserver/webSocketService"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

/*
Package process , this is the main process to control all the services
1、read process config from configuration
2、create and init each service
3、start each configed service
*/

type processConfiguration struct {
	RTMPConfigName          string `json:"RTMP"`
	WebSocketConfigName     string `json:"WebSocket"`
	BackendConfigName       string `json:"Backend"`
	LogPath                 string `json:"LogPath"`
	StreamManagerConfigName string `json:"Streamer"`
	HLSConfigName           string `json:"HLS"`
	DASHConfigName          string `json:"DASH,omitempty"`
	RTSPConfigName          string `json:"RTSP,omitempty"`
}

//Context : Context holding all the Service that will be launched in Process
type Context struct {
	mutexServices sync.RWMutex
	services      map[string]wssAPI.MsgHandler
}

var processContext *Context

var processConfig *processConfiguration

func init() {
	processContext = &Context{}
	wssAPI.SetHandler(processContext)
}

// Start init the server processing
func Start() {
	processContext.Init(nil)
	processContext.createAllService(nil)
	processContext.Start(nil)
}

// Init the configed service such as hls/dash/rtsp
func (processCtx *Context) Init(msg *wssAPI.Msg) (err error) {
	processCtx.services = make(map[string]wssAPI.MsgHandler)
	err = processCtx.loadConfig()
	if err != nil {
		logger.LOGE("svr bus load config failed")
		return
	}
	return
}

func (processCtx *Context) createAllService(msg *wssAPI.Msg) (err error) {

	if true {
		livingSvr := &streamer.StreamerService{}
		msg := &wssAPI.Msg{}
		if len(processConfig.StreamManagerConfigName) > 0 {
			msg.Param1 = processConfig.StreamManagerConfigName
		}
		err = livingSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[livingSvr.GetType()] = livingSvr
			processCtx.mutexServices.Unlock()
		}
	}

	//create RTMP Service
	if len(processConfig.RTMPConfigName) > 0 {
		rtmpSvr := &RTMPService.RTMPService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = processConfig.RTMPConfigName
		err = rtmpSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[rtmpSvr.GetType()] = rtmpSvr
			processCtx.mutexServices.Unlock()
		}
	}

	//create WebSocket Service
	if len(processConfig.WebSocketConfigName) > 0 {
		webSocketSvr := &webSocketService.WebSocketService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = processConfig.WebSocketConfigName
		err = webSocketSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[webSocketSvr.GetType()] = webSocketSvr
			processCtx.mutexServices.Unlock()
		}
	}

	//create backendService
	if len(processConfig.BackendConfigName) > 0 {
		backendSvr := &backend.BackendService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = processConfig.BackendConfigName
		err = backendSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[backendSvr.GetType()] = backendSvr
			processCtx.mutexServices.Unlock()
		}
	}

	//create RTSP Service
	if len(processConfig.RTSPConfigName) > 0 {
		rtspSvr := &RTSPService.RTSPService{}
		msg := &wssAPI.Msg{}
		msg.Param1 = processConfig.RTSPConfigName
		err = rtspSvr.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[rtspSvr.GetType()] = rtspSvr
			processCtx.mutexServices.Unlock()
		}
	}
	//create HLS Service
	if len(processConfig.HLSConfigName) > 0 {
		hls := &HLSService.HLSService{}
		msg := &wssAPI.Msg{Param1: processConfig.HLSConfigName}
		err = hls.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[hls.GetType()] = hls
			processCtx.mutexServices.Unlock()
		}
	}
	//create DASH Service
	if len(processConfig.DASHConfigName) > 0 {
		dash := &DASHService.DASHService{}
		msg := &wssAPI.Msg{Param1: processConfig.DASHConfigName}
		err = dash.Init(msg)
		if err != nil {
			logger.LOGE(err.Error())
		} else {
			processCtx.mutexServices.Lock()
			processCtx.services[dash.GetType()] = dash
			processCtx.mutexServices.Unlock()
		}
	}

	return
}

func (processCtx *Context) loadConfig() (err error) {
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
	processConfig = &processConfiguration{}
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

func (processCtx *Context) createLogFile(logPath string) {
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
func (processCtx *Context) Start(msg *wssAPI.Msg) (err error) {
	//if false {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//}
	HTTPMUX.Start()
	processCtx.mutexServices.RLock()
	defer processCtx.mutexServices.RUnlock()
	for k, v := range processCtx.services {
		//v.SetParent(processCtx)
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
func (processCtx *Context) Stop(msg *wssAPI.Msg) (err error) {
	processCtx.mutexServices.RLock()
	defer processCtx.mutexServices.RUnlock()
	for _, v := range processCtx.services {
		err = v.Stop(nil)
	}
	return
}

//GetType for process
func (processCtx *Context) GetType() string {
	return wssAPI.OBJProcess
}

//HandleTask for process
func (processCtx *Context) HandleTask(task wssAPI.Task) (err error) {
	processCtx.mutexServices.RLock()
	defer processCtx.mutexServices.RUnlock()
	handler, exist := processCtx.services[task.Receiver()]
	if exist == false {
		return errors.New("invalid task")
	}
	return handler.HandleTask(task)
}

//ProcessMessage for process
func (processCtx *Context) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return nil
}

//SetParent for process
func (processCtx *Context) SetParent(arent wssAPI.MsgHandler) {

}

//AddSvr for process
func AddSvr(svr wssAPI.MsgHandler) {
	logger.LOGE("add svr")
}
