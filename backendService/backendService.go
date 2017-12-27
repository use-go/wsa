package backendService

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type backendServiceHander interface {
	init(msg *wssAPI.Msg) error
	getRoute() string
}

//BackendService for web
type BackendService struct {
}

//BackendConfig for web
type BackendConfig struct {
	Port     int    `json:"Port"`
	RootName string `json:"Usr"`
	RootPwd  string `json:"Pwd"`
}

var serviceConfig BackendConfig

//Init BackendService
func (backendService *BackendService) Init(msg *wssAPI.Msg) (err error) {
	if msg == nil || msg.Param1 == nil {
		logger.LOGE("init backend service failed")
		return errors.New("invalid param")
	}

	fileName := msg.Param1.(string)
	err = backendService.loadConfigFile(fileName)
	if err != nil {
		logger.LOGE(err.Error())
		return errors.New("load backend config failed")
	}

	return
}

func (backendService *BackendService) loadConfigFile(fileName string) (err error) {
	data, err := wssAPI.ReadFileAll(fileName)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		return
	}
	return
}

//Start Service in Goroutine
func (backendService *BackendService) Start(msg *wssAPI.Msg) (err error) {

	go func() {
		strPort := ":" + strconv.Itoa(serviceConfig.Port)
		handlers := backendHandlerInit()
		mux := http.NewServeMux()
		for _, item := range handlers {
			backHandler := item.(backendServiceHander)
			logger.LOGD(backHandler.getRoute())
			//http.Handle(backHandler.GetRoute(), http.StripPrefix(backHandler.GetRoute(), backHandler.(http.Handler)))
			mux.Handle(backHandler.getRoute(), http.StripPrefix(backHandler.getRoute(), backHandler.(http.Handler)))
		}
		err = http.ListenAndServe(strPort, mux)
		if err != nil {
			logger.LOGE("start backend serve failed")
		}
	}()

	return
}

//Stop Service
func (backendService *BackendService) Stop(msg *wssAPI.Msg) (err error) {
	return
}

//GetType of backend Service
func (backendService *BackendService) GetType() string {
	return wssAPI.OBJBackendServer
}

//HandleTask of Service
func (backendService *BackendService) HandleTask(task wssAPI.Task) (err error) {
	return
}

//ProcessMessage of Service
func (backendService *BackendService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

//SetParent handler for Service
func (backendService *BackendService) SetParent(parent wssAPI.MsgHandler) {
	return
}

func backendHandlerInit() []backendServiceHander {
	handers := make([]backendServiceHander, 0)
	adminLoginHandle := &adminLoginHandler{}
	lgData := &wssAPI.Msg{}
	loginData := adminLoginData{}
	loginData.password = serviceConfig.RootPwd
	loginData.username = serviceConfig.RootName
	lgData.Param1 = loginData
	err := adminLoginHandle.init(lgData)
	if err == nil {
		handers = append(handers, adminLoginHandle)
	} else {
		if err != nil {
			logger.LOGE("add adminLoginHandle error!")
		}
	}

	streamManagerHandle := &adminStreamManageHandler{}
	streamManagerHandle.init(nil)
	handers = append(handers, streamManagerHandle)
	return handers
}
