// Copyright 2017-2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	For detailed information on the Real Time Streaming Protocol (RTSP) protocol,
	see the RFC 2326: https://www.ietf.org/rfc/rfc2326.txt
*/

package RTSPClientProxy

import (
	"errors"
	"io"
	"net"
	"net/url"
	"regexp"

	"github.com/use-go/websocket-streamserver/logger"
)

//ConnectionInfo struct
type ConnectionInfo struct {
	DebugConn     bool
	url           *url.URL
	conn          net.Conn
	rconn         io.Reader
	requestURI    string
	cseq          uint
	session       string
	authorization string
	body          io.Reader
}

//Connect to SS
func Connect(uri string) (conInfo *ConnectionInfo, err error) {
	var URL *url.URL
	if URL, err = url.Parse(uri); err != nil {
		return
	}

	dailer := net.Dialer{}
	var conn net.Conn
	if conn, err = dailer.Dial("tcp", URL.Host); err != nil {
		return
	}
	u2 := *URL
	u2.User = nil

	conInfo = &ConnectionInfo{
		conn:       conn,
		rconn:      conn,
		url:        URL,
		requestURI: u2.String(),
	}
	return
}

func checkRequestContent(content string) (err error) {

	requestRegex := regexp.MustCompile(`([A-Z]*) (.*) RTSP/(.*)\r\n`)
	requestParams := requestRegex.FindStringSubmatch(content)
	if len(requestParams) != 4 {
		logger.LOGE("Could not understand request: %s", content)
		return errors.New("Could not understand request")
	}
	return
}
