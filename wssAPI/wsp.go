// Copyright 2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssAPI

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/use-go/websocket-streamserver/logger"
)

// wsp Control Cmd Type
const (
	WSPVersion = "WSP/1.1"
	WSPCInit   = "INIT"
	WSPCJoin   = "JOIN"
	WSPWarp    = "WARP"
	WSPRtsp    = "RTSP"
)

//WSPMessage hold a WSP Message
type WSPMessage struct {
	Msg     string
	MsgType string
	Data    map[string]string
	Payload string
}

//WSPInit Msg Package
type WSPInit struct {
	Proto string
	Host  string
	Port  uint
	Seq   uint
}

//DecodeCtrlMsg classify the msg type
func DecodeCtrlMsg(data []byte) (ret *WSPMessage, err error) {

	if checkWSPData(data) {

		contentString := string(data)
		logger.LOGD(contentString)
		strlines := strings.Split(contentString, "\r\n")
		arryLenth := len(strlines)
		if arryLenth > 1 {
			parseredMsg := WSPMessage{Data: map[string]string{}}
			parseredMsg.MsgType = parsingWSPMsgType(strlines[0])

			// \r\n\r\n in the end ,2 empty string occured
			for i := 1; i < arryLenth-2; i++ {
				//get the version
				tmpStr := strlines[i]
				if len(tmpStr) < 2 {
					continue
				}
				//tmpStr = strings.TrimSpace(tmpStr)
				tmpStr = strings.Replace(tmpStr, " ", "", -1)
				tmpStrArry := strings.SplitN(tmpStr, ":", 2)
				parseredMsg.Data[tmpStrArry[0]] = tmpStrArry[1]

			}
			ret = &parseredMsg
			return
		}
	}
	return ret, errors.New("unknown ctrl message")
}

func checkWSPData(data []byte) bool {

	lenth := len(data)
	logger.LOGD("control data arrived： (Lenth  " + strconv.Itoa(lenth) + " bytes)")
	logger.LOGD(data)
	// Check the Data beganing
	protocalDataFormatCheck := regexp.MustCompile(`WSP/1\.1\s+\w+`)
	if protocalDataFormatCheck.Match(data) {
		return true
	}
	return false

}

func parsingWSPMsgType(strHeadline string) string {

	headLineFormatReg := regexp.MustCompile(`[\w\.\\/]+`)
	targetArray := headLineFormatReg.FindAllString(strHeadline, -1)
	logger.LOGD("protocol version : " + strHeadline)

	return targetArray[1]
}
