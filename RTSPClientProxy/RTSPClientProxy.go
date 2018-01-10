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

	"github.com/use-go/websocket-streamserver/RTSPClient"
	"github.com/use-go/websocket-streamserver/logger"
)

//Open Stream
func Open(uri string) (cli *RTSPClient.Client, err error) {

	url, err := parserURL(uri)

	if err != nil {
		return nil, errors.New("URL error")
	}

	_cli, err := RTSPClient.Connect(url)
	if err != nil {
		return
	}

	streams, err := _cli.Describe()
	if err != nil {
		return
	}

	setup := []int{}
	for i := range streams {
		setup = append(setup, i)
	}

	err = _cli.Setup(setup)
	if err != nil {
		return
	}

	err = _cli.Play()
	if err != nil {
		return
	}

	cli = _cli
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
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return
	}
	return
}
