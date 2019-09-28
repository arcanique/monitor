package main

import (
	"flag"
	"monitor/pkg/sdk"
	"net"
)

func main() {
	stopCh := make(chan struct{})
	server := Init()
	go server.ServerProcessMsg()
	go server.Getdata()
	<- stopCh
}



func Init() *sdk.Server {
	var ip string
	var port int

	var cycle int

	flag.StringVar(&ip, "l", "0.0.0.0", "-l <server listen ip>")
	flag.IntVar(&port, "p", 65530, "-p <server listen port>")
	flag.IntVar(&cycle,"c",20,"-c <monitor cycle default 20 Second>")
	flag.Parse()
	serverIP := net.ParseIP(ip)

	s := &sdk.Server{
		Cycle:cycle,
		Laddr:&net.UDPAddr{IP:serverIP,Port:port},
		DataCh:make(chan *sdk.Message,1024),
		Host:make(map[string]string),
		HostInfoList: make(map[string]sdk.OsInf),
	}

	s.ListenAndServe()
	return s
}
