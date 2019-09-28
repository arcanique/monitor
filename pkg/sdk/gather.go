package sdk

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	spaceCh   = " "
	emptyCh   = ""
	LineBreak = "\n"
	topCmd    = "top"
	baseArgs  = "-b -n 1"
	PID       = "PID"
	USER      = "USER"
	RES       = "RES"
	SHR       = "SHR"
	CPU       = "%CPU"
	TIME      = "TIME+"
	CMD       = "COMMAND"
)

type Gather struct {
	AlarmCh chan *Info
	Program []Process
	OsInfo  *OsInf
	OsList  []Info
	Md MonitorData
}

type InfoList []Info

type MonitorData struct {
	OsInfoList InfoList
	ProcessInfoList InfoList
}

type Process struct {
	Pid uint64
	Cmd string
	Inf InfoList
}

type Info struct {
	Pid    uint64
	Cpu    float64
	Mem    uint64
	Shr    float64
	RunTim string
	Cmd    string
	Err    error
}

type OsInf struct {
	CpuNum uint8 `json:"cpu_num"`
	Mem    uint64 `json:"mem"`
	HostNmae   string `json:"host_nmae"`
	CpuFre string `json:"cpu_fre"`
}

func GetOsCpuMem() *Info {
	args := strings.Split(baseArgs+" -p 1", spaceCh)
	cmd := exec.Command(topCmd, args...)
	out, err := cmd.Output()
	if err != nil {
		return &Info{
			Pid: 0,
			Cmd: "host os",
			Err: fmt.Errorf("[Warning] get process pid cpu and mem error : " + err.Error()),
		}
	}

	data := strings.Split(string(out), LineBreak)
	num := 0
	mem := uint64(0)
	cpu := 0.0
	for _, line := range data {
		if strings.Contains(line, "Cpu") {
			m := strings.Index(line, "id")
			n := strings.Index(line, "ni")
			u := strings.Trim(line[n+3:m], " ")
			c, err := strconv.ParseFloat(u, 10)
			if err != nil {
				return &Info{
					Pid: 0,
					Cmd: "host os",
					Err: fmt.Errorf("[Warning] get process pid cpu and mem error : " + err.Error()),
				}
			}
			cpu = c
			num++
		} else if strings.Contains(line, "Mem") {
			m := strings.Index(line, "used")
			n := strings.Index(line, "free")
			u := strings.Trim(line[n+5:m], " ")
			p, err := strconv.ParseUint(u, 10, 64)
			if err != nil {
				return &Info{
					Pid: 0,
					Cmd: "host os",
					Err: fmt.Errorf("[Warning] get process pid cpu and mem error : " + err.Error()),
				}
			}
			mem = p
			num++
		}
		if num == 2 {
			break
		}
	}
	if err != nil {
		return &Info{
			Pid: 0,
			Cmd: "host os",
			Err: fmt.Errorf("[Warning] get process pid cpu and mem error : " + err.Error()),
		}
	}
	return &Info{
		Pid:    0,
		Cpu:    math.Round((100-cpu)*100) / 100,
		Mem:    mem,
		Cmd:    "host os",
		RunTim: "",
		Err:    nil,
	}
}

func GetProcessCpuMem(pid uint64, cmd string) *Info {
	args := strings.Split(baseArgs+" -p "+strconv.FormatUint(pid, 10), spaceCh)
	c := exec.Command(topCmd, args...)
	out, err := c.Output()
	if err != nil {
		return &Info{
			Pid: pid,
			Cmd: cmd,
			Err: fmt.Errorf("[Warning] get process pid cpu and mem error : " + err.Error()),
		}
	}

	data := strings.Split(string(out), LineBreak)
	mem := uint64(0)
	runTime := ""
	cm := ""
	cpu := 0.0
	shr := 0.0
	idxList := []string{}
	dataList := []string{}
	for _, line := range data {
		if strings.Contains(line, "VIRT") {
			f := strings.Split(line, " ")
			for _, ele := range f {
				if ele == "" {
					continue
				}
				idxList = append(idxList, ele)
			}
		}
		if strings.Contains(line, cmd) {
			f := strings.Split(line, " ")
			for _, ele := range f {
				if ele == "" {
					continue
				}
				dataList = append(dataList, ele)
			}
		}
	}
	if len(dataList) == 0 {
		return &Info{
			Pid:    pid,
			Cpu:    cpu,
			Mem:    mem,
			Shr:    shr,
			RunTim: runTime,
			Cmd:    cm,
			Err:    err,
		}
	}
	for idx, ele := range idxList {
		switch ele {
		case RES:
			mem, err = strconv.ParseUint(dataList[idx], 10, 64)
		case SHR:
			shr, err = strconv.ParseFloat(dataList[idx], 10)
		case CPU:
			cpu, err = strconv.ParseFloat(dataList[idx], 10)
		case TIME:
			runTime = dataList[idx]
		case CMD:
			cm = dataList[idx]
		default:
		}
	}
	return &Info{
		Pid:    pid,
		Cpu:    cpu,
		Mem:    mem,
		Shr:    shr,
		RunTim: runTime,
		Cmd:    cm,
		Err:    err,
	}
}

func (in *Info) getStringFromInfo() []byte {
	b, err := json.Marshal(in)
	if err != nil {
		return []byte("")
	}
	return b
}

func (g *Gather) Monitor() {
	g.OsInformation()
	fmt.Printf("%+v\n", g.OsInfo)
	MemThreshold := uint64(float64(g.OsInfo.Mem) * 0.85)
	go func() {
		timer := time.Tick(time.Second)
		for {
			<-timer
			osUsageRate := GetOsCpuMem()
			g.Md.InsertData("os",*osUsageRate)
			//fmt.Printf("[GetOsCpuMem] : %+v\n", osUsageRate)
			if osUsageRate.Cpu > 85 || osUsageRate.Mem > MemThreshold {
				g.AlarmCh <- osUsageRate
			}
			for idx, p := range g.Program {
				psUsageRate := GetProcessCpuMem(p.Pid, p.Cmd)
				//a ,_ := json.Marshal(psUsageRate)
				if len(g.Program[idx].Inf) == 10 {
					g.Program[idx].Inf = g.Program[idx].Inf[1:]
				}
				g.Program[idx].Inf = append(g.Program[idx].Inf,*psUsageRate)
				//g.Md.InsertData("process",*psUsageRate)
				//fmt.Printf("[GetProcessCpuMem] : %+v\n", a)
			}
		}
	}()
}

func (g *Gather)Alarm(c *Client){
	for {
		value := <- g.AlarmCh
		msg := BuildMsg(ALARMDATA,"",c.ID,net.UDPAddr{},value)
		data,err := json.Marshal(msg)
		if err != nil {
			continue
		}
		c.RConn.Write(data)
	}
}

func (g *Gather) OsInformation() error {
	cmd := exec.Command("uname", "-n")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Get OsInformation Arch %v", err)
	}
	g.OsInfo.HostNmae = string(out)

	memcmd := exec.Command("cat", "/proc/meminfo")
	out, err = memcmd.Output()
	if err != nil {
		return fmt.Errorf("Get OsInformation mem %v", err)
	}
	data := strings.Split(string(out), "\n")
	for _, line := range data {
		if strings.Contains(line, "MemTotal") {
			tmpStr := strings.Split(line, " ")
			for _, memoryStr := range tmpStr {
				g.OsInfo.Mem, err = strconv.ParseUint(memoryStr, 10, 64)
				if err != nil {
					continue
				}
				break
			}
			if err != nil {
				return fmt.Errorf("Get OsInformation mem %v", err)
			}
			break
		}
	}

	cpucmd := exec.Command("cat", "/proc/cpuinfo")
	out, err = cpucmd.Output()
	if err != nil {
		return fmt.Errorf("Get OsInformation cpuFre %v", err)
	}
	data = strings.Split(string(out), "\n")
	for _, line := range data {
		if strings.Contains(line, "cpu MHz") {
			g.OsInfo.CpuFre = line
			break
		}
	}

	g.OsInfo.CpuNum = uint8(runtime.NumCPU())

	return nil
}

func GetProgramPid(pid uint64) (string, error) {
	pidStr := strconv.FormatUint(pid, 10)
	cmd := exec.Command("ps", "-p", pidStr)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Get OsInformation Arch %v", err)
	}
	data := strings.Split(string(out), "\n")
	command := ""
	for _, line := range data {
		if strings.Contains(line, pidStr) {
			tmp := strings.Split(line, " ")
			command = tmp[len(tmp)-1]
			break
		}

	}
	return command, nil
}

func (md *MonitorData)InsertData(T string,data Info) {
	switch T {
	case "os":
		if len(md.OsInfoList) == 10 {
			md.OsInfoList = md.OsInfoList[1:]
		}
		md.OsInfoList = append(md.OsInfoList,data)
	default:
	}
}