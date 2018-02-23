package main


import (
	"./elev"
	"./network"
	"./client"
	"runtime"
//	"net"
	"flag"
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
	var id string
	flag.StringVar(&id, "id", "", "id of this peer")
	flag.Parse()

	
	idn,_:=strconv.Atoi(id)
	fmt.Printf("%d\n",idn)

	elev.Init(uint8(idn))
	client.Init(uint8(idn))

	com.Start(uint8(idn), client.ClientInit)

	for{}
	
}
