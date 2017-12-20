package main

import (
	"github.com/use-go/websocket-streamserver/logger"

	"github.com/use-go/websocket-streamserver/process"
)

func main() {
	initLogger()
	startServers()
}

func initLogger() {
	logger.SetFlags(logger.LOG_SHORT_FILE | logger.LOG_TIME)
	logger.SetLogLevel(logger.LOG_LEVEL_TRACE)
	logger.OutputInCmd(true)
}

func startServers() {
	process.Start()

	ch := make(chan int)
	<-ch
}
