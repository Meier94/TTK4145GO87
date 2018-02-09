package tcpmod

import (
	"net"
	"fmt"
	"bufio"
	"os"
	"strings"
	"time"
	"io"
	"bytes"
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

func Tcp_client2(){
	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":4487")

	// accept connection on port
	conn, _ := ln.Accept()
	fmt.Printf("Connected to %q\n", conn.RemoteAddr())

	// run loop forever (or until ctrl-c)
	for {
		// will listen for message to process ending in newline (\n)
		buf := make([]byte, 10)
		for {
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, err := conn.Read([]byte(buf))
			if err != nil {
				fmt.Printf("Read fail: %d, %s", n, err)
				time.Sleep(1 * time.Second)
				continue
			}
			fmt.Printf("Bytes read: %d, %d, %d\n", n, buf[n-2], buf[n-1])
			break
		}

		// output message received
		fmt.Printf("Message Received: %s", buf)
		// sample process for string received
		newmessage := strings.ToUpper(string(buf))
		// send new string back to client
		conn.Write([]byte(newmessage))
	}
}

func Tcp_client(){
	fmt.Println("Launching server...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":4487")

	// accept connection on port
	conn, _ := ln.Accept()
	fmt.Printf("Connected to %q\n", conn.RemoteAddr())
	// run loop forever (or until ctrl-c)
	count := 0
	for {
		// will listen for message to process ending in newline (\n)
		buf := []byte{'g','o','l','a','n','g'}
		buf2 := []byte{'g','o','l','a','n','g'}
		conn.Write([]byte(buf))
		for {
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, err := conn.Read([]byte(buf))
			if err != nil {
				fmt.Printf("Read fail: %d,%s, count: %d\n", n, err, count)
				return
			}
			if bytes.Compare(buf,buf2) != 0 {
				fmt.Printf("Byte arrays different\n")
				return;
			}
			count++
			break
		}
	}
}

func Tcp_server2(){
	// connect to this socket
	var err error
	var conn net.Conn

	//loops infinitely until it manages to connect
	for {
		conn, err = net.Dial("tcp", "129.241.187.152:4487")
		if err == nil {
			break
		}
		//fmt.Println(err)
		time.Sleep(100 * time.Millisecond)
	}
	for { 
		// read in input from stdin
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Text to send: ")
		text, _ := reader.ReadString('\n')
		// send to socket
		//fmt.Fprintf(conn, text + "\n")
		/*n, err := */conn.Write([]byte(text))
		// listen for reply
		message, _ := bufio.NewReader(conn).ReadString('\n')
		fmt.Print("Message from server: "+message)
	}
}

func Tcp_server(){
	// connect to this socket
	var err error
	var conn net.Conn

	//loops infinitely until it manages to connect
	for {
		conn, err = net.Dial("tcp", "129.241.187.152:4487")
		if err == nil {
			break
		}
		//fmt.Println(err)
		time.Sleep(100 * time.Millisecond)
	}
	count := 0
	for {
		buf := []byte{'g','o','l','a','n','g'}
		buf2 := []byte{'g','o','l','a','n','g'}
		for {
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, err := conn.Read([]byte(buf))
			if err != nil {
				fmt.Printf("Read fail: %d,%s, count: %d\n", n, err, count)
				return
			}
			if bytes.Compare(buf,buf2) != 0 {
				fmt.Printf("Byte arrays different\n")
				return
			}
			count++
			break
		}
		conn.Write([]byte(buf))
	}
}
