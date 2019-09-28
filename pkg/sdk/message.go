package sdk

import (
	"encoding/json"
	"fmt"
	"net"
)

type MessageType uint8

const (
	REGISTER = MessageType(0)
	REGISTERRESP = MessageType(1)
	HOSTINFO = MessageType(2)
	PROCESSINFO = MessageType(3)
	QUERYDATA = MessageType(4)
	ALARMDATA = MessageType(5)
	HARTBET = MessageType(6)

)

type Message struct {
	Header Header
	Router Router
	Content interface{}
}

type Header struct {
	MsgID  string
	HostID string
	MsgType MessageType
 }


type Router struct {
	Source net.UDPAddr
}


//type RegInfo struct {
//	Host string `json:"host"`
//	Port int `json:"port"`
//	CpuNum int  `json:"cpu_num"`
//	Mem    uint64 `json:"mem"`
//}


func (m *Message)GetHeader() *Header{
	return &m.Header
}

func (m *Message)GetSource() net.UDPAddr{
	return m.Router.Source
}

func (m *Message)GetContent() interface{}{
	return m.Content
}

func BuildMsg(msgType MessageType,MsgId,hostId string,msgSrc net.UDPAddr,c interface{})*Message{
	h := Header{
		MsgID:MsgId,
		HostID:hostId,
		MsgType:msgType,
	}
	r := Router{
		Source:msgSrc,
	}
	return &Message{
		Header:h,
		Router:r,
		Content:c,
	}
}

func ProcessMsg(data []byte) (*Message,error){
	var msg Message
	err := json.Unmarshal(data,&msg)
	if err != nil {
		return nil,err
	}
	return &msg,err
}

func ProcessByte(msg *Message) ([]byte,error){
	data ,err := json.Marshal(msg)
	if err != nil || len(data) == 0 || len(data) > BufSzie {
		return nil,fmt.Errorf("get data byte from message error")
	}
	return data,nil
}