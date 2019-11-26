// Copyright 2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssapi

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
	WSPWrap    = "WRAP"
	WSPRtsp    = "RTSP"
)

//WSPMessage hold a WSP Message
type WSPMessage struct {
	Msg     string
	MsgType string
	Headers map[string]string
	Payload string
}

//WSPInit Msg Package
type WSPInit struct {
	Proto string
	Host  string
	Port  uint
	Seq   uint
}

//DecodeWSPCtrlMsg decode ctrl msg & classify the msg type
func DecodeWSPCtrlMsg(data []byte) (ret *WSPMessage, err error) {

	if checkWSPData(data) {
		contentString := string(data)
		logger.LOGD(contentString)

		firstIndex := strings.Index(contentString, "\r\n\r\n")
		strlines := strings.Split(contentString[:firstIndex], "\r\n")
		arryLenth := len(strlines)
		// \r\n\r\n in the end ,2 empty string occured, Must more 3
		if arryLenth > 1 {
			parseredMsg := WSPMessage{Headers: map[string]string{}}
			parseredMsg.MsgType = parsingWSPMsgType(strlines[0])

			for i := 1; i < arryLenth; i++ {
				//get the version
				tmpStr := strlines[i]
				if len(tmpStr) < 2 {
					continue
				}
				tmpStrArry := strings.SplitN(tmpStr, ":", 2)
				keyStr := strings.TrimSpace(tmpStrArry[0])
				valueStr := strings.TrimSpace(tmpStrArry[1])
				parseredMsg.Headers[keyStr] = valueStr

			}

			//if this the wrapperd rtsp message
			if WSPWrap == strings.ToUpper(parseredMsg.MsgType) {
				parseredMsg.Payload = contentString[firstIndex+4:]
				logger.LOGD(parseredMsg.Payload)
			}
			parseredMsg.Msg = contentString
			ret = &parseredMsg
			return
		}
	}
	return ret, errors.New("unknown ctrl message")
}

func checkWSPData(data []byte) bool {

	lenth := len(data)
	logger.LOGD("control data arrived： (Lenth:  " + strconv.Itoa(lenth) + " bytes)")
	logger.LOGD(data)
	// Check the Data beganing
	protocalDataFormatCheck := regexp.MustCompile(`(?s)WSP/1\.1\s+\w+\r\n.+?\r\n\r\n$`)
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

//CheckHeader ok or not
func CheckHeader(headers map[string]string) bool {
	if len(headers) < 2 {
		return false
	}
	seqInt, err := strconv.Atoi(headers["seq"])

	if err != nil || seqInt < 0 {
		return false
	}

	ipformat := `((2[0-4]\d|25[0-5]|[01]?\d\d?)\.){3}(2[0-4]\d|25[0-5]|[01]?\d\d?)`
	ipformatCheck := regexp.MustCompile(ipformat)

	if ipformatCheck.MatchString(headers["host"]) {
		return true
	}

	return false
}

//EncodeWSPCtrlMsg return client request msg
func EncodeWSPCtrlMsg(code, seq string, headers map[string]string, payload string) string {

	msg := "WSP/1.1 " + code + "\r\n"
	msg += "seq: " + seq + "\r\n"
	if len(headers) > 0 {
		for key, value := range headers {
			msg += key + ": " + value + "\r\n"
		}
	}
	msg += "\r\n"
	if payload == "" {
		msg += "\r\n"
	}else{
		msg += payload
	}
	return msg
}
