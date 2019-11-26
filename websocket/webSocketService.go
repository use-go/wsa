package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/use-go/websocket-streamserver/httpmux"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/utils"
	"github.com/use-go/websocket-streamserver/wssapi"
)

// WebSocketService to handle webservice business
type WebSocketService struct {
	parent wssapi.MsgHandler
}

// WebSocketConfig to store webservice configuration information
type WebSocketConfig struct {
	Port  int    `json:"Port"`
	Route string `json:"Route"`
}

var wsService *WebSocketService
var serviceConfig WebSocketConfig
var serviceAddrWithPort string

// Init interface implemention to init current websocket
func (websockService *WebSocketService) Init(msg *wssapi.Msg) (err error) {

	if msg == nil || msg.Param1 == nil {
		logger.LOGE("init Websocket service failed,param from json file :nil")
		return errors.New("invalid param")
	}
	fileName := msg.Param1.(string)
	// Fill the serviceConfig from config file
	err = websockService.loadConfigFile(fileName)
	if err != nil {
		logger.LOGE(err.Error())
		return errors.New("load websocket config failed")
	}
	wsService = websockService
	serviceAddrWithPort = ":" + strconv.Itoa(serviceConfig.Port)
	httpmux.AddRoute(serviceAddrWithPort, serviceConfig.Route, websockService.ServeHTTP)

	return
}

// Start interface implemention
func (websockService *WebSocketService) Start(msg *wssapi.Msg) (err error) {

	serverMux, err := httpmux.GetPortServe(serviceAddrWithPort)
	if err != nil {
		logger.LOGE(err.Error())
		return errors.New("Somthing error when retrive httpMux of websocket")
	}

	go func(addr string, handler http.Handler) {
		for {
			err := http.ListenAndServe(addr, handler)
			if err != nil {
				logger.LOGE(err.Error())
				continue
			}
		}

	}(serviceAddrWithPort, serverMux)

	return
}

// Stop interface implemention
func (websockService *WebSocketService) Stop(msg *wssapi.Msg) (err error) {
	//TODO....
	return
}

// GetType interface implemention
func (websockService *WebSocketService) GetType() string {
	return wssapi.OBJWebSocketServer
}

// HandleTask interface implemention
func (websockService *WebSocketService) HandleTask(task wssapi.Task) (err error) {

	// defer func() {
	// 	conn.Close()
	// 	logger.LOGD("close websocket connection :" + conn.RemoteAddr().String())
	// }()
	return
}

// ProcessMessage interface implemention
func (websockService *WebSocketService) ProcessMessage(msg *wssapi.Msg) (err error) {
	return
}

//loadConfigFile from FS to config Current Service
func (websockService *WebSocketService) loadConfigFile(fileName string) (err error) {
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

func (websockService *WebSocketService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	path = strings.TrimPrefix(path, serviceConfig.Route)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	//logger.LOGT(path)
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	//sub Protocol Negotiation
	subProtocol := req.Header.Get("Sec-Websocket-Protocol")
	respHeader := http.Header{"Sec-Websocket-Protocol": []string{subProtocol}}

	conn, err := upgrader.Upgrade(w, req, respHeader)
	if err != nil {
		logger.LOGE("webSocket handshake failed with error " + err.Error() + "of remote :" + conn.RemoteAddr().String())
		return
	}
	//remmber to close the websocket connetction
	// defer func() {
	// 	conn.Close()
	// 	logger.LOGD("close websocket connection :" + conn.RemoteAddr().String())
	// }()

	//new connection came here
	logger.LOGT(fmt.Sprintf("new websocket client connected: %s", conn.RemoteAddr().String()))
	go websockService.handleConn(conn, req, path)

}

func (websockService *WebSocketService) handleConn(conn *websocket.Conn, req *http.Request, path string) {
	handler := &websocketHandler{}
	msg := &wssapi.Msg{}
	msg.Param1 = conn
	msg.Param2 = path
	handler.Init(msg)
	//close the handling proc
	defer func() {
		handler.processWSMessage(nil)
	}()

	//main Handling proc
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		switch messageType {
		case websocket.TextMessage:
			//err = conn.WriteMessage(websocket.TextMessage, data)
			err = handler.ProcessWSCtrlMessage(data)
			if err != nil {
				logger.LOGI("websocket control msg error : " + err.Error() + " target :" + conn.RemoteAddr().String())
				return
			}
		case websocket.BinaryMessage:
			err = handler.processWSMessage(data)
			if err != nil {
				logger.LOGE(err.Error())
				logger.LOGE("websocket binary error in connection :" + conn.RemoteAddr().String())
				return
			}
		case websocket.CloseMessage:
			err = errors.New("websocket closed by client:" + conn.RemoteAddr().String())
			return
		case websocket.PingMessage:
			//conn.WriteMessage(websocket.PongMessage, []byte(""))
			conn.WriteMessage(websocket.PongMessage, []byte("pongMessage"))
		case websocket.PongMessage:
			//the below line can be removed
			logger.LOGD("pong message received from connection :" + conn.RemoteAddr().String())
		default:
		}
	}
}

// SetParent handler for websocketservice
func (websockService *WebSocketService) SetParent(parent wssapi.MsgHandler) {
	websockService.parent = parent
}
