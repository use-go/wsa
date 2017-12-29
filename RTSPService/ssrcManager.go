package RTSPService

import (
	"math/rand"
	"sync"

	"github.com/use-go/websocket-streamserver/wssAPI"
)

type ssrcManager struct {
	set   *wssAPI.Set //mark for a Stremaer
	mutex sync.RWMutex
}

func newSSRCManager() (manager *ssrcManager) {
	manager = &ssrcManager{}
	manager.set = wssAPI.NewSet()
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
