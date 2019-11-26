package backend

import (
	"container/list"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/use-go/websocket-streamserver/events/eLiveListCtrl"
	"github.com/use-go/websocket-streamserver/events/eRTMPEvent"
	"github.com/use-go/websocket-streamserver/events/eStreamerEvent"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssapi"
)

type adminStreamManageHandler struct {
	Route string
}

type streamManageRequestData struct {
	Action Action
	LoginResponseData
}

func (asmh *adminStreamManageHandler) init(data *wssapi.Msg) (err error) {
	asmh.Route = "/admin/stream/manage"
	return
}

func (asmh *adminStreamManageHandler) getRoute() (route string) {
	return asmh.Route
}

func (asmh *adminStreamManageHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.RequestURI == asmh.Route {
		asmh.handleStreamManageRequest(w, req)
	} else {
		badrequestResponse, err := BadRequest(WSSSeverError, "server error in login")
		SendResponse(badrequestResponse, err, w)
	}
}

func (asmh *adminStreamManageHandler) handleStreamManageRequest(w http.ResponseWriter, req *http.Request) {
	if !LoginHandler.isLogin {
		doManage(w, req)
		//response, err := BadRequest(WSSNotLogin, "please login")
		//SendResponse(response, err, w)
		return
	}
	requestData := getRequestData(req)
	if requestData.Action.ActionToken != LoginHandler.AuthToken {
		response, err := BadRequest(WSSUserAuthError, "Auth error")
		SendResponse(response, err, w)
		return
	}
	doManage(w, req)

}

func getRequestData(req *http.Request) streamManageRequestData {
	data := streamManageRequestData{}
	code := req.PostFormValue("action_code")
	codeInt, err := strconv.Atoi(code)
	if err != nil {
		data.Action.ActionCode = codeInt
	} else {
		data.Action.ActionCode = -1
	}
	data.Action.ActionToken = req.PostFormValue("action_token")
	return data
}

func doManage(w http.ResponseWriter, req *http.Request) {
	err := parseActionEvent(w, req)
	if err != nil {
		logger.LOGI("parseActionEvent error ", err)
		sendBadResponse(w, "action code error", WSSParamError)
	}
}

func parseActionEvent(w http.ResponseWriter, req *http.Request) error {
	actionCode := req.PostFormValue("action_code")
	if len(actionCode) == 0 {
		return errors.New("no action code")
	}
	intCode, err := strconv.Atoi(actionCode)
	var task wssapi.Task
	if err != nil {
		return errors.New("action code is error")
	}
	switch intCode {
	case WSShowAllSream:
		doShowAllStream(w)
	case WSGetLivePlayerCount:
		doGetLivePlayerCount(w, req)
	case WSEventBlackList:
		doEnableBlackList(w, req)
	case WSSetBlackList:
		doSetBlackList(w, req)
	case WSEnableWhiteList:
		doEnableWhiteList(w, req)
	case WSSetWhiteList:
		doSetWhiteList(w, req)
	case WSSetUpStreamApp:
		doSetUpStreamApp(w, req)
	case WSPullRtmpStream:
		task = &eRTMPEvent.EvePullRTMPStream{}
	case WSAddSink:
		task = &eStreamerEvent.EveAddSink{}
	case WSDelSink:
		task = &eStreamerEvent.EveDelSink{}
	case WSAddSource:
		task = &eStreamerEvent.EveAddSource{}
	case WSDelSource:
		task = &eStreamerEvent.EveDelSource{}
	case WSGetSource:
		task = &eStreamerEvent.EveGetSource{}
	default:
		return errors.New("no function")
	}

	if task == nil {

	}

	return nil
}

func doShowAllStream(w http.ResponseWriter) {
	eve := eLiveListCtrl.EveGetLiveList{}
	wssapi.HandleTask(&eve)
	list := make([]object, 0)
	for item := eve.Lives.Front(); item != nil; item = item.Next() {
		info := item.Value.(*eLiveListCtrl.LiveInfo)
		list = append(list, *info)
	}
	sendSuccessResponse(nil, list, w)
}

// get one stream player
// exp: "ws:127.0.0.1:8080/ws/live/hks"
// 		"live_name should be "live/hks""
func doGetLivePlayerCount(w http.ResponseWriter, req *http.Request) {
	liveName := req.FormValue("live_name")
	if len(liveName) <= 0 {
		sendBadResponse(w, "need live_name", WSSParamError)
		return
	}
	eve := eLiveListCtrl.EveGetLivePlayerCount{}
	eve.LiveName = liveName
	wssapi.HandleTask(&eve)
	sendSuccessResponse(eve, nil, w)
}

func doEnableBlackList(w http.ResponseWriter, r *http.Request) {
	enableBlackOrWhiteList(0, w, r)
}

func doSetBlackList(w http.ResponseWriter, req *http.Request) {
	setBlackOrWhiteList(0, w, req)
}

func doEnableWhiteList(w http.ResponseWriter, req *http.Request) {
	enableBlackOrWhiteList(1, w, req)
}

func doSetWhiteList(w http.ResponseWriter, r *http.Request) {
	setBlackOrWhiteList(1, w, r)
}

func doSetUpStreamApp(w http.ResponseWriter, r *http.Request) {
}

func doPullRtmpStream(w http.ResponseWriter, r *http.Request) {
}

func doAddSink(w http.ResponseWriter, r *http.Request) {
}

func doDelSink(w http.ResponseWriter, r *http.Request) {
}

func doAddSource(w http.ResponseWriter, r *http.Request) {
}

func doDelSource(w http.ResponseWriter, r *http.Request) {
}

func doGetSource(w http.ResponseWriter, r *http.Request) {
}

//Enable BlackList
// need form data " opcode = 1
// 					opcode 1 for enable blacklist
//		  			other for disable blacklist
//bwcode : 0 for black
//	  	 : 1 for white
func enableBlackOrWhiteList(bwcode int, w http.ResponseWriter, req *http.Request) {
	code := req.FormValue("opcode")
	eve := eLiveListCtrl.EveEnableBlackList{}

	if code == "1" {
		eve.Enable = true
	} else if code == "0" {
		eve.Enable = false
	} else {
		sendBadResponse(w, "opcode error", WSSParamError)
		return
	}

	var err error
	if bwcode == 0 {
		err = wssapi.HandleTask(&eve)
		wssapi.HandleTask(&eve)
	} else if bwcode == 1 {
		task := &eLiveListCtrl.EveEnableWhiteList{}
		task.Enable = eve.Enable
		err = wssapi.HandleTask(task)
	} else {
		sendBadResponse(w, "bwcode error", WSSSeverError)
		return
	}

	if err != nil {
		sendBadResponse(w, "error in service ", WSSParamError)
		return
	}
	sendSuccessResponse("op success", nil, w)
}

//setBlacklist
//need formdata " list=xxx|xxxx|xxxx&opcode=1
//				" | to split names
//				" opcode 1 for add 0 for del
//bwcode : 0 for black
//	  	 : 1 for white
func setBlackOrWhiteList(bwcode int, w http.ResponseWriter, req *http.Request) {
	str := req.FormValue("list")
	opcode := req.FormValue("opcode")
	if str == "" || opcode == "" {
		sendBadResponse(w, "need list data,it is look like 'list=xxx|xxx|xxx|xxx'", WSSParamError)
		return
	}

	eve := eLiveListCtrl.EveSetBlackList{}

	if opcode == "0" {
		eve.Add = false
	} else if opcode == "1" {
		eve.Add = true
	} else {
		sendBadResponse(w, "opcode error , 0 for del 1 for add", WSSParamError)
		return
	}

	banList := strings.Split(str, "|")
	eve.Names = list.New()
	for _, item := range banList {
		eve.Names.PushBack(item)
	}

	var err error
	if bwcode == 0 {
		err = wssapi.HandleTask(&eve)
	} else if bwcode == 1 {
		task := &eLiveListCtrl.EveSetWhiteList{}
		task.Add = eve.Add
		task.Names = eve.Names
		err = wssapi.HandleTask(task)
	} else {
		sendBadResponse(w, "bwcode error", WSSSeverError)
		return
	}
	if err != nil {
		sendBadResponse(w, "error in service ", WSSParamError)
		return
	}

	sendSuccessResponse("op success", nil, w)
}

func sendSuccessResponse(data object, datas []object, w http.ResponseWriter) {
	responseData := ActionResponseData{}
	responseData.Code = WSSRequestOK
	responseData.Msg = "ok"
	responseData.Data = data
	responseData.Datas = datas
	jbyte, err := json.Marshal(responseData)
	SendResponse(jbyte, err, w)
}

func sendBadResponse(w http.ResponseWriter, msg string, code int) {
	respnse, err := BadRequest(code, msg)
	SendResponse(respnse, err, w)
}
