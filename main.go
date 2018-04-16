package main


import (
	"xx/elev"
	"xx/network"
	"xx/client"
	"xx/elev/io"
	"xx/print"
	"runtime"
	"flag"
	"os"
	"syscall"
	"os/signal"
	"time"
	"fmt"
	"strconv"
)

func stopMotorOnExit(){
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func(){
	    <- c 
        io.SetMotor(3)
        os.Exit(1)
	}()
}

func main() {
	var idString string
	flag.StringVar(&idString, "id", "", "id of this peer")
	flag.Parse()

	idNumber, _ := strconv.Atoi(idString)
	id := uint8(idNumber)
	fmt.Printf("%d\n",id)

	print.Init()
	print.Line(runtime.Version())
	

	elev.Init(id)

	stopMotorOnExit()
	
	client.Init(id)
	ConnectionHandler := client.NewClient
	com.Start(id, ConnectionHandler)

	for{
		print.Display()
		time.Sleep(200*time.Millisecond)
	}
	
}
