// Copyright 2017 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webSocketService

import (
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/use-go/websocket-streamserver/events/eStreamerEvent"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/flv"
	"github.com/use-go/websocket-streamserver/mediaTypes/mp4"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

func (websockHandler *websocketHandler) addSource(streamName string) (id int, src wssAPI.MsgHandler, err error) {
	taskAddSrc := &eStreamerEvent.EveAddSource{StreamName: streamName}
	taskAddSrc.RemoteIp = websockHandler.conn.RemoteAddr()
	err = wssAPI.HandleTask(taskAddSrc)
	if err != nil {
		logger.LOGE("add source " + streamName + " failed")
		return
	}
	websockHandler.hasSource = true
	return
}

func (websockHandler *websocketHandler) delSource(streamName string, id int) (err error) {
	taskDelSrc := &eStreamerEvent.EveDelSource{StreamName: streamName, ID: int64(id)}
	err = wssAPI.HandleTask(taskDelSrc)
	websockHandler.hasSource = false
	if err != nil {
		logger.LOGE("del source " + streamName + " failed:" + err.Error())
		return
	}
	return
}

func (websockHandler *websocketHandler) addSink(streamName, clientID string, sinker wssAPI.MsgHandler) (err error) {
	taskAddsink := &eStreamerEvent.EveAddSink{StreamName: streamName, SinkId: clientID, Sinker: sinker}
	err = wssAPI.HandleTask(taskAddsink)
	if err != nil {
		logger.LOGE(fmt.Sprintf("add sink %s %s failed :%s", streamName, clientID, err.Error()))
		return
	}
	websockHandler.hasSink = taskAddsink.Added
	return
}

func (websockHandler *websocketHandler) delSink(streamName, clientID string) (err error) {
	taskDelSink := &eStreamerEvent.EveDelSink{StreamName: streamName, SinkId: clientID}
	err = wssAPI.HandleTask(taskDelSink)
	websockHandler.hasSink = false
	if err != nil {
		logger.LOGE(fmt.Sprintf("del sink %s %s failed:\n%s", streamName, clientID, err.Error()))
	}
	logger.LOGE("del sinker")
	return
}

func (websockHandler *websocketHandler) appendFlvTag(tag *flv.FlvTag) (err error) {
	if false == websockHandler.isPlaying {
		err = errors.New("websocket client not playing")
		logger.LOGE(err.Error())
		return
	}
	tag = tag.Copy()

	//tag.Timestamp -= websockHandler.stPlay.beginTime
	//if false == websockHandler.stPlay.keyFrameWrited && tag.TagType == flv.FLV_TAG_Video {
	//	if websockHandler.stPlay.videoHeader == nil {
	//		websockHandler.stPlay.videoHeader = tag
	//	} else {
	//		if (tag.Data[0] >> 4) == 1 {
	//			websockHandler.stPlay.keyFrameWrited = true
	//		} else {
	//			return
	//		}
	//	}
	//
	//}

	if websockHandler.stPlay.audioHeader == nil && tag.TagType == flv.FLV_TAG_Audio {
		websockHandler.stPlay.audioHeader = tag
		websockHandler.stPlay.mutexCache.Lock()
		websockHandler.stPlay.cache.PushBack(tag)
		websockHandler.stPlay.mutexCache.Unlock()
		return
	}
	if websockHandler.stPlay.videoHeader == nil && tag.TagType == flv.FLV_TAG_Video {
		websockHandler.stPlay.videoHeader = tag
		websockHandler.stPlay.mutexCache.Lock()
		websockHandler.stPlay.cache.PushBack(tag)
		websockHandler.stPlay.mutexCache.Unlock()
		return
	}
	if false == websockHandler.stPlay.keyFrameWrited && tag.TagType == flv.FLV_TAG_Video {
		if false == websockHandler.stPlay.keyFrameWrited && ((tag.Data[0] >> 4) == 1) {
			websockHandler.stPlay.beginTime = tag.Timestamp
			websockHandler.stPlay.keyFrameWrited = true
		}
	}
	if false == websockHandler.stPlay.keyFrameWrited {
		return
	}

	tag.Timestamp -= websockHandler.stPlay.beginTime
	websockHandler.stPlay.mutexCache.Lock()
	defer websockHandler.stPlay.mutexCache.Unlock()
	websockHandler.stPlay.cache.PushBack(tag)

	return
}

func (websockHandler *websocketHandler) sendSlice(slice *mp4.FMP4Slice) (err error) {
	dataSend := make([]byte, len(slice.Data)+1)
	dataSend[0] = byte(slice.Type)
	copy(dataSend[1:], slice.Data)
	return websockHandler.conn.WriteMessage(websocket.BinaryMessage, dataSend)
}

func (websockHandler *websocketHandler) sendFmp4Slice(slice *mp4.FMP4Slice) (err error) {
	websockHandler.mutexWs.Lock()
	defer websockHandler.mutexWs.Unlock()
	dataSend := make([]byte, len(slice.Data)+1)
	dataSend[0] = byte(slice.Type)
	copy(dataSend[1:], slice.Data)
	err = websockHandler.conn.WriteMessage(websocket.BinaryMessage, dataSend)
	return
}
