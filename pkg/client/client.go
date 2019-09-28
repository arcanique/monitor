package main

import (
	"flag"
	"fmt"
	"monitor/pkg/sdk"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	defaultPort = 65435
)

func main() {
	stopCh := make(chan struct{}, 1)
	client := Init()
	if err := client.ListenLocal(); err != nil {
		fmt.Println("ListenLocal : ", err.Error())
		os.Exit(-1)
	}
	if status := client.ConnectServer(); status != sdk.OK {
		fmt.Println("ConnectServer : ", status)
		os.Exit(-1)
	}
	client.Gt.Monitor()
	client.Gt.Alarm(client)
	<-stopCh
}

func Init() *sdk.Client {
	var ip, lip string
	var port, lport int
	var pids string
	flag.StringVar(&ip,"s", "0.0.0.0", "-s <server ip>")
	flag.IntVar(&port, "p", 65530, "-p <server port>")

	flag.StringVar( &lip,"c", "0.0.0.0", "-c <local ip>")
	flag.IntVar(&lport, "cp", 65531, "-v <server port>")

	flag.StringVar(&pids, "i", "1", "-i <list of monitor program pid with Separator ,>")

	flag.Parse()

	ps := strings.Split(pids, ",")
	pidList := []uint64{}
	programs := []sdk.Process{}
	for _, p := range ps {
		if pid, err := strconv.ParseUint(p, 10, 64); err == nil {
			cm, er := sdk.GetProgramPid(pid)
			if er != nil {
				continue
			}
			program := sdk.Process{
				Pid: pid,
				Cmd: cm,
			}
			programs = append(programs, program)
			pidList = append(pidList, pid)
		}
	}

	laddr := &net.UDPAddr{
		IP:   net.ParseIP(lip),
		Port: lport,
	}

	gt := &sdk.Gather{
		AlarmCh: make(chan *sdk.Info, 1024),
		Program: programs,
		OsInfo:  &sdk.OsInf{},
	}

	gt.OsInformation()

	return &sdk.Client{
		Gt:         gt,
		Run:        make(chan struct{},2),
		Laddr:      laddr,
		ClientIP:   laddr.IP,
		ClientPort: laddr.Port,
		ServerIP:   net.ParseIP(ip),
		ServerPort: port,
		Pids:       pidList,
	}
}
