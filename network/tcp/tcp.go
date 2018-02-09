package tcp

import (
	"net"
	"fmt"
	"bufio"
	"os"
	"strings"
	"time"
	"io"
)


func CheckDisconnect(c net.Conn){
	one := []byte{}
	c.SetReadDeadline(time.Now())
	if _, err := c.Read(one); err == io.EOF {
		fmt.Printf("%s detected closed LAN connection", id)
		c.Close()
		c = nil
	} else {
		var zero time.Time
		c.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	}
}

func tcp_client(){
	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":8081")

	// accept connection on port
	conn, _ := ln.Accept()

	// run loop forever (or until ctrl-c)
	for {
		// will listen for message to process ending in newline (\n)
		message, _ := bufio.NewReader(conn).ReadString('\n')
		// output message received
		fmt.Print("Message Received:", string(message))
		// sample process for string received
		newmessage := strings.ToUpper(message)
		// send new string back to client
		conn.Write([]byte(newmessage + "\n"))
	}
}

func tcp_server(){
	// connect to this socket
	conn, _ := net.Dial("tcp", "127.0.0.1:8081")
	for { 
		// read in input from stdin
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Text to send: ")
		text, _ := reader.ReadString('\n')
		// send to socket
		fmt.Fprintf(conn, text + "\n")
		// listen for reply
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print("Message from server: "+message)
	}
}
