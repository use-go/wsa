package main

import (
	"github.com/use-go/websocketStreamServer/logger"

	"github.com/use-go/websocketStreamServer/svrBus"
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
	svrBus.Start()

	ch := make(chan int)
	<-ch
}
