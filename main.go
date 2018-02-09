package main

import (
//	"./network/bcast"
	"./network/localip"
//	"./network/peers"
	"./network/tcpmod"
	"flag"
	"fmt"
	"os"
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
		tcpmod.Tcp_client()

	}
	if id == "2" {
		tcpmod.Tcp_server()
		
	}

}
