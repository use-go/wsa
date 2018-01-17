package RTSPService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/utils"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

//RTSPService Service
type RTSPService struct {
}

//RTSPConfig for configuration from file
type RTSPConfig struct {
	Port       int `json:"port"`
	TimeoutSec int `json:"timeoutSec"`
}

var service *RTSPService
var serviceConfig RTSPConfig

//Init Service configuration for RTSPService
func (rtspService *RTSPService) Init(msg *wssAPI.Msg) (err error) {
	if nil == msg || nil == msg.Param1 {
		logger.LOGE("init rtsp server failed")
		return errors.New("invalid param init rtsp server")
	}
	fileName, ok := msg.Param1.(string)
	if false == ok {
		logger.LOGE("bad param init rtsp server")
		return errors.New("invalid param init rtsp server")
	}
	err = rtspService.loadConfigFile(fileName)
	if err != nil {
		logger.LOGE("load rtsp config failed:" + err.Error())
		return
	}
	return
}

func (rtspService *RTSPService) loadConfigFile(fileName string) (err error) {
	data, err := utils.ReadFileAll(fileName)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		return
	}
	if serviceConfig.TimeoutSec <= 0 {
		serviceConfig.TimeoutSec = 60
	}

	if serviceConfig.Port == 0 {
		serviceConfig.Port = 554
	}
	return
}

//Start RTSPService
func (rtspService *RTSPService) Start(msg *wssAPI.Msg) (err error) {
	logger.LOGT("start RTSP server")
	strPort := ":" + strconv.Itoa(serviceConfig.Port)
	tcp, err := net.ResolveTCPAddr("tcp4", strPort)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	listener, err := net.ListenTCP("tcp4", tcp)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	go rtspService.rtspLoop(listener)
	return
}

func (rtspService *RTSPService) rtspLoop(listener *net.TCPListener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.LOGE(err.Error())
			continue
		}
		logger.LOGT(fmt.Sprintf("new rtsp client connected: %s", conn.RemoteAddr().String()))
		go rtspService.handleConn(conn)
	}
}

func (rtspService *RTSPService) handleConn(conn net.Conn) {
	defer func() {
		conn.Close()
	}()
	handler := &RTSPHandler{}
	handler.conn = conn
	handler.Init(nil)
	for {
		data, err := ReadPacket(conn, handler.tcpTimeout)
		if err != nil {
			logger.LOGE("read rtsp failed with error: " + err.Error())
			handler.handlePacket(nil)
			return
		}
		logger.LOGT(string(data))
		err = handler.handlePacket(data)
		if err != nil {
			logger.LOGE("handle rtsp packet error: " + err.Error())
			return
		}
	}
}

//Stop RTSPService not implemention
func (rtspService *RTSPService) Stop(msg *wssAPI.Msg) (err error) {
	return
}

//GetType return name of current service
func (rtspService *RTSPService) GetType() string {
	return wssAPI.OBJRTSPServer
}

//HandleTask  not implemention
func (rtspService *RTSPService) HandleTask(task wssAPI.Task) (err error) {
	return
}

//ProcessMessage  not implemention
func (rtspService *RTSPService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}
