package main

import (
	"backdoor/util"
	"sync"
)

var ginfomgr InfoMgr

type InfoMgr struct {
	mutex sync.Mutex
	infos map[string]*util.Info
}

func (mgr *InfoMgr) Add(pinfo *util.Info) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	if mgr.infos == nil {
		mgr.infos = make(map[string]*util.Info)
	}
	delete(mgr.infos, pinfo.UUID)
	mgr.infos[pinfo.UUID] = pinfo
}

func (mgr *InfoMgr) All() []*util.Info {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	var results []*util.Info
	for _, v := range mgr.infos {
		results = append(results, v)
	}
	return results
}

func (mgr *InfoMgr) Exist(key string) bool {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()
	_, ok := mgr.infos[key]
	return ok
}
