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

func setStopSignal(){
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(){
	    <- c 
        io.SetMotor(3)
        os.Exit(1)
	}()
}

func main() {
	// Our id can be anything. Here we pass it on the command line, using
	//  `go run main.go -id=our_id`
	
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

	setStopSignal()
	
	client.Init(id)
	ConnectionHandler := client.NewClient
	com.Start(id, ConnectionHandler)

	for{
		print.Display()
		time.Sleep(200*time.Millisecond)
	}
	
}
