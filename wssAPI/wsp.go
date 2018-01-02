// Copyright 2018 The use-go websocket-streamserver Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wssAPI

// wsp Control Cmd Type
const (
	WSPCInit = 0
	WSPCJoin = 1
	WSPWarp  = 2
	WSPRtsp  = 3
)

//DecodeCtrlMsgType classify the msg type
func DecodeCtrlMsgType(data []byte) (ret uint32, err error) {

	return ret, nil
}
