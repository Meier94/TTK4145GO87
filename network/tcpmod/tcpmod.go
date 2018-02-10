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


type msg_t struct{
	msgId int
	length int
	data []byte
}

type client struct{
	var conn net.Conn
	var recv chan msg_t
	var send chan msg_t
	
}


func ClientListen(c client){
	num_rcvd := 0
	num_sent := 0

	client_c = make(chan *msg_t)
	talks_m := make(map[int]chan *msg_t)

	//Tcp receiver, passes incoming messages to client_c
	go TcpListen(&client, &client_c)

	for {
		select {
			//TcpListen has received a message from client
			case newMsg := <- client_c {
				//Notify correct protocol
				if !notifyTalk(talks_m, newMsg){
					new_c := make(chan *msg_t)
					go runProtocol(msg, new_c, c)
					talks_m[msg.pId] c = new_c
				}
			}
			case dc <- listen_c {
				//Client is non responsive
				//Distribute his orders
				//Notify talks_m
				//Return
			}
		}
	}

}

func notifyTalk(talks_m map[int]int, msg *msg_t){
	recvChan := talks_m[msg.pId]
	if recvChan == nil {
		return 0
	}
	recvChan <- newMsg
	return 1
}


//testfunksjon som ikke er så bra lenger i guess viser though en annen måte å lese på pluss input fra bruker
//Ikke testet fullstendig. Lagt til timeoutsjekk som er helt ny
//Må lage message translation funksjon
func TcpListen(c *client, notify chan<- *msg_t){
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
		msg := Translate(buf)

		notify <- msg

	}
}


//Ganske experimental nedenfor


func runProtocol(m *msg_t, talk_c <-chan *msg_t, c *client){
	rcvMsgId := m.msgId
	sndMsgId := 0
	var msg msg_t
	switch m.msgId{
		case HANDLE_ORDER:{
			status_c = addOrder(m.data)
			//compose answer
			var msg msg_t = {pId = m.pId, msgId = sndMsgId, data = ACK}
			//send some message (twice for redundancy)
			client.conn.Write(msg)
			client.conn.Write(msg)
			//wait for reply
			for {
				select {
				case status := <- status_c:
					msg.msgId++
					msg.data = COMPLETE
					client.conn.Write(msg)
					//TODO: wait for ack
				case msg <- talk_c:
					if msg.msgId == rcvMsgID{
						client.conn.Write(msg)
					}
					else if msg.data == COMPLETE{
						sm->removeOrder()
						return;
					}

				}
			}


			msg <- talk_c


			//interpret
			dosomething

			//terminate protocol
			

		}
		case ALIVE_PING:{

		}
	}
}

func trySend(c *client, m *msg_t, talk_c <-chan *msg_t, lastId int){
	//send some message
	client.conn.Write(msg)
	//wait for reply
	msg <- talk_c

	if msg.msgId == rcvMsgID{
		//rewrite
	}
}


//ikke i bruk as of now
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