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
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/use-go/websocket-streamserver/logger"
)

func checkRequestContent(content string) (err error) {

	requestRegex := regexp.MustCompile(`([A-Z]*) (.*) RTSP/(.*)\r\n`)
	requestParams := requestRegex.FindStringSubmatch(content)
	if len(requestParams) != 4 {
		logger.LOGE("Could not understand request: %s", content)
		return errors.New("Could not understand request")
	}
	return
}

func parserURL(uri string) (targetURL *url.URL, err error) {
	// Parse the URL and ensure there are no errors.
	url, err := url.Parse(uri)
	if err != nil {
		logger.LOGE("Could not parse request uri: %s. %s\n", uri, err.Error())
		return nil, err
	}
	return url, nil
}

func parseRTSPVersion(s string) (proto string, major int, minor int, err error) {
	parts := strings.SplitN(s, "/", 2)
	proto = parts[0]
	parts = strings.SplitN(parts[1], ".", 2)
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return
	}
	if minor, err = strconv.Atoi(parts[0]); err != nil {
		return
	}
	return
}
