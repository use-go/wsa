// Copyright 2017-2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


// RTSP请求的格式
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | 方法  | <空格>  | URL | <空格>  | 版本  | <回车换行>  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | 头部字段名  | : | <空格>  |      值     | <回车换行>  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                   ......                              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | 头部字段名  | : | <空格>  |      值     | <回车换行>  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | <回车换行>                                            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  实体内容                                             |
// |  （通常不用）                                         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+


package RTSPClient

import (
	"io"
	"net"
	"net/textproto"
	"net/url"
	"fmt"
	"strings"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/pubsub"
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



//Forward Data
func (cli *SocketChannel) Forward(message string) (str string, err error) {
	
	//send message
 	cnt,err :=	write(cli.conn ,message)
	if cnt <1 || err !=nil {	
		return "write socket failed",err
	}
	//wait to receive
   return read(cli.conn)
}


func (cli *SocketChannel) writeLine(line string) (err error) {
	if cli.DebugConn {
		fmt.Print("> ", line)
	}
	write(cli.conn, line)
	//_, err = fmt.Fprint(cli.conn, line)
	return
}

// WriteRequest Message to RTSP
func (cli *SocketChannel) WriteRequest(req Request) (err error) {
	cli.cseq++
	req.Header = append(req.Header, fmt.Sprintf("CSeq: %d", cli.cseq))
	if err = cli.writeLine(fmt.Sprintf("%s %s RTSP/1.0\r\n", req.Method, req.URI)); err != nil {
		return
	}
	for _, v := range req.Header {
		if err = cli.writeLine(fmt.Sprint(v, "\r\n")); err != nil {
			return
		}
	}
	if err = cli.writeLine("\r\n"); err != nil {
		return
	}

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

	var res Response

	for i := 0; i < 2; i++ {
		req := Request{
			Method: "DESCRIBE",
			URI: cli.requestURI,
		}
		if cli.authorization != "" {
			req.Header = append(req.Header, "Authorization: "+cli.authorization)
		}
		if err = cli.WriteRequest(req); err != nil {
			return
		}
		if res, err = cli.ReadResponse(); err != nil {
			return
		}
		if res.StatusCode == 200 {
			break
		}
	}
	if res.ContentLength == 0 {
		err = fmt.Errorf("rtsp: Describe failed, StatusCode=%d", res.StatusCode)
		return
	}

	body := string(res.Body)

	if cli.DebugConn {
		fmt.Println("<", body)
	}

	cli.streams = []*Stream{}
	for _, info := range sdp.Decode(body) {
		stream := &Stream{Sdp: info}

		if false {
			fmt.Println("sdp:", info.TimeScale)
		}

		if info.PayloadType >= 96 && info.PayloadType <= 127 {
			switch info.Type {
			case av.H264:
				var sps, pps []byte
				for _, nalu := range info.SpropParameterSets {
					if len(nalu) > 0 {
						switch nalu[0]&0x1f {
						case 7:
							sps = nalu
						case 8:
							pps = nalu
						}
					}
				}
				if len(sps) > 0 && len(pps) > 0 {
					if stream.CodecData, err = h264parser.NewCodecDataFromSPSAndPPS(sps, pps); err != nil {
						err = fmt.Errorf("rtsp: h264 sps/pps invalid: %s", err)
						return
					}
				} else {
					err = fmt.Errorf("rtsp: h264 sdp sprop-parameter-sets invalid: missing sps or pps")
					return
				}

			case av.AAC:
				if len(info.Config) == 0 {
					err = fmt.Errorf("rtsp: aac sdp config missing")
					return
				}
				if stream.CodecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(info.Config); err != nil {
					err = fmt.Errorf("rtsp: aac sdp config invalid: %s", err)
					return
				}
			}
		} else {
			switch info.PayloadType {
			case 0:
				stream.CodecData = codec.NewPCMMulawCodecData()

			default:
				err = fmt.Errorf("rtsp: PayloadType=%d unsupported", info.PayloadType)
				return
			}
		}

		cli.streams = append(cli.streams, stream)
	}

	for _, stream := range cli.streams {
		streams = append(streams, stream)
	}
//	cli.pktque = &pktqueue.Queue{
//		Poll: cli.poll,
//	}
//	cli.pktque.Alloc(streams)

	return
}

//Setup RTSP
func (cli *SocketChannel) Setup(streams []int) (err error) {
	for _, si := range streams {
		uri := ""
		control := cli.streams[si].Sdp.Control
		if strings.HasPrefix(control, "rtsp://") {
			uri = control
		} else {
			uri = cli.requestURI+"/"+control
		}
		req := Request{Method: "SETUP", URI: uri}
		req.Header = append(req.Header, fmt.Sprintf("Transport: RTP/AVP/TCP;unicast;interleaved=%d-%d", si*2, si*2+1))
		if cli.session != "" {
			req.Header = append(req.Header, "Session: "+cli.session)
		}
		if err = cli.WriteRequest(req); err != nil {
			return
		}
		if _, err = cli.ReadResponse(); err != nil {
			return
		}
	}
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
