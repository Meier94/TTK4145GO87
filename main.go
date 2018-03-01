package main


import (
	"87/elev"
	"87/network"
	"87/client"
	"87/elev/io"
	"87/print"
	"runtime"
	"flag"
	"os"
	"syscall"
	"os/signal"
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
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(){
	    <- c 
        io.SetMotor(3)
        os.Exit(1)
	}()

	var ids string
	flag.StringVar(&ids, "id", "", "id of this peer")
	flag.Parse()

	idn, _ := strconv.Atoi(ids)
	id := uint8(idn)
	fmt.Printf("%d\n",id)

	print.Init()
	print.Line(runtime.Version())
	if !elev.Init(id){
		print.Line("Couldn't start io")
		return
	}
	client.Init(id)

	com.Start(id, client.ClientInit)

	for{
		print.Display()
		time.Sleep(400*time.Millisecond)
	}
	
}
