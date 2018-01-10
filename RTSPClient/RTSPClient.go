package RTSPClient

import (
	"errors"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"time"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/use-go/websocket-streamserver/logger"
)

//Client info for RTSP Connetction
type Client struct {
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

//Connect to SS with timeout setting
func Connect(targetURL *url.URL) (cli *Client, err error) {

	dailer := net.Dialer{Timeout: 3 * time.Second}
	var conn net.Conn
	if conn, err = dailer.Dial("tcp", targetURL.Host); err != nil {
		return
	}
	u2 := *targetURL
	u2.User = nil

	cli = &Client{
		conn:       conn,
		rconn:      conn,
		url:        targetURL,
		requestURI: u2.String(),
	}
	return
}

//Read Data
func (cli *Client) Read() (str string, err error) {
	buffer := make([]byte, 4096)
	nb, err := cli.conn.Read(buffer)
	if err != nil || nb <= 0 {
		logger.LOGE("socket read failed", err)
		return "", errors.New("socket read failed")
	}
	return string(buffer[:nb]), nil

}

//Write Data
func (cli *Client) Write(message string) (cnt int, err error) {
	cli.cseq++
	count, e := cli.conn.Write([]byte(message))
	if e != nil {
		err = errors.New("socket write failed")
	}
	return count, nil
}

//ReadPacket handle RTP Packet
func (cli *Client) ReadPacket() (pkt av.Packet, err error) {
	return cli.pktque.ReadPacket()
}
