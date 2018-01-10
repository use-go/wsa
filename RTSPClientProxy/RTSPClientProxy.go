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

func checkRequestContent(content string) (err error) {

	requestRegex := regexp.MustCompile(`([A-Z]*) (.*) RTSP/(.*)\r\n`)
	requestParams := requestRegex.FindStringSubmatch(content)
	if len(requestParams) != 4 {
		logger.LOGE("Could not understand request: %s", content)
		return errors.New("Could not understand request")
	}
	return
}

func parseruRL(uri string) (targetURL *url.URL, err error) {

	// Parse the URL and ensure there are no errors.
	url, err := url.Parse(uri)
	if err != nil {
		logger.LOGE("Could not parse request uri: %s. %s\n", uri, err.Error())
		return nil, err
	}
	return url, nil
}
