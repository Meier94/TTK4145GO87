package tcpmod

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
		fmt.Print("detected closed LAN connection")
		c.Close()
		c = nil
	} else {
		c.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	}
}

func Tcp_client(){
	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":4487")

	// accept connection on port
	conn, _ := ln.Accept()
	fmt.Print("")

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

func Tcp_server(){
	// connect to this socket
	conn, _ := net.Dial("tcp", "129.241.187.152:4487")
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
