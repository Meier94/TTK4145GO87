package main

import (
//	"./network/bcast"
	"./network/localip"
//	"./network/peers"
	"./network/tcpmod"
	"net"
	"flag"
	"fmt"
	"os"
	"time"
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

	// ... or alternatively, we can use the local IP address.
	// (But since we can run multiple programs on the same PC, we also append the
	//  process ID)
	var localIP string
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}
	localIP, err := localip.LocalIP()
	if err == nil {
		println(localIP)
	}
	
	if id == "1" {
		var conn net.Conn
		var err error
		for {
			conn, err = net.Dial("tcp", "129.241.187.78:4487")
			if err == nil {
				break
			}
			//fmt.Println(err)
			time.Sleep(10 * time.Millisecond)
		}
		go tcpmod.ClientListen(conn, 2, false)

	}
	if id == "2" {
		// listen on all interfaces
		ln, _ := net.Listen("tcp", ":4487")
		// accept connection on port
		conn, _ := ln.Accept()
		go tcpmod.ClientListen(conn, 1, true)
	}
	tcpmod.ReadInput()
}
