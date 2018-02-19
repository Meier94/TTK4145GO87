package com

import (
	"../statemap"
	"net"
	"fmt"
	"bufio"
	"os"
	"time"
	"bytes"
	"encoding/binary"
	"sync"
)

//felt må ha stor forbokstav for å kunne bli konvertert fra []byte
type Msg_t struct{
	TalkId uint32
	ClientID uint8
	MsgID uint8
	Type uint8
	Evt sm.Evt
}

type client struct{
	id uint8
	smIndex int16
	conn net.Conn
	dc_c chan bool
	talkDone_c chan uint32
	msg_c chan *Msg_t
	evt_c chan *sm.Evt
	talks_m map[uint32]chan *Msg_t
}

var connections_m map[string]int
var mapTex *sync.Mutex
var talkTex *sync.Mutex
var myID uint8

func Init(id uint8){
	mapTex = &sync.Mutex{}
	talkTex = &sync.Mutex{}
	connections_m = make(map[string]int)
	myID = id
}

//flags
//Message Types
const ACK uint8 = 201
const CALL uint8 = 202
const PING uint8 = 205
const INTRO uint8 = 206
const EVT uint8 = 207
const BUFLEN uint8 = 13


func ClientInit(conn net.Conn){
	msg := Msg_t{ClientID: myID, Type: INTRO}
	status := &msg.Evt
	status.Floor, status.Target, status.Stuck = sm.GetState(0)
	send(&msg, conn)

	intro := TcpRead(conn)
	if intro == nil || intro.Type != INTRO {
		ip,_,_ := net.SplitHostPort(conn.RemoteAddr().String())
		removeFromMap(ip)
		conn.Close()
		return
	}
	status = &intro.Evt

	var cli client
	cli.id 			= intro.ClientID
	cli.conn 		= conn
	cli.talkDone_c  = make(chan uint32)
	cli.dc_c 		= make(chan bool)
	cli.evt_c 		= make(chan *sm.Evt)
	cli.msg_c 		= make(chan *Msg_t)
	cli.talks_m 	= make(map[uint32]chan *Msg_t)
	cli.smIndex		= sm.AddNode(cli.id, status.Floor, status.Target, status.Stuck, cli.evt_c)

	go ClientListen(&cli)
}

func closeClient(c *client){
	fmt.Printf("Connection to %s closed.\n",c.conn.RemoteAddr().String())

	sm.RemoveNode(c.smIndex)
	talkTex.Lock()
	close(c.talkDone_c)
	c.talkDone_c = nil
	talkTex.Unlock()
	close(c.dc_c)

	//issue that it returns before talks are finished cleaning up?
	//remove itself from map
	ip,_,_ := net.SplitHostPort(c.conn.RemoteAddr().String())
	removeFromMap(ip)
}

func ClientListen(c *client){
	var TalkCounter uint32 = 0
	if myID < c.id {
		TalkCounter++
	}

	go TcpListen(c, c.msg_c)
	go Ping_out(myID,TalkCounter, c)
	TalkCounter+=2
	

	for {
		select {
			//TcpListen has received a message from client
			case newMsg, ok := <- c.msg_c: {
				if !ok {
					//Client is non responsive
					closeClient(c)
					return
				}
				//Notify correct protocol
				if !notifyTalk(c.talks_m, newMsg){
					new_c := make(chan *Msg_t)
					go runProtocol(newMsg, new_c, c, false)
					c.talks_m[newMsg.TalkId] = new_c
				}
			}

			case evt := <- c.evt_c :
				newMsg := Msg_t{TalkId: TalkCounter, Type: EVT, Evt: *evt}
				new_c := make(chan *Msg_t)
				c.talks_m[TalkCounter] = new_c
				go runProtocol(&newMsg, new_c, c, true)
				TalkCounter += 2

			case id := <- c.talkDone_c:{
				delete(c.talks_m, id)
			}
		}
	}
}

func notifyTalk(talks_m map[uint32]chan *Msg_t, msg *Msg_t) bool{
	recvChan := talks_m[msg.TalkId]
	if recvChan == nil {
		return false
	}
	recvChan <- msg
	return true
}


func send(msg *Msg_t, conn net.Conn){
	buf := toBytes(msg)
	conn.Write(buf)
}


func toMsg(data []byte) *Msg_t{
	var msg Msg_t
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &msg)
	if err != nil {
		panic(err)
	}
	return &msg
}


func toBytes(data *Msg_t) []byte{
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}


func TcpListen(c *client, msg_c chan<- *Msg_t){
	buf := make([]byte, BUFLEN)
	for {
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil || n != int(BUFLEN){

			if err, ok := err.(net.Error); ok && err.Timeout() {
        		fmt.Printf("Timeout caused disconnect\n")
    		}
			c.conn.Close()
			close(msg_c)
			return
		}

		msg := toMsg(buf)
		//Translate into msg format
		msg_c <- msg
	}
}

func TcpRead(conn net.Conn) *Msg_t{
	buf := make([]byte, BUFLEN)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := conn.Read(buf)
	if err != nil || n != int(BUFLEN){

		if err, ok := err.(net.Error); ok && err.Timeout() {
    		fmt.Printf("Tcp read timed out\n")
		} else {
			fmt.Printf("Tcp read failed\n")
		}
		conn.Close()
		return nil
	}
	msg := toMsg(buf)
	return msg
}


func ReadInput(){
	reader := bufio.NewReader(os.Stdin)
	var msg Msg_t
	msg.Type = CALL
	msg.ClientID = 0
	for {
	    fmt.Print("Text to send: ")
	    text, _ := reader.ReadString('\n')
	    switch text{
	    case "add\n":
	    	elev_c <- &msg
	    case "complete\n":
	    	msg.Type = EVT
	    	elev_c <- &msg
	    case "fail\n":
	    	msg.Type = PING
	    	elev_c <- &msg
	    default:
	    	fmt.Printf("%s\n",text)
	    	fmt.Printf("Didn't catch test\n")
	    }
	}
}


func addToMap(ip string) bool {
	mapTex.Lock()
	_, ok := connections_m[ip]
	if ok {
		mapTex.Unlock()
		return false
	}
	connections_m[ip] = 1
	mapTex.Unlock()
	return true
}

func removeFromMap(ip string){
	mapTex.Lock()
	delete(connections_m, ip)
	mapTex.Unlock()
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
		_,addr,err := SerConn.ReadFromUDP(buf)
        
        //fmt.Println("Received UDP from id:",buf[0], ", ip: ",addr)
        if err != nil {
            fmt.Println("Error: ",err)
            continue
        }

        if buf[0] >= myID {
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
			removeFromMap(ip)
			continue
		}
		fmt.Printf("Connection established, id: %d\n", buf[0])
		go ClientInit(conn)
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
			go ClientInit(conn)
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

var elev_c chan *Msg_t


//rewrite
func runProtocol(msg *Msg_t, talk_c <-chan *Msg_t, c *client, outgoing bool){
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
		case PING :
			Ping_in(msg, talk_c, c)
		case CALL :
			Call_in(msg, talk_c, c)
		case COMPLETE_CALL : 
			CallComplete_in(msg, talk_c, c)
		case FAILED_CALL :
			CallFailed_in(msg, talk_c, c)
		}
	}
}

func endTalk(c *client, id uint32){
	talkTex.Lock()
	if c.talkDone_c != nil {
		c.talkDone_c <- id
	}
	talkTex.Unlock()
}


func Ping_in(msg *Msg_t, talk_c <-chan *Msg_t, c *client){
	lastID := msg.MsgID
	run := true
	for run{
		select {
		case msg := <- talk_c: 
			if(msg.MsgID != lastID){
				lastID = msg.MsgID
				//notify sm
			}
		case <- c.dc_c :
			//client dc
			run = false
		}
	}
	endTalk(c,msg.TalkId)
}


func Ping_out(id uint8, talkId uint32, c *client){
	var lastID uint8 = 0
	msg := Msg_t{talkId, id, lastID, PING, 0}
	for{
		select {
		case <- c.dc_c :
			//client dc
			return
		case <- time.After(30 * time.Millisecond) :
			msg.MsgID++
			send(&msg, c.conn)
		}
	}
}

//Ikke fornøyd med denne måten å gjøre ting på
//Vil helst at man skal anta at tcp meldinger går gjennom og heller sørge for ack med timeout
func Call_in(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	addOrder(msg.Data)
	sendACK(msg, talk_c, c)
	endTalk(c,msg.TalkId)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func Call_out(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	send(msg, c.conn)
	if getACK(msg, talk_c, c) {
		CallAccepted(msg.Data, c.id)
	} else {
		CallUnhandled(msg.Data)
	}
	endTalk(c,msg.TalkId)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}



func CallFailed_out(msg *Msg_t, talk_c <-chan *Msg_t, c *client){
	
	send(msg, c.conn)
	getACK(msg, talk_c, c)
	endTalk(c,msg.TalkId)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func CallFailed_in(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	fmt.Printf("Remove order %d\n", msg.Data)
	sendACK(msg, talk_c, c)
	endTalk(c,msg.TalkId)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func CallComplete_out(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	send(msg, c.conn)
	getACK(msg, talk_c, c)
	endTalk(c,msg.TalkId)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}

func CallComplete_in(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	fmt.Printf("Remove order %d\n", msg.Data)
	sendACK(msg, talk_c, c)
	endTalk(c,msg.TalkId)
	fmt.Printf("Goroutine ended %d\n", msg.Type)
}


func getACK(msg *Msg_t, talk_c <-chan *Msg_t, c *client) bool {
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
			send(msg, c.conn)
		}
	}
}

func sendACK(msg *Msg_t, talk_c <-chan *Msg_t, c *client) bool {
	//Wait for call to be handled / request for new ack if prev failed
	msg.Type = ACK
	send(msg, c.conn)
	for {
		select {
		case rcvMsg := <- talk_c: 
			//Ack not received
			send(msg, c.conn)
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