
// RTSP响应的格式
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | 版本  | <空格>  | 状态码 | <空格>  | 状态描述  | <回车换行> |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | 头部字段名  | : | <空格>  |      值           | <回车换行>  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           ......                            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | 头部字段名  | : | <空格>  |      值           | <回车换行>  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | <回车换行>                                                  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  实体内容                                                   |
// |  （有些响应不用）                                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

package RTSPService

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/mediaTypes/aac"
)

func (rtspHandler *RTSPHandler) sendErrorReply(lines []string, code int) (err error) {
	cseq := getCSeq(lines)
	strOut := RTSPVer + " " + strconv.Itoa(code) + " " + getRTSPStatusByCode(code) + RTSPEndLine
	strOut += "CSeq: " + strconv.Itoa(cseq) + RTSPEndLine
	strOut += RTSPEndLine
	err = rtspHandler.send([]byte(strOut))
	return
}

func (rtspHandler *RTSPHandler) serveOptions(lines []string) (err error) {
	cseq := getCSeq(lines)
	cliSession := rtspHandler.getSession(lines)
	hasSession := false
	if len(cliSession) > 0 && cliSession != rtspHandler.session {
		hasSession = true
	}
	str := RTSPVer + " " + strconv.Itoa(200) + " " + getRTSPStatusByCode(200) + RTSPEndLine
	str += HDRCSEQ + ": " + strconv.Itoa(cseq) + RTSPEndLine
	if hasSession {
		str += "Session: " + rtspHandler.session + RTSPEndLine
	}
	str += "Public: OPTIONS, DESCRIBE, GET_PARAMETER, PAUSE, PLAY, SETUP, SET_PARAMETER, TEARDOWN " + RTSPEndLine 
	str += "Server: GStreamer RTSP server " + RTSPEndLine
	//Date: Mon, 15 Jan 2018 09:20:38 GMT
	str += "Date: "+  time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")  + RTSPEndLine + RTSPEndLine
	err = rtspHandler.send([]byte(str))

	if rtspHandler.tcpTimeout {
		userAgent := getHeaderByName(lines, "User-Agent:", true)
		if len(userAgent) > 0 {
			rtspHandler.tcpTimeout = !strings.Contains(userAgent, "LibVLC")
		}
	}
	return
}

func (rtspHandler *RTSPHandler) serveDescribe(lines []string) (err error) {
	cseq := getCSeq(lines)
	str := removeSpace(lines[0])
	strSpaces := strings.Split(str, " ")
	if len(strSpaces) < 2 {
		return rtspHandler.sendErrorReply(lines, 455)
	}

	_, rtspHandler.streamName, err = rtspHandler.parseURL(strSpaces[1])
	if err != nil {
		logger.LOGE(err.Error())
		return rtspHandler.sendErrorReply(lines, 455)
	}

	//添加槽
	if false == rtspHandler.addSink() {
		err = errors.New("add sink failed:" + rtspHandler.streamName)
		return
	}
	//看是否需要sdp
	acceptLine := getHeaderByName(lines, "Accept", false)
	needSdp := false
	if len(acceptLine) > 0 {
		if strings.Contains(acceptLine, "application/sdp") {
			needSdp = true
		}
	}
	if needSdp {
		//等待直到有音视频头或超时2s
		counts := 0
		needSdp = false
		for {
			counts++
			if counts > 200 {
				break
			}
			if rtspHandler.audioHeader != nil && rtspHandler.videoHeader != nil {
				needSdp = true
				break
			}
			rtspHandler.mutexVideo.RLock()
			if nil != rtspHandler.videoCache && rtspHandler.videoCache.Len() > 0 {
				rtspHandler.mutexVideo.RUnlock()
				needSdp = true
				break
			}
			rtspHandler.mutexVideo.RUnlock()
			time.Sleep(10 * time.Millisecond)
		}
	}
	sdp := ""
	if needSdp {
		//生成sdp
		ok := false
		sdp, ok = generateSDP(rtspHandler.videoHeader.Copy(), rtspHandler.audioHeader.Copy())
		if false == ok {
			logger.LOGE("generate sdp failed")
			needSdp = false
		}
	}

	strOut := RTSPVer + " " + strconv.Itoa(200) + " " + getRTSPStatusByCode(200) + RTSPEndLine
	strOut += HDRCSEQ + ": " + strconv.Itoa(cseq) + RTSPEndLine

	if needSdp {
		strOut += "Content-Type: application/sdp" + RTSPEndLine
		strOut += "Content-Length: " + strconv.Itoa(len(sdp)+len(RTSPEndLine)) + RTSPEndLine + RTSPEndLine
		strOut += sdp + RTSPEndLine
	} else {
		//strOut += RTSPEndLine
		logger.LOGE("can not generate sdp")
		err = rtspHandler.sendErrorReply(lines, 415)
		return
	}

	err = rtspHandler.send([]byte(strOut))
	return
}

func (rtspHandler *RTSPHandler) parseURL(url string) (port int, streamName string, err error) {
	if false == strings.HasPrefix(url, "rtsp://") {
		err = errors.New("bad rtsp url:" + url)
		logger.LOGE(err.Error())
		return
	}
	sub := strings.TrimPrefix(url, "rtsp://")
	subs := strings.Split(sub, "/")
	if len(subs) < 2 {
		err = errors.New("bad rtsp url:" + url)
		logger.LOGE(err.Error())
		return
	}
	streamName = strings.TrimPrefix(sub, subs[0])
	streamName = strings.TrimPrefix(streamName, "/")
	if false == strings.Contains(subs[0], ":") {
		port = 554
	} else {
		subs = strings.Split(subs[0], ":")
		if len(subs) != 2 {
			err = errors.New("bad rtsp url:" + url)
			logger.LOGE(err.Error())
			return
		}
		port, err = strconv.Atoi(subs[1])
	}
	return
}

func (rtspHandler *RTSPHandler) getSession(lines []string) (cliSession string) {
	tmp := getHeaderByName(lines, "Session:", true)
	if len(tmp) > 0 {
		tmp = strings.Replace(tmp, " ", "", -1)
		fmt.Sscanf(tmp, "Session:%s", &cliSession)
		logger.LOGD(cliSession)
		if strings.Contains(cliSession, ";") {
			cliSession = strings.Split(cliSession, ";")[0]
		}
	}
	return
}

func (rtspHandler *RTSPHandler) serveSetup(lines []string) (err error) {
	cseq := getCSeq(lines)
	if rtspHandler.isPlaying {
		return rtspHandler.sendErrorReply(lines, 405)
	}
	cliSession := rtspHandler.getSession(lines)
	if len(cliSession) > 0 && strings.Compare(cliSession, rtspHandler.session) != 0 {
		//session错误
		logger.LOGE("session wrong")
		return rtspHandler.sendErrorReply(lines, 454)
	}
	//取出track
	trackName := ""
	{
		strLine := removeSpace(lines[0])
		subs := strings.Split(strLine, " ")
		if len(subs) < 2 {
			logger.LOGE("setup failed")
			return rtspHandler.sendErrorReply(lines, 400)
		}
		logger.LOGD(subs)
		subs = strings.Split(subs[1], "/")
		trackName = subs[len(subs)-1]
	}
	if strings.Compare(trackName, CtrlTrackAudio) != 0 && strings.Compare(trackName, CtrlTrackVideo) != 0 {
		logger.LOGE("track :" + trackName + " not found")
		return rtspHandler.sendErrorReply(lines, 404)
	}

	rtspHandler.mutexTracks.RLock()
	track, exist := rtspHandler.tracks[trackName]
	if exist {
		logger.LOGE("track :" + trackName + " has setuped,sotp play ")
		rtspHandler.stopPlayThread()
	} else {
		track = &trackInfo{}
	}
	rtspHandler.mutexTracks.RUnlock()
	logger.LOGD("lok track")
	rtspHandler.mutexTracks.Lock()
	logger.LOGD("unlock track")
	defer rtspHandler.mutexTracks.Unlock()
	//取出协议类型和端口号
	{

		//video 90000
		if strings.Compare(trackName, CtrlTrackAudio) == 0 {
			if rtspHandler.audioHeader != nil {
				asc := aac.GenerateAudioSpecificConfig(rtspHandler.audioHeader.Data[2:])
				track.clockRate = uint32(asc.SamplingFrequency)
			}
		}
		//audio 90000
		if strings.Compare(trackName, CtrlTrackVideo) == 0 {
			track.clockRate = RTPH264Freq
		}
		strTransport := getHeaderByName(lines, HDRTRANSPORT, true)
		if len(strTransport) == 0 {
			logger.LOGE("setup failed,no transport")
			return rtspHandler.sendErrorReply(lines, 400)
		}
		strTransport = removeSpace(strTransport)
		strTransport = strings.TrimPrefix(strTransport, HDRTRANSPORT)
		strTransport = strings.TrimPrefix(strTransport, ":")
		strTransport = strings.TrimPrefix(strTransport, " ")
		subs := strings.Split(strTransport, ";")

		logger.LOGT(subs)
		if len(subs) != 3 {
			logger.LOGT(len(subs))
			logger.LOGE("setup failed,parse transport failed")
			return rtspHandler.sendErrorReply(lines, 400)
		}
		track.transPort = subs[0]
		if strings.Compare(subs[1], "unicast") == 0 {
			track.unicast = true
		} else {
			track.unicast = false
		}
		if subs[0] == RTSPRTPAVP || subs[0] == RTSPRTPAVPUDP {
			track.transPort = "udp"
			if false == strings.HasPrefix(subs[2], "client_port=") {
				logger.LOGE("udp not found client port")
				return rtspHandler.sendErrorReply(lines, 461)
			}
			cliports := strings.TrimPrefix(subs[2], "client_port=")
			ports := strings.Split(cliports, "-")
			if len(ports) != 2 {
				logger.LOGE("udp not found client port")
				return rtspHandler.sendErrorReply(lines, 461)
			}
			track.RTPCliPort, err = strconv.Atoi(ports[0])

			if err != nil {
				logger.LOGE(err.Error())
				return rtspHandler.sendErrorReply(lines, 461)
			}
			track.RTCPCliPort, err = strconv.Atoi(ports[1])
			if err != nil {
				logger.LOGE(err.Error())
				return rtspHandler.sendErrorReply(lines, 461)
			}
			ok := false

			track.RTPSvrPort, track.RTCPSvrPort, track.RTPSvrConn, track.RTCPSvrConn, ok = genRTPRTCP()
			if false == ok {
				logger.LOGE(err.Error())
				return rtspHandler.sendErrorReply(lines, 461)
			}

		} else if subs[0] == RTSPRTPAVPTCP {
			track.transPort = "tcp"
			if false == strings.Contains(strTransport, "interleaved=") {
				logger.LOGE("not found tcp interleaved")
				return rtspHandler.sendErrorReply(lines, 461)
			}
			strsubs := strings.Split(strTransport, "interleaved=")
			if len(strsubs) != 2 {
				logger.LOGE("not found tcp interleaved")
				return rtspHandler.sendErrorReply(lines, 461)
			}

			strsubs = strings.Split(strsubs[1], "-")
			if len(strsubs) != 2 {
				logger.LOGE("not found tcp interleaved")
				return rtspHandler.sendErrorReply(lines, 461)
			}
			track.RTPChannel, err = strconv.Atoi(strsubs[0])
			if err != nil {
				logger.LOGE(err.Error())
				return rtspHandler.sendErrorReply(lines, 461)
			}
			track.RTCPChannel, err = strconv.Atoi(strsubs[1])
			if err != nil {
				logger.LOGE(err.Error())
				return rtspHandler.sendErrorReply(lines, 461)
			}
			logger.LOGT(track.RTCPChannel)
		} else {
			logger.LOGE(subs[0] + " not support now")
			return rtspHandler.sendErrorReply(lines, 551)
		}
	}
	track.trackID = trackName
	rtspHandler.tracks[trackName] = track
	track.firstSeq = rand.Intn(0xffff)
	track.mark = true
	track.ssrc = uint32(rand.Intn(0xffff))
	//返回结果

	strOut := RTSPVer + " " + strconv.Itoa(200) + " " + getRTSPStatusByCode(200) + RTSPEndLine
	strOut += HDRCSEQ + ": " + strconv.Itoa(cseq) + RTSPEndLine
	strOut += "Server: " + RTSPServerName + RTSPEndLine
	strOut += "Session: " + rtspHandler.session + ";timeout=" + strconv.Itoa(serviceConfig.TimeoutSec) + RTSPEndLine
	if track.transPort == "udp" {
		strOut += "Transport: RTP/AVP;unicast;"
		strOut += "client_port=" + strconv.Itoa(track.RTPCliPort) + "-" + strconv.Itoa(track.RTCPCliPort) + ";"
		strOut += "server_port=" + strconv.Itoa(track.RTPSvrPort) + "-" + strconv.Itoa(track.RTCPSvrPort)
		strOut += RTSPEndLine
	} else {
		//tcp
		strOut += "Transport: RTP/AVP/TCP;unicast;"
		strOut += "interleaved=" + strconv.Itoa(track.RTPChannel) + "-" + strconv.Itoa(track.RTCPChannel)
		strOut += RTSPEndLine
	}
	strOut += RTSPEndLine

	return rtspHandler.send([]byte(strOut))
}

func (rtspHandler *RTSPHandler) servePlay(lines []string) (err error) {
	errCode := 0
	defer func() {
		if err != nil {
			logger.LOGE(err.Error())
		}
		if errCode != 0 {
			rtspHandler.sendErrorReply(lines, errCode)
		}
	}()
	//状态
	if len(rtspHandler.tracks) == 0 {
		err = errors.New("play a playing stream")
		errCode = 405
		return
	}
	if rtspHandler.isPlaying || len(rtspHandler.tracks) == 0 {
		err = errors.New("play a playing stream")
		errCode = 405
		return
	}
	//cseq
	cseq := getCSeq(lines)
	//session
	cliSession := rtspHandler.getSession(lines)
	if cliSession != rtspHandler.session {
		logger.LOGD(cliSession)
		err = errors.New("session wrong")
		errCode = 454
		return
	}
	//begin time:use default zero
	//start play

	strOut := RTSPVer + " " + strconv.Itoa(200) + " " + getRTSPStatusByCode(200) + RTSPEndLine
	strOut += HDRCSEQ + ": " + strconv.Itoa(cseq) + RTSPEndLine
	strOut += "Session: " + rtspHandler.session + RTSPEndLine
	strOut += "Range: npt=0.0-" + RTSPEndLine
	strOut += "RTP-Info: "
	addCmma := false
	line0 := removeSpace(lines[0])
	subs := strings.Split(line0, " ")
	if len(subs) != 3 {
		err = errors.New("play cmd failed")
		errCode = 450
		return
	}
	url := subs[1]
	rtspHandler.mutexTracks.RLock()
	for _, v := range rtspHandler.tracks {
		if addCmma {
			strOut += ","
		}
		addCmma = true
		strOut += url + "/" + v.trackID + ";"
		strOut += "seq=" + strconv.Itoa(v.firstSeq) + ";"
		strOut += "rtptime=" + strconv.Itoa(int(v.RTPStartTime))
	}
	strOut += RTSPEndLine
	strOut += RTSPEndLine
	rtspHandler.mutexTracks.RUnlock()

	err = rtspHandler.send([]byte(strOut))
	go rtspHandler.threadPlay()
	return
}

func (rtspHandler *RTSPHandler) servePause(lines []string) (err error) {
	cseq := getCSeq(lines)
	errCode := 0
	defer func() {
		if err != nil {
			logger.LOGE(err.Error())
		}
		if errCode != 0 {
			rtspHandler.sendErrorReply(lines, errCode)
		}
	}()
	//session
	cliSession := rtspHandler.getSession(lines)
	if cliSession != rtspHandler.session {
		logger.LOGD(cliSession)
		err = errors.New("session wrong")
		errCode = 454
		return
	}
	if rtspHandler.isPlaying == false {
		err = errors.New("pause not playing")
		errCode = 455
		return
	}
	//停止播放
	rtspHandler.stopPlayThread()

	strOut := RTSPVer + " " + strconv.Itoa(200) + " " + getRTSPStatusByCode(200) + RTSPEndLine
	strOut += HDRCSEQ + ": " + strconv.Itoa(cseq) + RTSPEndLine
	strOut += "Session: " + rtspHandler.session + RTSPEndLine
	strOut += RTSPEndLine
	err = rtspHandler.send([]byte(strOut))
	return
}
