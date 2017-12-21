// Copyright 2017 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

/*
life circle of app
*/
import (
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/process"
)

func main() {
	initLogger()
	runServer()
}

func initLogger() {
	logger.SetFlags(logger.LOG_SHORT_FILE | logger.LOG_TIME)
	logger.SetLogLevel(logger.LOG_LEVEL_TRACE)
	logger.OutputInCmd(true)
}

func runServer() {
	process.Run()

	ch := make(chan int)
	<-ch
}
