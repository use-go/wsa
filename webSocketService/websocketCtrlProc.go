// Copyright 2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webSocketService

import (
	"encoding/json"
	"errors"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/amf"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

//ProcessWSCtrlMessage to deal the RTPS over RTSP
func (websockHandler *websocketHandler) ProcessWSCtrlMessage(data []byte) (err error) {
	if nil == data || len(data) < 4 {
		return errors.New("invalid msg")
	}
	ctrlType, err := wssAPI.DecodeCtrlMsgType(data)

	switch ctrlType {
	case wssAPI.WSPCInit:
	case wssAPI.WSPCJoin:
	case wssAPI.WSPWarp:
	default:
		logger.LOGE("unknown websocket control type :from" + websockHandler.conn.RemoteAddr().String())
		return errors.New("invalid ctrl msg type :from" + websockHandler.conn.RemoteAddr().String())
	}
	return
}

// controlMsg for control actions
func (websockHandler *websocketHandler) controlMsg(data []byte) (err error) {
	if nil == data || len(data) < 4 {
		return errors.New("invalid msg")
	}
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
