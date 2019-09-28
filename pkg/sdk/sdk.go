package sdk

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

type Status uint8

const (
	OK    = Status('y')
	NotOK = Status('n')

	BufSzie = 1400
)

var HostID int

type Server struct {
	Cycle int
	locker sync.RWMutex
	Laddr *net.UDPAddr
	DataCh chan *Message
	AgentList []net.UDPAddr
	// id for info
	HostInfoList map[string]OsInf
	// get id from agent ip
	Host    map[string]string
	MsgId string
	Data Statistics
}

type Client struct {
	ID         string
	Run        chan struct{}
	Gt         *Gather
	Laddr      *net.UDPAddr
	ClientIP   net.IP
	ClientPort int
	ServerIP   net.IP
	ServerPort int
	Pids       []uint64
	RConn      *net.UDPConn
}

func (c *Client) ConnectServer() Status {
	rAddr := &net.UDPAddr{
		IP:c.ServerIP,
		Port:c.ServerPort,
	}
	c.Gt.OsInformation()

	rconn,err := net.DialUDP("udp",nil,rAddr)
	if err != nil {
		return NotOK
	}

	data := c.Gt.OsInfo
	msg := BuildMsg(REGISTER,"",c.ID,*c.Laddr ,data)
	if msg == nil {
		return NotOK
	}
	buf,err := json.Marshal(*msg)
	if err != nil {
		fmt.Println("data Marshal error : ",err.Error())
		return NotOK
	}
	_,err = rconn.Write(buf)

	if err != nil {
		fmt.Println("data Marshal error : ",err.Error())
		return NotOK
	}

	fmt.Println("sent to server for register")

	<- c.Run

	fmt.Println("Register Success,Agent running")
	c.RConn = rconn

	//go c.sendHeartBet()

	return OK
}

//func (c *Client)sendHeartBet() {
//	tic := time.Tick(time.Minute)
//	for {
//
//	}
//
//}

func (c *Client) ListenLocal() error {
	conn, err := net.ListenUDP("udp", c.Laddr)
	if err != nil {
		return err
	}
	go c.Clienter(conn)
	return nil
}

func (c *Client)Clienter(con *net.UDPConn) {
	buf := make([]byte, BufSzie)
	for {
		n, err := con.Read(buf)
		if err != nil {
			continue
		}
		data := buf[:n]
		msg,err := ProcessMsg(data)
		if err != nil {
			continue
		}
		go c.ClientProcessMsg(msg)
	}
}


func (s *Server)ListenAndServe() error {
	conn, err := net.ListenUDP("udp",s.Laddr)
	if err != nil {
		return err
	}
	go s.Serve(conn)
	return nil
}

func (s *Server)Serve(conn *net.UDPConn) {
	buf := make([]byte,BufSzie)
	for {
		n,err := conn.Read(buf)
		//fmt.Println("read")
		if err != nil {
			continue
		}
		if n <= 0 {
			continue
		}
		tmp := buf[:n]
		msg,err := ProcessMsg(tmp)
		if err != nil {
			continue
		}
		s.DataCh <- msg
		msg = msg
	}
}

func (s *Server)ServerProcessMsg(){
	HostID = 100
	for {
		msg := <- s.DataCh
		switch msg.GetHeader().MsgType {
		case REGISTER:
			agentAddr := msg.GetSource()
			s.locker.Lock()
			s.AgentList = append(s.AgentList,agentAddr)
			s.locker.Unlock()
			id := s.Host[agentAddr.IP.String()]
			if id == "" {
				id = strconv.Itoa(HostID)
				s.Host[agentAddr.IP.String()] = id
				HostID++
			}
			var value OsInf
			data,err := json.Marshal(msg.Content)
			if err == nil {
				err = json.Unmarshal(data,&value)
			}
			if err == nil  {
				s.HostInfoList[id] = value
				mg := BuildMsg(REGISTERRESP,"","",net.UDPAddr{},id)
				Send2Agent(&agentAddr,mg)
			}
		case HOSTINFO:
			if msg.Header.MsgID != s.MsgId {
				fmt.Println("wrong msg ID")
			}
			var value InfoList
			data,err := json.Marshal(msg.Content)
			if err == nil {
				err = json.Unmarshal(data,&value)
			}
			if err == nil  {
				kd := s.Data.Data[msg.Header.HostID]
				kd.HostList = value
			}
			//fmt.Printf("%+v",msg)
		case PROCESSINFO:
			if msg.Header.MsgID != s.MsgId {
				fmt.Println("wrong msg ID")
			}
			var value InfoList
			data,err := json.Marshal(msg.Content)
			if err == nil {
				err = json.Unmarshal(data,&value)
			}
			if err == nil {
				kd := s.Data.Data[msg.Header.HostID]
				kd.ProInfoList = append(kd.ProInfoList,value)
			}
			//fmt.Printf("%+v",msg)
		case HARTBET:
		}
	}

}

func (c *Client)ClientProcessMsg(msg *Message) {
	switch msg.GetHeader().MsgType {
	case QUERYDATA:
		//fmt.Println("agent get a query")
		md := c.Gt.Md
		osmsg := BuildMsg(HOSTINFO,msg.Header.MsgID,c.ID,net.UDPAddr{},md.OsInfoList)
		data,err := json.Marshal(osmsg)
		if err != nil {
			data = []byte("ERROR")
		}
		c.RConn.Write(data)
		for _,p := range c.Gt.Program {
			hostmsg := BuildMsg(PROCESSINFO,msg.Header.MsgID,c.ID,net.UDPAddr{},p.Inf)
			data,err = json.Marshal(hostmsg)
			if err != nil {
				data = []byte("ERROR")
			}
			c.RConn.Write(data)
		}

	case REGISTERRESP:
		id,ok := msg.GetContent().(string)
		if ok {
			c.ID = id
			c.Run <- struct{}{}
		}
	}
}

//func SendQueryData2Server(conn *net.UDPConn,msg *Message){
//	conn,err := net.DialUDP("udp",nil,agnetAddr)
//	if err != nil {
//		return
//	}
//
//	data, err:= json.Marshal(msg)
//	if err != nil {
//		return
//	}
//
//	conn.Write(data)
//}

func Send2Agent(agnetAddr *net.UDPAddr,msg *Message) {
	conn,err := net.DialUDP("udp",nil,agnetAddr)
	if err != nil {
		return
	}

	data, err:= json.Marshal(msg)
	if err != nil {
		return
	}

	_,err = conn.Write(data)
	if err != nil {
		fmt.Println(err.Error())
	}

}

func (s *Server)Getdata() {
	tic := time.Tick(time.Duration(s.Cycle) * time.Second)
	for {
		<- tic
		//fmt.Println("get data from Agent")
		s.Data = Statistics{
			Data:make(map[string]*KDdata),
		}
		msgID := strconv.FormatInt(time.Now().UnixNano(),10)
		mg := BuildMsg(QUERYDATA,msgID,"",net.UDPAddr{},nil)
		for _, agent := range s.AgentList {
			fmt.Println("get data from Agent")
			id := s.Host[agent.IP.String()]
			s.Data.Data[id] = &KDdata{}
			Send2Agent(&agent,mg)
		}
		s.MsgId = msgID
		go s.CalculateAvg()
	}

}

func (s *Server)CalculateAvg(){
	time.Sleep(time.Second * 10)
	for _, agent := range s.AgentList {
		id := s.Host[agent.IP.String()]
		kd := s.Data.Data[id]

		hNum := 0
		hCpu := 0.0
		hMem := uint64(0)
		hShr := 0.0
		//hMse := 0.0
		var hInf Info
		for _,inf := range kd.HostList {
			//Pid    uint64
			//Cpu    float64
			//Mem    uint64
			//Shr    float64
			//RunTim string
			//Cmd    string
			//Err    error
			if inf.Err != nil {
				continue
			}
			hCpu = hCpu + inf.Cpu
			hMem = hMem + inf.Mem
			hShr = hShr + inf.Shr
			hNum++
			hInf = inf
		}
		if hNum == 0{
			hNum = 1
		}
		kd.HostInfo = Info{
			Pid:hInf.Pid,
			Cpu:hCpu/float64(hNum),
			Mem:hMem/uint64(hNum),
			Shr:hShr/float64(hNum),
			Cmd:hInf.Cmd,
			RunTim:hInf.RunTim,
		}
		for _,pro := range kd.ProInfoList {

			pNum := 0
			pCpu := 0.0
			pMem := uint64(0)
			pShr := 0.0
			var pInf Info
			//pMse := 0.0
			for _,inf := range pro {
				if inf.Err != nil {
					continue
				}
				pCpu = pCpu + inf.Cpu
				pMem = pMem + inf.Mem
				pShr = pShr + inf.Shr
				pNum++
				pInf = inf
			}
			if pNum == 0{
				pNum = 1
			}
			info := Info{
				Pid:pInf.Pid,
				Cpu:pCpu/float64(pNum),
				Mem:pMem/uint64(pNum),
				Shr:pShr/float64(pNum),
				Cmd:pInf.Cmd,
				RunTim:pInf.RunTim,
			}
			kd.ProInfo = append(kd.ProInfo,info)
		}
	}
	s.Show(s.MsgId,time.Now().String())

}