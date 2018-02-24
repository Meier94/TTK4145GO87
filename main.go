package main


import (
	"87/elev"
	"87/network"
	"87/client"
	"87/statemap"
	"runtime"
//	"net"
	"flag"
	"time"
	"fmt"
	"strconv"
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
	fmt.Println(runtime.Version())
	var ids string
	flag.StringVar(&ids, "id", "", "id of this peer")
	flag.Parse()

	idn, _ := strconv.Atoi(ids)
	id := uint8(idn)
	fmt.Printf("%d\n",id)

	if !elev.Init(id){
		fmt.Println("Couldn't start io")
		return
	}
	client.Init(id)

	com.Start(id, client.ClientInit)

	for{
		sm.PrintMap()
		time.Sleep(400*time.Millisecond)
	}
	
}
