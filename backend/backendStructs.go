package backend

//LoginResponseData struct
type LoginResponseData struct {
	Code int    `json:"code"`
	Data Data   `json:"data,omitempty"`
	Msg  string `json:"msg"`
}

// Data struct
type Data struct {
	UserData Usr    `json:"usr_data"`
	Action   Action `json:"action"`
}

//Usr struct
type Usr struct {
	Usrname string `json:"usrname"`
	Token   string `json:"token"`
}

// Action struct
type Action struct {
	ActionCode  int    `json:"action_code"`
	ActionToken string `json:"action_token"`
}
type object interface{}

// ActionResponseData struct
type ActionResponseData struct {
	Code  int      `json:"code"`
	Datas []object `json:"datas"`
	Data  object   `json:"data"`
	Msg   string   `json:"msg,omitempty"`
}

// action enum
const (
	WSShowAllSream = iota
	WSGetLivePlayerCount
	WSEventBlackList
	WSSetBlackList
	WSEnableWhiteList
	WSSetWhiteList
	WSSetUpStreamApp
	WSPullRtmpStream
	WSAddSink
	WSDelSink
	WSAddSource
	WSDelSource
	WSGetSource
)
