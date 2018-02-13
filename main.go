package main

import (
//	"./network/bcast"
//	"./network/peers"
	"./network/tcp"
//	"net"
	"flag"
	"fmt"
	"strconv"
//	"time"
//	"time"
)

// We define some custom struct to send over the network.
// Note that all members we want to transmit must be public. Any private members
//  will be received as zero-values.
type HelloMsg struct {
	Message string
	Iter    int
}

func main() {
	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()
	
	idn,_:=strconv.Atoi(id)
	fmt.Printf("%d\n",idn)
	tcp.Init(uint8(idn))

	go tcp.UdpListen()
	go tcp.TcpAccept()
	go tcp.UdpBroadcast()
	tcp.ReadInput()
}
