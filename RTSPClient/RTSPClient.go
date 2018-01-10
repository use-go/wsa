package RTSPClient

import (
	"io"
	"net"
	"net/textproto"
	"net/url"
)

type Client struct {
	DebugConn     bool
	url           *url.URL
	conn          net.Conn
	rconn         io.Reader
	requestUri    string
	cseq          uint
	streams       []*Stream
	session       string
	authorization string
	body          io.Reader
	pktque        *pktqueue.Queue
}

//Request of RTSP
type Request struct {
	Header []string
	Uri    string
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

//Connect to SS
func Connect(uri string) (conInfo *ConnectionInfo, err error) {
	var URL *url.URL
	if URL, err = url.Parse(uri); err != nil {
		return
	}

	dailer := net.Dialer{}
	var conn net.Conn
	if conn, err = dailer.Dial("tcp", URL.Host); err != nil {
		return
	}
	u2 := *URL
	u2.User = nil

	conInfo = &ConnectionInfo{
		conn:       conn,
		rconn:      conn,
		url:        URL,
		requestURI: u2.String(),
	}
	return
}
