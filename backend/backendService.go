package backend

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssapi"
	"github.com/use-go/websocket-streamserver/utils"
)

//interface for each service instance
type backendServiceHander interface {
	init(msg *wssapi.Msg) error
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

func serveDefaultHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", 404)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	http.ServeFile(w, r, "../test-websocket/video-player-living.html")
}

//Init BackendService
func (backend *BackendService) Init(msg *wssapi.Msg) (err error) {
	if msg == nil || msg.Param1 == nil {
		logger.LOGE("init backend service failed")
		return errors.New("invalid param")
	}

	fileName := msg.Param1.(string)
	err = backend.loadConfigFile(fileName)
	if err != nil {
		logger.LOGE(err.Error())
		return errors.New("load backend config failed")
	}

	return
}

func (backend *BackendService) loadConfigFile(fileName string) (err error) {
	data, err := utils.ReadFileAll(fileName)
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
func (backend *BackendService) Start(msg *wssapi.Msg) (err error) {

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
		//handle static assert
		mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("../test-websocket"))))
		mux.HandleFunc("/", serveDefaultHome)

		err = http.ListenAndServe(strPort, mux)
		if err != nil {
			logger.LOGE("start backend serve failed")
		}
	}()

	return
}

//Stop Service
func (backend *BackendService) Stop(msg *wssapi.Msg) (err error) {
	return
}

//GetType of backend Service
func (backend *BackendService) GetType() string {
	return wssapi.OBJBackendServer
}

//HandleTask of Service
func (backend *BackendService) HandleTask(task wssapi.Task) (err error) {
	return
}

//ProcessMessage of Service
func (backend *BackendService) ProcessMessage(msg *wssapi.Msg) (err error) {
	return
}

//SetParent handler for Service
func (backend *BackendService) SetParent(parent wssapi.MsgHandler) {
	return
}

func backendHandlerInit() []backendServiceHander {
	handers := make([]backendServiceHander, 0)
	adminLoginHandle := &adminLoginHandler{}
	lgData := &wssapi.Msg{}
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
