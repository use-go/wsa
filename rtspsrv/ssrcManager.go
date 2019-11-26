package rtspsrv

import (
	"math/rand"
	"sync"

	"github.com/use-go/websocket-streamserver/wssapi"
)

type ssrcManager struct {
	set   *wssapi.Set //mark for a Stremaer
	mutex sync.RWMutex
}

func newSSRCManager() (manager *ssrcManager) {
	manager = &ssrcManager{}
	manager.set = wssapi.NewSet()
	return
}

//Get a Globol Unique SSRC for Streamer
func (ssrcmanager *ssrcManager) NewSSRC() (id uint32) {
	ssrcmanager.mutex.Lock()
	defer ssrcmanager.mutex.Unlock()
	for {
		id = rand.Uint32()
		if false == ssrcmanager.set.Has(id) {
			ssrcmanager.set.Add(id)
			return
		}
	}
}
