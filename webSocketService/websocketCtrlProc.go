// Copyright 2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webSocketService

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gorilla/websocket"

	"github.com/use-go/websocket-streamserver/RTSPClient"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/amf"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

type channelLink struct {
	rtspSocketChannel *RTSPClient.SocketChannel
	webSocketChannel  *websocket.Conn
	//  RTPSocketChannel map[int]*net.Conn
}

var (
	//channelIndex Denote the Channel Index In Session
	channelIndex int //channnel order
	channelList  map[*websocket.Conn]*RTSPClient.SocketChannel
)

func init() {
	//webSocketChannel = map[string]*websocket.Conn{}
	//rtspSocketChannel = map[string]*RTSPClient.SocketChannel{}
	channelIndex = 0
	channelList = make(map[*websocket.Conn]*RTSPClient.SocketChannel)
}

//ProcessWSCtrlMessage to deal the RTPS over RTSP
func (websockHandler *websocketHandler) ProcessWSCtrlMessage(data []byte) (err error) {
	if nil == data || len(data) < 4 {
		return errors.New("invalid msg")
	}
	ctrlMsg, err := wssAPI.DecodeWSPCtrlMsg(data)

	switch ctrlMsg.MsgType {
	case wssAPI.WSPCInit:
		//Init Video Session
		err = websockHandler.procWSPCInit(ctrlMsg)
	case wssAPI.WSPCJoin:
		//Join the streaming
		err = websockHandler.procWSPCJoin(ctrlMsg)
	case wssAPI.WSPWrap:
		//further to handle the wapperd rtsp message in content
		err = websockHandler.procWSPWarp(ctrlMsg)
	default:
		logger.LOGE("unknown websocket control type :from" + websockHandler.conn.RemoteAddr().String())
		err = errors.New("invalid ctrl msg type :from" + websockHandler.conn.RemoteAddr().String())
	}
	return
}

func (websockHandler *websocketHandler) checkChannel() (err error) {
	if _, exist := channelList[websockHandler.conn]; !exist {
		err = errors.New("Channel information not exist")
	}
	return
}

func (websockHandler *websocketHandler) procWSPWarp(ctrlMsg *wssAPI.WSPMessage) (err error) {
	//from here ,we send RTSP message to SS
	//when receiving reply ,forward it to client
	//check the end Channel exist
	if errr := websockHandler.checkChannel(); errr != nil {
		err = errr
		return
	}
	channelList[websockHandler.conn].ForwardToSer(ctrlMsg.Payload)

	return
}

func (websockHandler *websocketHandler) procWSPCJoin(ctrlMsg *wssAPI.WSPMessage) (err error) {

	seqStr := ctrlMsg.Headers["seq"]
	codeMsg := "200 OK"
	headerMsg := map[string]string{}
	payLoadMsg := ""
	//strChannel := ctrlMsg.Headers["channel"]
	if errr := websockHandler.checkChannel(); errr != nil {
		err = errr
		codeMsg = " 400 Bad Request"
	}

	replymsg := wssAPI.EncodeWSPCtrlMsg(codeMsg, seqStr, headerMsg, payLoadMsg)
	websockHandler.conn.WriteMessage(websocket.TextMessage, []byte(replymsg))
	return
}
func (websockHandler *websocketHandler) procWSPCInit(ctrlMsg *wssAPI.WSPMessage) (err error) {

	if wssAPI.CheckHeader(ctrlMsg.Headers) {
		seqStr := ctrlMsg.Headers["seq"]
		codeMsg := "200 OK"
		headerMsg := map[string]string{}
		payLoadMsg := ""
		strChannel, errr := websockHandler.initChannel(ctrlMsg.Headers)
		headerMsg["channel"] = strChannel
		if errr != nil {
			err = errors.New("initChannel error with inner eror: " + errr.Error())
			codeMsg = "500 Internal Server Error"
		}
		replymsg := wssAPI.EncodeWSPCtrlMsg(codeMsg, seqStr, headerMsg, payLoadMsg)
		websockHandler.conn.WriteMessage(websocket.TextMessage, []byte(replymsg))
		return
	}
	return errors.New("Check WSPInit Header Failed")
}

func (websockHandler *websocketHandler) initChannel(headers map[string]string) (strChannel string, err error) {

	ipStr := headers["host"]
	portStr := headers["port"]

	encodedIntIP := wssAPI.IP2Int(ipStr)
	channelIndex++
	strChannelIndex := strconv.Itoa(channelIndex)
	encodedIPStr := strconv.Itoa(encodedIntIP)
	strChannel = ipStr + "-" + strChannelIndex + portStr + " " + encodedIPStr
	if _, exist := channelList[websockHandler.conn]; !exist {
		channleCli, err := RTSPClient.Connect(ipStr + ":" + portStr)
		if err != nil {
			channelList[websockHandler.conn] = channleCli
			return strChannel, nil
		}
	}

	return "channel error", errors.New("channnel initial error")
	//create a tcp channel with rtsp server
	// cli := &websocket.Dialer{}
	// req := http.Header{}
	// wsURL := "ws://" + ipStr + ":" + portStr + "/ws/"
	// wsConn, _, errr := cli.Dial(wsURL, req)
	// if errr != nil {
	// 	logger.LOGE(errr.Error())
	// 	err = errr
	// 	return
	// }
	// //save Session
	// webSocketChannel[channelIndex] = wsConn

	// below is processing to build up a tcp connection to transferdata to client

	// sock := net.Dialer{Timeout: 3 * time.Second}
	// conn, errr := sock.Dial("tcp", ipStr+":"+portStr)
	// if errr != nil {
	// 	err = errr
	// }

	// if _, exist := channelRTPSocket[channelIndex]; !exist {
	// 	channelRTPSocket[channelIndex] = &conn
	// }

	//var sock = net.connect({ host: ipStr, port: portStr }, function() {
	//     okFunc()
	//     sock.connectInfo = true

	//     sock.rtpBuffer = new Buffer(2048)
	//     sock.on("data", function(data) {
	//         if (sock.outControlData) {
	//             sock.outControlData = false
	//             return
	//         }

	//         var flag = 0
	//         if (sock.SubBuffer && sock.SubBufferLen > 0) {
	//             flag = sock.SubBuffer.length - sock.SubBufferLen
	//             data.copy(sock.SubBuffer, sock.SubBufferLen, 0, flag - 1)
	//             sock.emit("rtpData", sock.SubBuffer)

	//             sock.SubBufferLen = 0
	//         }

	//         while (flag < data.length) {
	//             var len = data.readUIntBE(flag + 2, 2)
	//             sock.SubBuffer = new Buffer(4 + len)
	//             if ((flag + 4 + len) <= data.length) {
	//                 data.copy(sock.SubBuffer, 0, flag, flag + len - 1)
	//                 sock.emit("rtpData", sock.SubBuffer)
	//                 sock.SubBufferLen = 0
	//             } else {
	//                 data.copy(sock.SubBuffer, 0, flag, data.length - 1)
	//                 sock.SubBufferLen = data.length - flag
	//             }
	//             flag += 4
	//             flag += len
	//         }
	//     })

	// }).on("error", function(e) {
	//     //clean all client;
	//     console.log(e)
	// })
	// sock.setTimeout(1000 * 3, function() {
	//     if (!sock.connectInfo) {
	//         console.log("time out")
	//         failFunc("relink host[" + ip + "] time out")
	//         sock.destroy()
	//     }
	// })

	// sock.on("close", function(code) {
	//     //关闭所有子项目

	// })

}

// rtmpControlMsg for control actions
func (websockHandler *websocketHandler) rtmpControlMsg(data []byte) (err error) {
	if nil == data || len(data) < 4 {
		return errors.New("invalid msg")
	}
	//decode the rtmp ctrl msg type
	ctrlType, err := amf.AMF0DecodeInt24(data)
	if err != nil {
		logger.LOGE("get ctrl type failed")
		return
	}
	logger.LOGT(ctrlType)
	switch ctrlType {
	case WSCPlay:
		return websockHandler.ctrlPlay(data[3:])
	case WSCPlay2:
		return websockHandler.ctrlPlay2(data[3:])
	case WSCResume:
		return websockHandler.ctrlResume(data[3:])
	case WSCPause:
		return websockHandler.ctrlPause(data[3:])
	case WSCSeek:
		return websockHandler.ctrlSeek(data[3:])
	case WSCClose:
		return websockHandler.ctrlClose(data[3:])
	case WSCStop:
		return websockHandler.ctrlStop(data[3:])
	case WSCPublish:
		return websockHandler.ctrlPublish(data[3:])
	case WSCOnMetaData:
		return websockHandler.ctrlOnMetadata(data[3:])
	default:
		logger.LOGE("unknown websocket control type :from" + websockHandler.conn.RemoteAddr().String())
		return errors.New("invalid ctrl msg type :from" + websockHandler.conn.RemoteAddr().String())
	}
}

func (websockHandler *websocketHandler) ctrlPlay(data []byte) (err error) {
	st := &stPlay{}
	defer func() {
		if err != nil {
			logger.LOGE("play failed")
			err = websockHandler.sendWsStatus(websockHandler.conn, WSStatusError, NetStreamPlayFailed, st.Req)
		} else {
			websockHandler.lastCmd = WSCPlay
		}
	}()
	err = json.Unmarshal(data, st)
	if err != nil {
		logger.LOGE("invalid params")
		return err
	}
	if false == supportNewCmd(websockHandler.lastCmd, WSCPlay) {
		logger.LOGE("bad cmd")
		err = errors.New("bad cmd")
		return
	}
	//清除之前的

	switch websockHandler.lastCmd {
	case WSCClose:
		err = websockHandler.doPlay(st)
	case WSCPlay:
		err = websockHandler.doClose()
		if err != nil {
			logger.LOGE("close failed ")
			return
		}
		err = websockHandler.doPlay(st)
	case WSCPlay2:
		err = websockHandler.doClose()
		if err != nil {
			logger.LOGE("close failed ")
			return
		}
		err = websockHandler.doPlay(st)
	case WSCPause:
		err = websockHandler.doClose()
		if err != nil {
			logger.LOGE("close failed ")
			return
		}
		err = websockHandler.doPlay(st)
	default:
		logger.LOGW("invalid last cmd")
		err = errors.New("invalid last cmd")
	}
	return
}

func (websockHandler *websocketHandler) ctrlPlay2(data []byte) (err error) {
	st := &stPlay2{}
	err = json.Unmarshal(data, st)
	if err != nil {
		return err
	}

	return
}

func (websockHandler *websocketHandler) ctrlResume(data []byte) (err error) {

	st := &stResume{}
	defer func() {
		if err != nil {
			logger.LOGE("resume failed do nothing")
			websockHandler.sendWsStatus(websockHandler.conn, WSStatusStatus, NetStreamFailed, st.Req)

		} else {
			websockHandler.lastCmd = WSCPlay
		}
	}()
	if false == supportNewCmd(websockHandler.lastCmd, WSCResume) {
		logger.LOGE("bad cmd")
		err = errors.New("bad cmd")
		return
	}
	err = json.Unmarshal(data, st)
	if err != nil {
		return err
	}
	//only pase support resume
	switch websockHandler.lastCmd {
	case WSCPause:
		err = websockHandler.doResume(st)
	default:
		err = errors.New("invalid last cmd")
		logger.LOGE(err.Error())
	}
	return
}

func (websockHandler *websocketHandler) ctrlPause(data []byte) (err error) {
	st := &stPause{}
	defer func() {
		if err != nil {
			logger.LOGE("pause failed")
			websockHandler.sendWsStatus(websockHandler.conn, WSStatusStatus, NetStreamFailed, st.Req)
		} else {
			websockHandler.lastCmd = WSCPause
		}
	}()
	if false == supportNewCmd(websockHandler.lastCmd, WSCPause) {
		logger.LOGE("bad cmd")
		err = errors.New("bad cmd")
		return
	}

	err = json.Unmarshal(data, st)
	if err != nil {
		return err
	}
	switch websockHandler.lastCmd {
	case WSCPlay:
		websockHandler.doPause(st)
	case WSCPlay2:
		websockHandler.doPause(st)
	default:
		err = errors.New("invalid last cmd in pause")
		logger.LOGE(err.Error())
	}

	return
}

func (websockHandler *websocketHandler) ctrlSeek(data []byte) (err error) {
	st := &stSeek{}
	err = json.Unmarshal(data, st)
	if err != nil {
		return err
	}
	return
}

func (websockHandler *websocketHandler) ctrlClose(data []byte) (err error) {
	st := &stClose{}
	defer func() {
		if err != nil {
			websockHandler.sendWsStatus(websockHandler.conn, WSStatusError, NetStreamFailed, st.Req)
		} else {

			websockHandler.lastCmd = WSCClose
		}
	}()
	err = json.Unmarshal(data, st)
	if err != nil {
		return err
	}
	err = websockHandler.doClose()
	return
}

func (websockHandler *websocketHandler) ctrlStop(data []byte) (err error) {
	logger.LOGW("stop do the same as close now")
	st := &stStop{}
	err = json.Unmarshal(data, st)
	defer func() {
		if err != nil {
			websockHandler.sendWsStatus(websockHandler.conn, WSStatusError, NetStreamFailed, st.Req)
		} else {
			websockHandler.lastCmd = WSCClose
		}
	}()
	if err != nil {
		return err
	}
	err = websockHandler.doClose()
	return
}

func (websockHandler *websocketHandler) ctrlPublish(data []byte) (err error) {
	st := &stPublish{}
	err = json.Unmarshal(data, st)
	if err != nil {
		return err
	}
	logger.LOGE("publish not coded")
	return
}

func (websockHandler *websocketHandler) ctrlOnMetadata(data []byte) (err error) {
	logger.LOGT(string(data))
	logger.LOGW("on metadata not processed")
	return
}

func (websockHandler *websocketHandler) doClose() (err error) {
	if websockHandler.isPlaying {
		websockHandler.stopPlay()
	}
	if websockHandler.hasSink {
		websockHandler.delSink(websockHandler.streamName, websockHandler.clientID)
	}
	if websockHandler.isPublish {
		websockHandler.stopPublish()
	}
	if websockHandler.hasSource {
		websockHandler.delSource(websockHandler.streamName, websockHandler.sourceIdx)
	}
	return
}

func (websockHandler *websocketHandler) doPlay(st *stPlay) (err error) {

	logger.LOGT("play")
	websockHandler.clientID = wssAPI.GenerateGUID()
	if len(websockHandler.app) > 0 {
		websockHandler.streamName = websockHandler.app + "/" + st.Name
	} else {
		websockHandler.streamName = st.Name
	}

	err = websockHandler.addSink(websockHandler.streamName, websockHandler.clientID, websockHandler)
	if err != nil {
		logger.LOGE("add sink failed: " + err.Error())
		return
	}

	err = websockHandler.sendWsStatus(websockHandler.conn, WSStatusStatus, NetStreamPlayStart, st.Req)
	return
}

func (websockHandler *websocketHandler) doPlay2() (err error) {
	logger.LOGW("play2 not coded")
	err = errors.New("not processed")
	return
}

func (websockHandler *websocketHandler) doResume(st *stResume) (err error) {
	logger.LOGT("resume play start")
	err = websockHandler.sendWsStatus(websockHandler.conn, WSStatusStatus, NetStreamPlayStart, st.Req)
	return
}

func (websockHandler *websocketHandler) doPause(st *stPause) (err error) {
	logger.LOGT("pause do nothing")

	websockHandler.sendWsStatus(websockHandler.conn, WSStatusStatus, NetStreamPauseNotify, st.Req)
	return
}

func (websockHandler *websocketHandler) doSeek() (err error) {
	return
}

func (websockHandler *websocketHandler) doPublish() (err error) {
	return
}
