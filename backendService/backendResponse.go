package backendService

import (
	"encoding/json"
	"net/http"
)

//Error String
const (
	WSSRequestMethodError = 1
	WSSUserAuthError      = 101
	WSSParamError         = 102
	WSSNotLogin           = 103
	WSSRequestOK          = 200
	WSSSeverHandleError   = 501
	WSSSeverError         = 500
)

// BadRequestData struct
type BadRequestData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// SendResponse To Usr
func SendResponse(responseData []byte, err error, w http.ResponseWriter) {
	if err != nil {
		w.Write([]byte("{\"code\":500,\"msg\":\"sevr error!\"}"))
	} else {
		w.Write(responseData)
	}
}

// BadRequest func
func BadRequest(code int, msg string) ([]byte, error) {
	result := &BadRequestData{}
	result.Code = code
	result.Msg = msg
	bytes, err := json.Marshal(result)
	return bytes, err
}
