package sdk

import (
	"fmt"
	"strings"
	"sync"
)

type Statistics struct {
	loker sync.RWMutex
	Data map[string]*KDdata
}

type KDdata struct {
	HostList InfoList
	HostInfo Info
	ProInfoList []InfoList
	ProInfo   []Info
}

func ShowTitle(msID, stamp string) {
	fmt.Println("=====================================================================================================")
	fmt.Printf(">>>> MsgID <%s>, TIME <%s> \n",msID,stamp)
	fmt.Println("=====================================================================================================")
}


func ShowHost(inf *OsInf,info *Info) {
	if info.Err == nil {
		fmt.Printf(">>>>>>>> Host <%s> Info Total : cpu-num %d, cpu-freq %sMHz MEM %dKB\n", strings.ReplaceAll(inf.HostNmae,"\n",""), inf.CpuNum, inf.CpuFre, inf.Mem)
		fmt.Printf(">>>>>>>> Host Info Used : CPU %0.2f%s, MEM %d KB \n", info.Cpu,"%", info.Mem)
	}else {
		fmt.Printf(">>>>>>>>>>>> Host Info Used  Error %s -->>>>  ",info.Err.Error())
	}
}

func showProcess(info *Info) {
	if info.Err == nil {
		fmt.Printf(">>>>>>>>>>>> Prgram :%s Used -->>>> ",info.Cmd)
		fmt.Printf("CPU: %.2f%s,Mem ：%d ,SHR :%.2f runTime: %s\n",info.Cpu,"%",info.Mem,info.Shr,info.RunTim)
	} else {
		fmt.Printf(">>>>>>>>>>>> Prgram :%s Error %s -->>>>  ",info.Cmd,info.Err.Error())
	}
}

func (s *Server)Show(msID, stamp string){
	ShowTitle(msID, stamp )
	for id,inf := range s.HostInfoList {
		kd := s.Data.Data[id]
		ShowHost(&inf,&kd.HostInfo)
		for _,info := range kd.ProInfo {
			showProcess(&info)
		}
	}
	fmt.Println("")
	fmt.Println("")
}