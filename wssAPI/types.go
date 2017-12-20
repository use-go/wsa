package wssAPI

/*
Mark the Serivce Type
*/

const (
	OBJServerBus       = "ServerBus"
	OBJRTMPServer      = "RTMPServer"
	OBJWebSocketServer = "WebsocketServer"
	OBJBackendServer   = "BackendServer"
	OBJStreamerServer  = "StreamerServer"
	OBJRTSPServer      = "RTSPServer"
	OBJHLSServer       = "HLSServer"
	OBJDASHServer      = `DASHServer`
)

const (
	MSG_FLV_TAG            = "FLVTag"
	MSG_GetSource_NOTIFY   = "MSG.GetSource.Notify.Async"
	MSG_GetSource_Failed   = "MSG.GetSource.Failed"
	MSG_SourceClosed_Force = "MSG.SourceClosed.Force"
	MSG_PUBLISH_START      = "NetStream.Publish.Start"
	MSG_PUBLISH_STOP       = "NetStream.Publish.Stop"
	MSG_PLAY_START         = "NetStream.Play.Start"
	MSG_PLAY_STOP          = "NetStream.Play.Stop"
)
