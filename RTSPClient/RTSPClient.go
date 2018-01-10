package RTSPClient

import (
	"io"
	"net"
	"net/textproto"
	"net/url"
	"time"

	"github.com/nareix/joy4/av/pktque"
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
	pktque        *pktque.Buf
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
