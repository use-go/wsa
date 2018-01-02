package webSocketService

import (
	"encoding/json"

	"github.com/gorilla/websocket"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

// WS Status
const (
	WSStatusStatus = "status"
	WSStatusError  = "error"
)

//1byte type
const (
	WSPktAudio   = 8
	WSPktVideo   = 9
	WSPktControl = 18
)

// WS Control Cmd
const (
	WSCConnect      = 0
	WSCPlay         = 1
	WSCPlay2        = 2
	WSCResume       = 3
	WSCPause        = 4
	WSCSeek         = 5
	WSCStop         = 6
	WSCClose        = 7
	WSCPublish      = 8
	WSCOnMetaData   = 9
	WSCStreamBegin  = 11
	WSCStreamEnd    = 12
)

var cmdsMap map[int]*wssAPI.Set

func init() {
	cmdsMap = make(map[int]*wssAPI.Set)
	//初始状态close，可以play,close,publish
	{
		tmp := wssAPI.NewSet()
		tmp.Add(WSCPlay)
		tmp.Add(WSCPlay2)
		tmp.Add(WSCClose)
		tmp.Add(WSCPublish)
		cmdsMap[WSCClose] = tmp
	}
	//play 可以close pause seek
	{
		tmp := wssAPI.NewSet()
		tmp.Add(WSCPause)
		tmp.Add(WSCPlay)
		tmp.Add(WSCPlay2)
		tmp.Add(WSCSeek)
		tmp.Add(WSCClose)
		cmdsMap[WSCPlay] = tmp
	}
	//play2 =play
	{
		tmp := wssAPI.NewSet()
		tmp.Add(WSCPause)
		tmp.Add(WSCPlay)
		tmp.Add(WSCPlay2)
		tmp.Add(WSCSeek)
		tmp.Add(WSCClose)
		cmdsMap[WSCPlay2] = tmp
	}
	//pause
	{
		tmp := wssAPI.NewSet()
		tmp.Add(WSCResume)
		tmp.Add(WSCPlay)
		tmp.Add(WSCPlay2)
		tmp.Add(WSCClose)
		cmdsMap[WSCPause] = tmp
	}
	//publish
	{
		tmp := wssAPI.NewSet()
		tmp.Add(WSCClose)
		cmdsMap[WSCPublish] = tmp
	}
}

func supportNewCmd(cmdOld, cmdNew int) bool {
	_, exist := cmdsMap[cmdOld]
	if false == exist {
		return false
	}
	return cmdsMap[cmdOld].Has(cmdNew)
}

// SendWsControl Control command
func SendWsControl(conn *websocket.Conn, ctrlType int, data []byte) (err error) {
	dataSend := make([]byte, len(data)+4)
	dataSend[0] = WSPktControl
	dataSend[1] = byte((ctrlType >> 16) & 0xff)
	dataSend[2] = byte((ctrlType >> 8) & 0xff)
	dataSend[3] = byte((ctrlType >> 0) & 0xff)
	copy(dataSend[4:], data)
	return conn.WriteMessage(websocket.BinaryMessage, dataSend)
}

//SendWsStatus code to client
func SendWsStatus(conn *websocket.Conn, level, code string, req int) (err error) {
	st := &stResult{Level: level, Code: code, Req: req}
	dataJSON, err := json.Marshal(st)
	if err != nil {
		logger.LOGE(err.Error())
		return
	}
	dataSend := make([]byte, len(dataJSON)+4)
	dataSend[0] = WSPktControl
	dataSend[1] = 0
	dataSend[2] = 0
	dataSend[3] = 0
	copy(dataSend[4:], dataJSON)
	err = conn.WriteMessage(websocket.BinaryMessage, dataSend)
	return
}

type stPlay struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	Len   int    `json:"len"`
	Reset int    `json:"reset"`
	Req   int    `json:"req"`
}

type stPlay2 struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	Len   int    `json:"len"`
	Reset int    `json:"reset"`
	Req   int    `json:"req"`
}

type stResume struct {
	Req int `json:"req"`
}

type stPause struct {
	Req int `json:"req"`
}

type stSeek struct {
	Offset int `json:"offset"`
	Req    int `json:"req"`
}

type stClose struct {
	Req int `json:"req"`
}

type stStop struct {
	Req int `json:"req"`
}

type stPublish struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Req  int    `json:"req"`
}

type stResult struct {
	Level string `json:"level"`
	Code  string `json:"code"`
	Req   int    `json:"req"`
}

// const string resource
const (
	NetconnectionCallFailed           = "NetConnection.Call.Failed"
	NetconnectionConnectAppShutdown   = "NetConnection.Connect.AppShutdown"
	NetconnectionConnectClosed        = "NetConnection.Connect.Closed"
	NetconnectionConnectFailed        = "NetConnection.Connect.Failed"
	NetconnectionConnectIdleTimeout   = "NetConnection.Connect.IdleTimeout"
	NetconnectionConnectInvalidApp    = "NetConnection.Connect.InvalidApp"
	NetconnectionConnectRejected      = "NetConnection.Connect.Rejected"
	NetconnectionConnectSuccess       = "NetConnection.Connect.Success"
	NetStreamBufferEmpty              = "NetStream.Buffer.Empty"
	NetStreamBufferFlush              = "NetStream.Buffer.Flush"
	NetStreamBufferFull               = "NetStream.Buffer.Full"
	NetStreamFailed                   = "NetStream.Failed"
	NetStreamPauseNotify              = "NetStream.Pause.Notify"
	NetStreamPlayFailed               = "NetStream.Play.Failed"
	NetStreamPlayFileStructureInvalid = "NetStream.Play.FileStructureInvalid"
	NetStreamPlayPublishNotify        = "NetStream.Play.PublishNotify"
	NetStreamPlayReset                = "NetStream.Play.Reset"
	NetStreamPlayStart                = "NetStream.Play.Start"
	NetStreamPlayStop                 = "NetStream.Play.Stop"
	NetStreamPlayStreamNotFound       = "NetStream.Play.StreamNotFound"
	NetStreamPlayUnpublishNotify      = "NetStream.Play.UnpublishNotify"
	NetStreamPublishBadName           = "NetStream.Publish.BadName"
	NetStreamPublishIdle              = "NetStream.Publish.Idle"
	NetStreamPublishStart             = "NetStream.Publish.Start"
	NetStreamRecordAlreadyExists      = "NetStream.Record.AlreadyExists"
	NetStreamRecordFailed             = "NetStream.Record.Failed"
	NetStreamRecordNoAccess           = "NetStream.Record.NoAccess"
	NetStreamRecordStart              = "NetStream.Record.Start"
	NetStreamRecordStop               = "NetStream.Record.Stop"
	NetStreamSeekFailed               = "NetStream.Seek.Failed"
	NetStreamSeekInvalidTime          = "NetStream.Seek.InvalidTime"
	NetStreamSeekNotify               = "NetStream.Seek.Notify"
	NetStreamStepNotify               = "NetStream.Step.Notify"
	NetStreamUnpauseNotify            = "NetStream.Unpause.Notify"
	NetStreamUnpublishSuccess         = "NetStream.Unpublish.Success"
	NetStreamVideoDimensionChange     = "NetStream.Video.DimensionChange"
)
