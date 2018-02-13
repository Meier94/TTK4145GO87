package tcp

import (
	"net"
	"fmt"
	"bufio"
	"os"
//	"strings"
	"time"
//	"io"
	"bytes"
	"encoding/binary"
	"sync"
)

//felt må ha stor forbokstav for å kunne bli konvertert fra []byte
type msg_t struct{
	TalkId uint32	//talk-id
	ClientId uint8
	Type uint8
	Data uint16 
}

type client struct{
	id uint8
	conn net.Conn
	dc_c chan bool
	talkDone_c chan uint32	
}


var connectionMap map[string]int
var mapTex *sync.Mutex
var myID uint8

func Init(id uint8){
	mapTex = &sync.Mutex{}
	connectionMap = make(map[string]int)
	myID = id
}

func ClientListen(conn net.Conn, dialer bool){
//	num_rcvd := 0
//	num_sent := 0
	var TalkCounter uint32 = 0
	if dialer {
		TalkCounter++
	}
	elev_c = make(chan *msg_t)
	msg_c := make(chan *msg_t)
	talks_m := make(map[uint32]chan *msg_t)

	var cli client
	c := &cli
	c.id = 0
	c.conn = conn
	c.talkDone_c = make(chan uint32)
	c.dc_c = make(chan bool)


	go TcpListen(c, msg_c)

	

	for {
		select {
			//TcpListen has received a message from client
			case newMsg, ok := <- msg_c: {
				if !ok {
					//channel is closed
					//Client is non responsive
					//Notify talks:
					close(c.dc_c)

					//Distribute his/her orders

					//issue that it returns before talks are finished cleaning up?

					//remove itself from map
					return
				}
				//Notify correct protocol
				if !notifyTalk(talks_m, newMsg){
					new_c := make(chan *msg_t)
					go runProtocol(newMsg, new_c, c, false)
					talks_m[newMsg.TalkId] = new_c
				}
			}

			case newMsg := <- elev_c :
				newMsg.TalkId = TalkCounter
				new_c := make(chan *msg_t)
				talks_m[TalkCounter] = new_c
				go runProtocol(newMsg, new_c, c, true)
				TalkCounter += 2

			case id := <- c.talkDone_c:{
				delete(talks_m, id)
			}
		}
	}
}

func notifyTalk(talks_m map[uint32]chan *msg_t, msg *msg_t) bool{
	recvChan := talks_m[msg.TalkId]
	if recvChan == nil {
		return false
	}
	recvChan <- msg
	return true
}


func send(msg *msg_t, c *client){
	buf := toBytes(msg)
	c.conn.Write(buf)
}


func toMsg(data []byte) *msg_t{
	var msg msg_t
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &msg)
	if err != nil {
		panic(err)
	}
	return &msg
}


func toBytes(data *msg_t) []byte{
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}


//Ikke testet.
func TcpListen(c *client, msg_c chan<- *msg_t){
	buf := make([]byte, BUFLEN)
	for {
		//c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil || n != int(BUFLEN){
			fmt.Printf("Read failed client %q\n", c.conn.RemoteAddr())
			c.conn.Close()
			close(msg_c)
			return
		}

		msg := toMsg(buf)
		//Translate into msg format
		msg_c <- msg
	}
}

func ReadInput(){
	reader := bufio.NewReader(os.Stdin)
	var msg msg_t
	msg.Type = CALL
	msg.ClientId = 0
	msg.Data = 3
	for {
	    fmt.Print("Text to send: ")
	    text, _ := reader.ReadString('\n')
	    switch text{
	    case "add\n":
	    	elev_c <- &msg
	    case "complete\n":
	    	msg.Type = COMPLETE_CALL
	    	elev_c <- &msg
	    case "fail\n":
	    	msg.Type = FAILED_CALL
	    	elev_c <- &msg
	    default:
	    	fmt.Printf("%s\n",text)
	    	fmt.Printf("Didn't catch test\n")
	    }
	}
}


func addToMap(ip string) bool {
	mapTex.Lock()
	_, ok := connectionMap[ip]
	if !ok {
		mapTex.Unlock()
		return false
	}
	connectionMap[ip] = 1
	mapTex.Unlock()
	return true
}


func UdpListen(){

	Addr,err := net.ResolveUDPAddr("udp",":55087")
    if err != nil {
    	panic(err)
    }
	SerConn, err := net.ListenUDP("udp", Addr)
    if err != nil {
        fmt.Printf("Some error %v\n", err)
        return
    }
    buf := make([]byte, 1024)
    defer SerConn.Close()
	for {
		// connect to this socket
		n,addr,err := SerConn.ReadFromUDP(buf)
        fmt.Println("Received ",string(buf[0:n]), " from ",addr)
        if err != nil {
            fmt.Println("Error: ",err)
            continue
        }

        if buf[0] > myID {
        	continue
        }

		ip,_,_ := net.SplitHostPort(addr.String())

        if !addToMap(ip) {
        	continue
        }
        

		var conn net.Conn
		for i := 0; i < 3; i++{
			conn, err = net.Dial("tcp", ip+":4487")
			if err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			continue
		}
		go ClientListen(conn, true)
	}
}

func UdpBroadcast(){
	ServerAddr,err := net.ResolveUDPAddr("udp","255.255.255.255:55087")
    if err != nil {
        fmt.Println("Error: ",err)
    }

 
    Conn, err := net.DialUDP("udp", nil, ServerAddr)
    if err != nil {
        fmt.Println("Error: ",err)
    }
 
    defer Conn.Close()
    for {
        buf := []byte{myID}
        _,err := Conn.Write(buf)
        if err != nil {
            fmt.Println(err)
        }
        time.Sleep(time.Second * 1)
    }
}

func TcpAccept(){
	for {
		ln, err := net.Listen("tcp", ":4487")
		if err != nil {
			// handle error
			continue
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				// handle error
				continue
			}
			ip,_,_ := net.SplitHostPort(conn.RemoteAddr().String())
			if !addToMap(ip) {
	        	continue
	        }
			
			fmt.Printf("Connected to %s\n", conn.RemoteAddr())
			go ClientListen(conn, false)
		}
	}
}


//placeholder for compilation
func addOrder(data uint16) {
	fmt.Printf("Order to be handled by me: %d\n",data)
}

func CallAccepted(data uint16, id uint8){
	fmt.Printf("Call acc: %d, %d\n", data, id)
}

func CallUnhandled(data uint16){
	fmt.Printf("Call unhandled: %d\n", data)
}

var elev_c chan *msg_t


//flags
//Message Types
const COMPLETE_CALL uint8 = 200
const ACK uint8 = 201
const CALL uint8 = 202
const FAILED_CALL uint8 = 204
const BUFLEN uint8 = 8

func runProtocol(msg *msg_t, talk_c <-chan *msg_t, c *client, outgoing bool){
	if outgoing {
		switch msg.Type{
		case CALL :
			Call_out(msg, talk_c, c)
		case COMPLETE_CALL : 
			CallComplete_out(msg, talk_c, c)
		case FAILED_CALL :
			CallFailed_out(msg, talk_c, c)
		}
	} else {
		switch msg.Type{
		case CALL:
			Call_in(msg, talk_c, c)
		case COMPLETE_CALL : 
			CallComplete_in(msg, talk_c, c)
		case FAILED_CALL :
			CallFailed_in(msg, talk_c, c)
		}
	}
}

//Ikke fornøyd med denne måten å gjøre ting på
//Vil helst at man skal anta at tcp meldinger går gjennom og heller sørge for ack med timeout
func Call_in(msg *msg_t, talk_c <-chan *msg_t, c *client){

	addOrder(msg.Data)
	sendACK(msg, talk_c, c)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func Call_out(msg *msg_t, talk_c <-chan *msg_t, c *client){

	send(msg, c)
	if getACK(msg, talk_c, c) {
		CallAccepted(msg.Data, c.id)
	} else {
		CallUnhandled(msg.Data)
	}
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}



func CallFailed_out(msg *msg_t, talk_c <-chan *msg_t, c *client){
	
	send(msg, c)
	getACK(msg, talk_c, c)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func CallFailed_in(msg *msg_t, talk_c <-chan *msg_t, c *client){

	fmt.Printf("Remove order %d\n", msg.Data)
	sendACK(msg, talk_c, c)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func CallComplete_out(msg *msg_t, talk_c <-chan *msg_t, c *client){

	send(msg, c)
	getACK(msg, talk_c, c)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func CallComplete_in(msg *msg_t, talk_c <-chan *msg_t, c *client){

	fmt.Printf("Remove order %d\n", msg.Data)
	sendACK(msg, talk_c, c)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}


func getACK(msg *msg_t, talk_c <-chan *msg_t, c *client) bool {
	for {
		select {
		case rcvMsg := <- talk_c: 
			if rcvMsg.Type == ACK {
				return true
			} else {
				fmt.Printf("Talk : %d, Received unexpected message: %d\n", rcvMsg.TalkId, rcvMsg.Type)
			}
		case <- c.dc_c :
			//client dc
			return false
		case <- time.After(40 * time.Millisecond) :
			//Ack not received
			fmt.Printf("Ack not received\n")
			send(msg, c)
		}
	}
}

func sendACK(msg *msg_t, talk_c <-chan *msg_t, c *client) bool {
	//Wait for call to be handled / request for new ack if prev failed
	msg.Type = ACK
	send(msg, c)
	for {
		select {
		case rcvMsg := <- talk_c: 
			//Ack not received
			send(msg, c)
			fmt.Printf("Talk : %d, Resending: %d\n", rcvMsg.TalkId, rcvMsg.Type)
		case <- c.dc_c :
			//client dc
			return false
		case <- time.After(100 * time.Millisecond) :
			//Ack assumed received
			return true
		}
	}
}