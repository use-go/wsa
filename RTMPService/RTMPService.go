package RTMPService

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/use-go/websocketStreamServer/events/eRTMPEvent"
	"github.com/use-go/websocketStreamServer/logger"
	"github.com/use-go/websocketStreamServer/wssAPI"
)

const (
	rtmpTypeHandler  = "rtmpHandler"
	rtmpTypePuller   = "rtmpPuller"
	livePathDefault  = "live"
	timeoutDefault   = 3000
	rtmpCacheDefault = 1000
)

type RTMPService struct {
	listener *net.TCPListener
	parent   wssAPI.MsgHandler
}

func init() {

}

type RTMPConfig struct {
	Port       int    `json:"Port"`
	TimeoutSec int    `json:"TimeoutSec"`
	LivePath   string `json:"LivePath"`
	CacheCount int    `json:"CacheCount"`
}

var service *RTMPService
var serviceConfig RTMPConfig

func (rtmpService *RTMPService) Init(msg *wssAPI.Msg) (err error) {
	if nil == msg || nil == msg.Param1 {
		logger.LOGE("invalid param init rtmp server")
		return errors.New("init rtmp service failed")
	}
	fileName := msg.Param1.(string)
	err = rtmpService.loadConfigFile(fileName)
	if err != nil {
		logger.LOGE(err.Error())
		return errors.New("init rtmp service failed")
	}
	service = rtmpService
	return
}

func (rtmpService *RTMPService) Start(msg *wssAPI.Msg) (err error) {
	logger.LOGT("start rtmp service")
	strPort := ":" + strconv.Itoa(serviceConfig.Port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", strPort)
	if nil != err {
		logger.LOGE(err.Error())
		return
	}
	rtmpService.listener, err = net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	go rtmpService.rtmpLoop()
	return
}

func (rtmpService *RTMPService) Stop(msg *wssAPI.Msg) (err error) {
	rtmpService.listener.Close()
	return
}

func (rtmpService *RTMPService) GetType() string {
	return wssAPI.OBJRTMPServer
}

func (rtmpService *RTMPService) HandleTask(task wssAPI.Task) (err error) {
	if task.Receiver() != rtmpService.GetType() {
		return errors.New("not my task")
	}
	switch task.Type() {
	case eRTMPEvent.PullRTMPStream:
		taskPull, ok := task.(*eRTMPEvent.EvePullRTMPStream)
		if false == ok {
			return errors.New("invalid param to pull rtmp stream")
		}
		taskPull.Protocol = strings.ToLower(taskPull.Protocol)
		switch taskPull.Protocol {
		case "rtmp":
			PullRTMPLive(taskPull)
		default:
			logger.LOGE(fmt.Sprintf("fmt %s not support now", taskPull.Protocol))
			close(taskPull.Src)
			return errors.New("fmt not support")
		}
		return
	default:
		return fmt.Errorf("task %s not prossed", task.Type())
	}
}

func (rtmpService *RTMPService) ProcessMessage(msg *wssAPI.Msg) (err error) {
	return
}

func (rtmpService *RTMPService) loadConfigFile(fileName string) (err error) {
	data, err := wssAPI.ReadFileAll(fileName)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &serviceConfig)
	if err != nil {
		return
	}

	if serviceConfig.TimeoutSec == 0 {
		serviceConfig.TimeoutSec = timeoutDefault
	}

	if len(serviceConfig.LivePath) == 0 {
		serviceConfig.LivePath = livePathDefault
	}
	if serviceConfig.CacheCount == 0 {
		serviceConfig.CacheCount = rtmpCacheDefault
	}
	strPort := ""
	if serviceConfig.Port != 1935 {
		strPort = strconv.Itoa(serviceConfig.Port)
	}
	logger.LOGI("rtmp://address:" + strPort + "/" + serviceConfig.LivePath + "/streamName")
	logger.LOGI("rtmp timeout: " + strconv.Itoa(serviceConfig.TimeoutSec) + " s")
	return
}

func (rtmpService *RTMPService) rtmpLoop() {
	for {
		conn, err := rtmpService.listener.Accept()
		if err != nil {
			logger.LOGW(err.Error())
			continue
		}
		go rtmpService.handleConnect(conn)
	}
}

func (rtmpService *RTMPService) handleConnect(conn net.Conn) {
	var err error
	defer conn.Close()
	defer logger.LOGT("close connect>>>")
	err = rtmpHandleshake(conn)
	if err != nil {
		logger.LOGE("rtmp handle shake failed")
		return
	}

	rtmp := &RTMP{}
	rtmp.Init(conn)

	msgInit := &wssAPI.Msg{}
	msgInit.Param1 = rtmp

	handler := &RTMPHandler{}
	err = handler.Init(msgInit)
	if err != nil {
		logger.LOGE("rtmp handler init failed")
		return
	}
	logger.LOGT("new connect:" + conn.RemoteAddr().String())
	for {
		var packet *RTMPPacket
		packet, err = rtmpService.readPacket(rtmp, handler.isPlaying())
		if err != nil {
			handler.HandleRTMPPacket(nil)
			return
		}
		err = handler.HandleRTMPPacket(packet)
		if err != nil {
			return
		}
	}
}

func (rtmpService *RTMPService) readPacket(rtmp *RTMP, playing bool) (packet *RTMPPacket, err error) {
	if false == playing {
		err = rtmp.Conn.SetReadDeadline(time.Now().Add(time.Duration(serviceConfig.TimeoutSec) * time.Second))
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
		defer rtmp.Conn.SetReadDeadline(time.Time{})
	}
	packet, err = rtmp.ReadPacket()
	return
}

func (rtmpService *RTMPService) SetParent(parent wssAPI.MsgHandler) {
	rtmpService.parent = parent
}
