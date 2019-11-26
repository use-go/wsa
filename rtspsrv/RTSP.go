// Copyright 2017-2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtspsrv

import (
	"net"
	"strconv"

	"github.com/use-go/websocket-streamserver/logger"
)

//RTSP Descriptions
const (
	RTSPVer             = "RTSP/1.0"
	RTSPEndLine         = "\r\n"
	RTSPRTPAVP          = "RTP/AVP"
	RTSPRTPAVPTCP       = "RTP/AVP/TCP"
	RTSPRTPAVPUDP       = "RTP/AVP/UDP"
	RTSPRawUDP          = "RTP/RAW/UDP"
	RTSPControlID0      = "track1"
	RTSPControlID1      = "track2"
	HDRCONTENTLENGTH    = "Content-Length"
	HDRACCEPT           = "Accept"
	HDRALLOW            = "Allow"
	HDRBLOCKSIZE        = "Blocksize"
	HDRCONTENTTYPE      = "Content-Type"
	HDRDATE             = "Date"
	HDRREQUIRE          = "Require"
	HDRTRANSPORTREQUIRE = "Transport-Require"
	HDRSEQUENCENO       = "SequenceNo"
	HDRCSEQ             = "CSeq"
	HDRSTREAM           = "Stream"
	HDRSESSION          = "Session"
	HDRTRANSPORT        = "Transport"
	HDRRANGE            = "Range"
	HDRUSERAGENT        = "User-Agent"

	RTSPMethodMAXLEN        = 15
	RTSPMethodDescribe      = "DESCRIBE"
	RTSPMethodAnnounce      = "ANNOUNCE"
	RTSPMethodGetParameters = "GETPARAMETERS"
	RTSPMethodOptions       = "OPTIONS"
	RTSPMethodPause         = "PAUSE"
	RTSPMethodPlay          = "PLAY"
	RTSPMethodRecord        = "RECORD"
	RTSPMethodRedirect      = "REDIRECT"
	RTSPMethodSetup         = "SETUP"
	RTSPMethodSetParameter  = "SET_PARAMETER"
	RTSPMethodTeardown      = "TEARDOWN"

	RTPDefaultMTU2 = 0xfff

	RTSPPayloadH264     = 96
	RTPVideoFreq        = 90000
	RTPAudioFreq        = 8000
	RTPSocketPacketSize = 1456

	RTSPServerName = "StreamServer_RTSP_alpha"
	CtrlTrackAudio = "track_audio"
	CtrlTrackVideo = "track_video"
)

func getRTSPStatusByCode(code int) (status string) {
	switch code {
	case 100:
		status = "Continue"
	case 200:
		status = "OK"
	case 201:
		status = "Created"
	case 202:
		status = "Accepted"
	case 203:
		status = "Non-Authoritative Information"
	case 204:
		status = "No Content"
	case 205:
		status = "Reset Content"
	case 206:
		status = "Partial Content"
	case 300:
		status = "Multiple Choices"
	case 301:
		status = "Moved Permanently"
	case 302:
		status = "Moved Temporarily"
	case 400:
		status = "Bad Request"
	case 401:
		status = "Unauthorized"
	case 402:
		status = "Payment Required"
	case 403:
		status = "Forbidden"
	case 404:
		status = "Not Found"
	case 405:
		status = "Method Not Allowed"
	case 406:
		status = "Not Acceptable"
	case 407:
		status = "Proxy Authentication Required"
	case 408:
		status = "Request Time-out"
	case 409:
		status = "Conflict"
	case 410:
		status = "Gone"
	case 411:
		status = "Length Required"
	case 412:
		status = "Precondition Failed"
	case 413:
		status = "Request Entity Too Large"
	case 414:
		status = "Request-URI Too Large"
	case 415:
		status = "Unsupported Media Type"
	case 420:
		status = "Bad Extension"
	case 450:
		status = "Invalid Parameter"
	case 451:
		status = "Parameter Not Understood"
	case 452:
		status = "Conference Not Found"
	case 453:
		status = "Not Enough Bandwidth"
	case 454:
		status = "Session Not Found"
	case 455:
		status = "Method Not Valid In This State"
	case 456:
		status = "Header Field Not Valid for Resource"
	case 457:
		status = "Invalid Range"
	case 458:
		status = "Parameter Is Read-Only"
	case 461:
		status = "Unsupported transport"
	case 500:
		status = "Internal Server Error"
	case 501:
		status = "Not Implemented"
	case 502:
		status = "Bad Gateway"
	case 503:
		status = "Service Unavailable"
	case 504:
		status = "Gateway Time-out"
	case 505:
		status = "RTSP Version Not Supported"
	case 551:
		status = "Option not supported"
	case 911:
		status = "Extended Error:"
	}
	return
}

func sendRTSPErrorReply(code int, cseq int, conn net.Conn) error {
	strOut := RTSPVer + " " + strconv.Itoa(code) + " " + getRTSPStatusByCode(code) + RTSPEndLine
	strOut += "CSeq: " + strconv.Itoa(cseq) + RTSPEndLine
	strOut += RTSPEndLine
	_, err := conn.Write([]byte(strOut))
	if err != nil {
		logger.LOGE(err.Error())
	}
	return err
}
