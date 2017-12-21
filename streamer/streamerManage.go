package streamer

import (
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/use-go/websocket-streamserver/events/eLiveListCtrl"
	"github.com/use-go/websocket-streamserver/events/eRTMPEvent"
	"github.com/use-go/websocket-streamserver/logger"
	"github.com/use-go/websocket-streamserver/wssAPI"
)

func enableBlackList(enable bool) (err error) {

	service.mutexBlackList.Lock()
	defer service.mutexBlackList.Unlock()
	service.blackOn = enable
	return
}

func addBlackList(blackList *list.List) (err error) {

	service.mutexBlackList.Lock()
	defer service.mutexBlackList.Unlock()
	errs := ""
	for e := blackList.Front(); e != nil; e = e.Next() {
		name, ok := e.Value.(string)
		if false == ok {
			logger.LOGE("add blackList itor not string")
			errs += " add blackList itor not string \n"
			continue
		}
		service.blacks[name] = name
		if service.blackOn {
			service.delSource(name, 0xffffffff)
		}
	}
	if len(errs) > 0 {
		err = errors.New(errs)
	}
	return
}

func delBlackList(blackList *list.List) (err error) {
	service.mutexBlackList.Lock()
	defer service.mutexBlackList.Unlock()
	errs := ""
	for e := blackList.Front(); e != nil; e = e.Next() {
		name, ok := e.Value.(string)
		if ok == false {
			logger.LOGE("del blackList itor not string")
			errs += " del blackList itor not string \n"
			continue
		}
		delete(service.blacks, name)
	}
	if len(errs) > 0 {
		err = errors.New(errs)
	}
	return
}

func enableWhiteList(enable bool) (err error) {

	service.mutexWhiteList.Lock()
	defer service.mutexWhiteList.Unlock()
	service.whiteOn = enable
	return
}

func addWhiteList(whiteList *list.List) (err error) {

	service.mutexWhiteList.Lock()
	defer service.mutexWhiteList.Unlock()
	errs := ""
	for e := whiteList.Front(); e != nil; e = e.Next() {
		name, ok := e.Value.(string)
		if ok == false {
			logger.LOGE("add whiteList itor not string")
			errs += " add blackList itor not string \n"
			continue
		}
		service.whites[name] = name
	}
	if len(errs) > 0 {
		err = errors.New(errs)
	}
	return
}

func delWhiteList(whiteList *list.List) (err error) {

	service.mutexWhiteList.Lock()
	defer service.mutexWhiteList.Unlock()
	errs := ""
	for e := whiteList.Front(); e != nil; e = e.Next() {
		name, ok := e.Value.(string)
		if ok == false {
			logger.LOGE("del whiteList itor not string")
			errs += " del blackList itor not string \n"
			continue
		}
		delete(service.whites, name)
		if service.whiteOn {
			service.delSource(name, 0xffffffff)
		}
	}
	if len(errs) > 0 {
		err = errors.New(errs)
	}
	return
}

func getLiveCount() (count int, err error) {
	service.mutexSources.RLock()
	defer service.mutexSources.RUnlock()
	count = len(service.sources)
	return
}

func getLiveList() (liveList *list.List, err error) {
	service.mutexSources.RLock()
	defer service.mutexSources.RUnlock()
	liveList = list.New()
	for k, v := range service.sources {
		info := &eLiveListCtrl.LiveInfo{}
		info.StreamName = k
		v.mutexSink.RLock()
		info.PlayerCount = len(v.sinks)
		info.IP = v.addr.String()
		v.mutexSink.RUnlock()
		liveList.PushBack(info)
	}
	return
}

func getPlayerCount(name string) (count int, err error) {

	service.mutexSources.RLock()
	defer service.mutexSources.RUnlock()
	src, exist := service.sources[name]
	if exist == false {
		count = 0
	} else {
		count = len(src.sinks)
	}

	return
}

func (streamerService *StreamerService) checkStreamAddAble(appStreamname string) bool {
	tmp := strings.Split(appStreamname, "/")
	var name string
	if len(tmp) > 1 {
		name = tmp[1]
	} else {
		name = appStreamname
	}
	streamerService.mutexBlackList.RLock()
	defer streamerService.mutexBlackList.RUnlock()
	if streamerService.blackOn {
		for k, _ := range streamerService.blacks {
			if name == k {
				return false
			}
		}
	}
	streamerService.mutexWhiteList.RLock()
	defer streamerService.mutexWhiteList.RUnlock()
	if streamerService.whiteOn {
		for k, _ := range streamerService.whites {
			if name == k {
				return true
			}
		}
		return false
	}
	return true
}

func (streamerService *StreamerService) addUpstream(app *eLiveListCtrl.EveSetUpStreamApp) (err error) {
	streamerService.mutexUpStream.Lock()
	defer streamerService.mutexUpStream.Unlock()
	exist := false
	if app.Weight < 1 {
		app.Weight = 1
	}
	logger.LOGD(app.ID)
	for e := streamerService.upApps.Front(); e != nil; e = e.Next() {
		v := e.Value.(*eLiveListCtrl.EveSetUpStreamApp)
		if v.Equal(app) {
			exist = true
			break
		}
	}

	if exist {
		return errors.New("add up app:" + app.ID + " existed")
	}
	streamerService.upApps.PushBack(app.Copy())

	return
}

func (streamerService *StreamerService) delUpstream(app *eLiveListCtrl.EveSetUpStreamApp) (err error) {
	streamerService.mutexUpStream.Lock()
	defer streamerService.mutexUpStream.Unlock()
	for e := streamerService.upApps.Front(); e != nil; e = e.Next() {
		v := e.Value.(*eLiveListCtrl.EveSetUpStreamApp)
		if v.Equal(app) {
			streamerService.upApps.Remove(e)
			return
		}
	}
	return errors.New("del up app: " + app.ID + " not existed")
}

func (streamerService *StreamerService) SetParent(parent wssAPI.MsgHandler) {
	streamerService.parent = parent
}

// badIni IF err happened  during initialization we call streamerService
func (streamerService *StreamerService) badIni() {
	logger.LOGW("some bad init here!!!")
	//taskAddUp := eLiveListCtrl.NewSetUpStreamApp(true, "live", "rtmp", "live.hkstv.hk.lxdns.com", 1935)
	//	taskAddUp := eLiveListCtrl.NewSetUpStreamApp(true, "live", "rtmp", "127.0.0.1", 1935)
	//	streamerService.HandleTask(taskAddUp)
}

func (streamerService *StreamerService) InitUpstream(up eLiveListCtrl.EveSetUpStreamApp) {
	up.Add = true
	streamerService.HandleTask(&up)
}

func (streamerService *StreamerService) getUpAddrAuto() (addr *eLiveListCtrl.EveSetUpStreamApp) {
	streamerService.mutexUpStream.RLock()
	defer streamerService.mutexUpStream.RUnlock()
	size := streamerService.upApps.Len()
	if size > 0 {
		totalWeight := 0
		for e := streamerService.upApps.Front(); e != nil; e = e.Next() {
			v := e.Value.(*eLiveListCtrl.EveSetUpStreamApp)
			totalWeight += v.Weight
		}
		if totalWeight == 0 {
			logger.LOGF(totalWeight)
			return
		}
		idx := rand.Intn(totalWeight) + 1
		cur := 0
		for e := streamerService.upApps.Front(); e != nil; e = e.Next() {
			v := e.Value.(*eLiveListCtrl.EveSetUpStreamApp)
			cur += v.Weight
			if cur >= idx {
				return v
			}
		}
	}
	return
}

func (streamerService *StreamerService) getUpAddrCopy() (addrs *list.List) {
	streamerService.mutexUpStream.RLock()
	defer streamerService.mutexUpStream.RUnlock()
	addrs = list.New()
	for e := streamerService.upApps.Front(); e != nil; e = e.Next() {
		addrs.PushBack(e.Value.(*eLiveListCtrl.EveSetUpStreamApp))
	}
	return
}

func (streamerService *StreamerService) pullStreamExec(app, streamName string, addr *eLiveListCtrl.EveSetUpStreamApp) (src wssAPI.MsgHandler, ok bool) {
	chRet := make(chan wssAPI.MsgHandler) //这个ch由任务执行者来关闭
	protocol := strings.ToLower(addr.Protocol)
	switch protocol {
	case "rtmp":
		task := &eRTMPEvent.EvePullRTMPStream{}
		task.App = addr.App
		if strings.Contains(app, "/") {
			tmp := strings.Split(app, "/")
			task.Instance = strings.TrimPrefix(app, tmp[0])
			task.Instance = strings.TrimPrefix(task.Instance, "/")
			task.App += "/" + task.Instance
		} else {
			task.Instance = addr.Instance
		}
		task.Address = addr.Addr
		task.Port = addr.Port
		task.Protocol = addr.Protocol
		task.StreamName = streamName
		task.Src = chRet
		task.SourceName = app + "/" + streamName
		err := wssAPI.HandleTask(task)
		if err != nil {
			logger.LOGE(err.Error())
			return
		}
	default:
		close(chRet)
		logger.LOGE(fmt.Sprintf("%s not support now...", addr.Protocol))
		return
	}
	//wait for success or timeout
	select {
	case src, ok = <-chRet:
		if ok {
			logger.LOGD("pull up stream true")
		} else {
			logger.LOGD("pull up stream false")
		}
		return
	case <-time.After(time.Duration(serviceConfig.UpstreamTimeoutSec) * time.Second):
		logger.LOGD("pull up stream timeout")
		return
	}
	return
}

func (streamerService *StreamerService) pullStream(app, streamName, sinkId string, sinker wssAPI.MsgHandler) {
	//按权重随机一个
	addr := streamerService.getUpAddrAuto()
	if nil == addr {
		logger.LOGE("upstream not found")
		return
	}
	src, ok := streamerService.pullStreamExec(app, streamName, addr)
	defer func() {
		if true == ok && wssAPI.InterfaceValid(src) {
			source, ok := src.(*streamSource)
			if true == ok {
				logger.LOGD("add sink")
				msg := &wssAPI.Msg{}
				msg.Type = wssAPI.MsgGetSourceNotify
				sinker.ProcessMessage(msg)
				source.AddSink(sinkId, sinker)
			} else {
				logger.LOGE("add sink failed", source, ok)
				msg := &wssAPI.Msg{Type: wssAPI.MsgGetSourceFailed}
				sinker.ProcessMessage(msg)
			}
		} else {
			logger.LOGE("bad add", ok, src)
			logger.LOGD(reflect.TypeOf(src))
			msg := &wssAPI.Msg{Type: wssAPI.MsgGetSourceFailed}
			sinker.ProcessMessage(msg)
		}

	}()
	if true == ok && wssAPI.InterfaceValid(src) {
		return
	}
	//按顺序进行
	addrs := streamerService.getUpAddrCopy()
	for e := addrs.Front(); e != nil; e = e.Next() {
		var addr *eLiveListCtrl.EveSetUpStreamApp
		addr, ok = e.Value.(*eLiveListCtrl.EveSetUpStreamApp)
		if false == ok || nil == addr {
			logger.LOGE("invalid addr")
			continue
		}
		src, ok = streamerService.pullStreamExec(app, streamName, addr)
		if true == ok && wssAPI.InterfaceValid(src) {
			return
		}
	}
}
