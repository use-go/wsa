package webSocketService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/use-go/websocketStreamServer/HTTPMUX"
	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

// WebSocketService to handle webservice business
type WebSocketService struct {
	parent wssAPI.MsgHandler
}

// WebSocketConfig to store webservice configinfo
type WebSocketConfig struct {
	Port  int    `json:"Port"`
	Route string `json:"Route"`
}

var service *WebSocketService
var serviceConfig WebSocketConfig

// Init  interface implemention
func (websockService *WebSocketService) Init(msg *wssAPI.Msg) (err error) {

	if msg == nil || msg.Param1 == nil {
		logger.LOGE("init Websocket service failed")
		return errors.New("invalid param")
	}

	fileName := msg.Param1.(string)
	err = websockService.loadConfigFile(fileName)
	if err != nil {
		logger.LOGE(err.Error())
		return errors.New("load websocket config failed")
	}
	service = websockService
	strPort := ":" + strconv.Itoa(serviceConfig.Port)
	HTTPMUX.AddRoute(strPort, serviceConfig.Route, websockService.ServeHTTP)
	//go func() {
	//	if true {
	//		strPort := ":" + strconv.Itoa(serviceConfig.Port)
	//		//http.Handle("/", websockService)
	//		mux := http.NewServeMux()
	//		mux.Handle("/", websockService)
	//		err = http.ListenAndServe(strPort, mux)
	//		if err != nil {
	//			logger.LOGE("start websocket failed:" + err.Error())
	//		}
	//	}
	//}()
	return
}

// Start interface implemention
func (websockService *WebSocketService) Start(msg *wssAPI.Msg) (err error) {
	return
}

// Stop interface implemention
func (websockService *WebSocketService) Stop(msg *wssAPI.Msg) (err error) {
	return
}

// GetType interface implemention
func (websockService *WebSocketService) GetType() string {
	return wssAPI.OBJ_WebSocketServer
}

// HandleTask interface implemention
func (websockService *WebSocketService) HandleTask(task wssAPI.Task) (err error) {
	return
}

// ProcessMessage interface implemention
func (websockService *WebSocketService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

func (websockService *WebSocketService) loadConfigFile(fileName string) (err error) {
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

func (websockService *WebSocketService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	path = strings.TrimPrefix(path, serviceConfig.Route)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	//logger.LOGT(path)
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	logger.LOGT(fmt.Sprintf("new websocket connect %s", conn.RemoteAddr().String()))
	websockService.handleConn(conn, req, path)
	defer func() {
		conn.Close()
		logger.LOGD("close websocket conn")
	}()
}

func (websockService *WebSocketService) handleConn(conn *websocket.Conn, req *http.Request, path string) {
	handler := &websocketHandler{}
	msg := &wssAPI.Msg{}
	msg.Param1 = conn
	msg.Param2 = path

	handler.Init(msg)
	defer func() {
		handler.processWSMessage(nil)
	}()
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		switch messageType {
		case websocket.TextMessage:
			err = conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				logger.LOGI("send text msg failed:" + err.Error())
				return
			}
		case websocket.BinaryMessage:
			err = handler.processWSMessage(data)
			if err != nil {
				logger.LOGE(err.Error())
				logger.LOGE("ws binary error")
				return
			}
		case websocket.CloseMessage:
			err = errors.New("websocket closed:" + conn.RemoteAddr().String())
			return
		case websocket.PingMessage:
			//conn.WriteMessage()
			conn.WriteMessage(websocket.PongMessage, []byte(" "))
		case websocket.PongMessage:
		default:
		}
	}
}

func (websockService *WebSocketService) SetParent(parent wssAPI.MsgHandler) {
	websockService.parent = parent
}
