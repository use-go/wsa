package RTSPClient

import (
	"errors"
	"io"
	"net"
	"net/textproto"
	"net/url"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/use-go/websocket-streamserver/logger"
)

//SocketChannel info for RTSP Connetction
type SocketChannel struct {
	DebugConn     bool
	url           *url.URL
	conn          net.Conn
	rconn         io.Reader
	requestURI    string
	cseq          uint
	streams       []*Stream
	session       string
	authorization string
	body          io.Reader
	pktque        *pubsub.QueueCursor
}

//Request of RTSP
type Request struct {
	Header []string
	URI    string
	Method string
}

//Response of RTSP
type Response struct {
	BlockLength int
	Block       []byte
	BlockNo     int

	StatusCode    int
	Header        textproto.MIMEHeader
	ContentLength int
	Body          []byte
}

//ForwardToSer Data
func (cli *SocketChannel) ForwardToSer(message string) (cnt int, err error) {
	cli.cseq++
	count, e := cli.conn.Write([]byte(message))
	if e != nil {
		err = errors.New("socket write failed")
	}
	return count, nil
}

//Read Data
func (cli *SocketChannel) Read() (str string, err error) {
	buffer := make([]byte, 4096)
	nb, err := cli.conn.Read(buffer)
	if err != nil || nb <= 0 {
		logger.LOGE("socket read failed", err)
		return "", errors.New("socket read failed")
	}
	return string(buffer[:nb]), nil

}

//Write Data
func (cli *SocketChannel) Write(message string) (cnt int, err error) {
	cli.cseq++
	count, e := cli.conn.Write([]byte(message))
	if e != nil {
		err = errors.New("socket write failed")
	}
	return count, nil
}

// WriteRequest Message to RTSP
func (cli *SocketChannel) WriteRequest(req Request) (err error) {

	return
}

//ReadResponse handle rtsp response
func (cli *SocketChannel) ReadResponse() (res Response, err error) {

	return
}

//Options RTSP
func (cli *SocketChannel) Options() (err error) {
	if err = cli.WriteRequest(Request{
		Method: "OPTIONS",
		URI:    cli.requestURI,
	}); err != nil {
		return
	}
	if _, err = cli.ReadResponse(); err != nil {
		return
	}
	return
}

//Describe RTSP
func (cli *SocketChannel) Describe() (streams []av.CodecData, err error) {

	return
}

//Setup RTSP
func (cli *SocketChannel) Setup(streams []int) (err error) {

	return
}

//Play Video
func (cli *SocketChannel) Play() (err error) {
	req := Request{
		Method: "PLAY",
		URI:    cli.requestURI,
	}
	req.Header = append(req.Header, "Session: "+cli.session)
	if err = cli.WriteRequest(req); err != nil {
		return
	}
	return
}

//ReadPacket handle RTP Packet
func (cli *SocketChannel) ReadPacket() (pkt av.Packet, err error) {
	return cli.pktque.ReadPacket()
}
