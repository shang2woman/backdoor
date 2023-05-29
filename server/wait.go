package main

import (
	"backdoor/util"
	"net"
	"sync"
)

var gwait WaitSession

type WaitSession struct {
	mutex    sync.Mutex
	uuid     string
	pdecoder *util.Decoder
	conn     net.Conn
	notify   chan int
}

func (ws *WaitSession) Wait(uuid string) (net.Conn, *util.Decoder) {
	ws.mutex.Lock()
	ws.uuid = uuid
	if ws.notify == nil {
		ws.notify = make(chan int, 1)
	}
	ws.mutex.Unlock()
	<-ws.notify
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	tmpDecoder := ws.pdecoder
	tmpConn := ws.conn
	ws.pdecoder = nil
	ws.conn = nil
	return tmpConn, tmpDecoder
}

func (ws *WaitSession) IsNeed(uuid string, conn net.Conn, pdecoder *util.Decoder) bool {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if len(ws.uuid) == 0 {
		return false
	}
	if ws.uuid != uuid {
		return false
	}
	ws.uuid = ""
	ws.pdecoder = pdecoder
	ws.conn = conn
	ws.notify <- 1
	return true
}
