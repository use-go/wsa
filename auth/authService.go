package auth

import (
	b64 "encoding/base64"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/use-go/websocket-streamserver/wssAPI"
)

//RTSPClient Denote the RTSP Client
type RTSPClient struct {
	socket   net.Conn
	outgoing chan []byte //out chanel
	signals  chan bool   //signals quit
	host     string      //host
	port     string      //port
	uri      string      //url
	auth     bool        //aut
	login    string
	password string   //password
	session  string   //rtsp session
	responce string   //responce string
	bauth    string   //string b auth
	track    []string //rtsp track
	cseq     int      //qury number
	videow   int
	videoh   int
}

//Write info
func (rtspClient *RTSPClient) Write(message string) bool {
	rtspClient.cseq++
	if _, e := rtspClient.socket.Write([]byte(message)); e != nil {
		return false
	}
	return true
}

//Read info
func (rtspClient *RTSPClient) Read() (bool, string) {
	buffer := make([]byte, 4096)
	if nb, err := rtspClient.socket.Read(buffer); err != nil || nb <= 0 {
		log.Println("socket read failed", err)
		return false, ""
	}
	return true, string(buffer[:nb])

}

// AuthBasic for auth
func (rtspClient *RTSPClient) AuthBasic(phase string, message string) bool {
	rtspClient.bauth = "\r\nAuthorization: Basic " + b64.StdEncoding.EncodeToString([]byte(rtspClient.login+":"+rtspClient.password))
	if !rtspClient.Write(phase + " " + rtspClient.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(rtspClient.cseq) + rtspClient.bauth + "\r\n\r\n") {
		return false
	}
	if status, message := rtspClient.Read(); status && strings.Contains(message, "200") {
		rtspClient.track = rtspClient.ParseMedia(message)
		return true
	}
	return false
}

//AuthDigest calculate Digest
func (rtspClient *RTSPClient) AuthDigest(phase string, message string) bool {
	nonce := ParseDirective(message, "nonce")
	realm := ParseDirective(message, "realm")
	hs1 := wssAPI.GetMD5Hash(rtspClient.login + ":" + realm + ":" + rtspClient.password)
	hs2 := wssAPI.GetMD5Hash(phase + ":" + rtspClient.uri)
	responce := wssAPI.GetMD5Hash(hs1 + ":" + nonce + ":" + hs2)
	dauth := "\r\n" + `Authorization: Digest username="` + rtspClient.login + `", realm="` + realm + `", nonce="` + nonce + `", uri="` + rtspClient.uri + `", response="` + responce + `"`
	if !rtspClient.Write(phase + " " + rtspClient.uri + " RTSP/1.0\r\nCSeq: " + strconv.Itoa(rtspClient.cseq) + dauth + "\r\n\r\n") {
		return false
	}
	if status, message := rtspClient.Read(); status && strings.Contains(message, "200") {
		rtspClient.track = rtspClient.ParseMedia(message)
		return true
	}
	return false
}

// AuthDigestOnly alculate Digest only
func (rtspClient *RTSPClient) AuthDigestOnly(phase string, message string) string {
	nonce := ParseDirective(message, "nonce")
	realm := ParseDirective(message, "realm")
	hs1 := wssAPI.GetMD5Hash(rtspClient.login + ":" + realm + ":" + rtspClient.password)
	hs2 := wssAPI.GetMD5Hash(phase + ":" + rtspClient.uri)
	responce := wssAPI.GetMD5Hash(hs1 + ":" + nonce + ":" + hs2)
	dauth := "\r\n" + `Authorization: Digest username="` + rtspClient.login + `", realm="` + realm + `", nonce="` + nonce + `", uri="` + rtspClient.uri + `", response="` + responce + `"`
	return dauth
}

//ParseDirective Get Directive
func ParseDirective(header, name string) string {
	index := strings.Index(header, name)
	if index == -1 {
		return ""
	}
	start := 1 + index + strings.Index(header[index:], `"`)
	end := start + strings.Index(header[start:], `"`)
	return strings.TrimSpace(header[start:end])
}

//ParseMedia Get Media
func (rtspClient *RTSPClient) ParseMedia(header string) []string {
	letters := []string{}
	mparsed := strings.Split(header, "\r\n")
	paste := ""
	for _, element := range mparsed {
		if strings.Contains(element, "a=control:") && !strings.Contains(element, "*") && strings.Contains(element, "tra") {
			paste = element[10:]
			if strings.Contains(element, "/") {
				striped := strings.Split(element, "/")
				paste = striped[len(striped)-1]
			}
			letters = append(letters, paste)
		}

		dimensionsPrefix := "a=x-dimensions:"
		if strings.HasPrefix(element, dimensionsPrefix) {
			dims := []int{}
			for _, s := range strings.Split(element[len(dimensionsPrefix):], ",") {
				v := 0
				fmt.Sscanf(s, "%d", &v)
				if v <= 0 {
					break
				}
				dims = append(dims, v)
			}
			if len(dims) == 2 {
				rtspClient.videow = dims[0]
				rtspClient.videoh = dims[1]
			}
		}
	}
	return letters
}
