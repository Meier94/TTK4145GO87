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
	"container/list"
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


type msg_t struct{
	msgId int
	length int
	data []byte
}

type protocolChannel struct{
	pId int
	c chan msg_t
}

type client struct{
	var conn net.Conn
	var recv chan msg_t
	var send chan msg_t
	
}


func ClientListen(c client){
	num_rcvd := 0
	num_sent := 0

	client.recv = make(chan msg_t)
	client.send = make(chan msg_t)

	l := list.New()

	//Tcp receiver, passes incoming messages to client.recv
	TcpListen(&client, &l)

	for {
		select {
			case recv <- client.recv {
				go runProtocol(recv, client.conn)
			}
			case send <- client.send {
				go some_protocol(send, client.conn)
			}
		}
	}

}

func runProtocol(m msg_t, recv chan msg_t, c *client){
	var msg msg_t
	switch m.msgId{
		case NEW_ENTRY:{

			//send some message
			client.conn.Write(msg)
			//wait for reply
			msg <- recv

			//interpret
			dosomething

			//terminate protocol
			

		}
		case OTHER_CASE:{

		}
	}

}
//testfunksjon som ikke er så bra lenger i guess viser though en annen måte å lese på pluss input fra bruker

//Ikke testet fullstendig. Lagt til timeoutsjekk som er helt ny
//Må lage message translation funksjon
func TcpListen(c *client, l *List){
	for {
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		n, err := c.conn.Read([]byte(buf))
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout(){
				continue
			}
			fmt.Printf("Read fail: %d,%s, count: %d\n", n, err, count)
			return
		}

		//Translate into msg format

		//Notify protocol routine
		recvChan := GetChannelById(l, msg.msgId)
		if recvChan == nil {
			newchan := make(chan msg_t)
			go runProtocol(msg, newchan, c)
			l.PushBack(protocolChannel{pId = msg.msgId, c = newchan})
		}
	}
}

//Ikke testet
func GetChannelById(l *List, id int){
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value.pId == id {
			return e.Value.c
		}
	}
	return nil
}

//Klarer å sende ~215k meldinger frem og tilbake med annen maskin på 30 sek (139 us pr meldingsutveksling (mld + ack))
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



/*
import (
    "encoding/base64"
    "encoding/gob"
    "bytes"
)

type SX map[string]interface{}

// go binary encoder
func ToGOB64(m SX) string {
    b := bytes.Buffer{}
    e := gob.NewEncoder(&b)
    err := e.Encode(m)
    if err != nil { fmt.Println(`failed gob Encode`, err) }
    return base64.StdEncoding.EncodeToString(b.Bytes())
}

// go binary decoder
func FromGOB64(str string) SX {
    m := SX{}
    by, err := base64.StdEncoding.DecodeString(str)
    if err != nil { fmt.Println(`failed base64 Decode`, err); }
    b := bytes.Buffer{}
    b.Write(by)
    d := gob.NewDecoder(&b)
    err = d.Decode(&m)
    if err != nil { fmt.Println(`failed gob Decode`, err); }
    return m
}*/