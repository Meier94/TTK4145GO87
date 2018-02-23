package com

import (
	"../statemap"
	"net"
	"fmt"
	"time"
	"bytes"
	"encoding/binary"
	"sync"
	"runtime"
)

type Et struct{
	Type uint8
	Floor int16
	Target int16
	Button uint8
	Stuck bool
	Supervise bool
}

//felt må ha stor forbokstav for å kunne bli konvertert fra []byte
type Msg_t struct{
	TalkID uint32
	ClientID uint8
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
	fmt.Println(runtime.Version())
	BUFLEN = uint8(binary.Size(Msg_t{}))
	mapTex = &sync.Mutex{}
	talkTex = &sync.Mutex{}
	connections_m = make(map[string]int)
	myID = id
}

//flags
//Message Types
const ACK uint8 = 200
const PING uint8 = 201
const INTRO uint8 = 202
const EVT uint8 = 203

var BUFLEN uint8 = 14
var talks uint32 = 0


func ClientInit(conn net.Conn, flag bool){
	msg := Msg_t{ClientID: myID, Type: INTRO}
	status := &msg.Evt
	status.Floor, status.Target, status.Stuck = sm.GetState(0)
	println(conn.RemoteAddr().String())
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

	fmt.Printf("Node added, Floor: %d, Target: %d, Stuck: %t, ID: %d\n", status.Floor, status.Target, status.Stuck, cli.id)

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
	go Ping_out(TalkCounter, c)
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
					newTalk(newMsg, c, nil, false)
				}
			}

			case evt := <- c.evt_c :
				newMsg := &Msg_t{Type: EVT, Evt: *evt}
				newTalk(newMsg, c, &TalkCounter, true)

			case id := <- c.talkDone_c:{
				delete(c.talks_m, id)
			}
		}
	}
}

func newTalk(msg *Msg_t, c *client, counter *uint32, outgoing bool){
	new_c := make(chan *Msg_t)
	talkTex.Lock()

	talks++
	if outgoing {
		msg.TalkID = *counter
		*counter += 2
	}

	c.talks_m[msg.TalkID] = new_c
	talkTex.Unlock()

	go runProtocol(msg, new_c, c, outgoing)
}

func notifyTalk(talks_m map[uint32]chan *Msg_t, msg *Msg_t) bool{
	if msg.TalkID > 1 {
		talkTex.Lock()
		recvChan := talks_m[msg.TalkID]
		talkTex.Unlock()
		if recvChan == nil {
			return false
		}
		recvChan <- msg
	}
	return true
}


func send(msg *Msg_t, conn net.Conn){
	buf := toBytes(msg)
	_, err := conn.Write(buf)
	if testErr(err, "") {
		panic(err)
	}
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
	err := binary.Write(buf, binary.BigEndian, *data)
	if testErr(err, "Couldn't convert message") {
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
	conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
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

func perror(err error, msg string){
	if err != nil {
		fmt.Println(msg)
		panic(err)
	}
}

func testErr(err error, msg string) bool {
	if err != nil {
		fmt.Println(msg,err)
		return true
	}
	return false
}



func UdpListen(){

	Addr,err := net.ResolveUDPAddr("udp",":55087")
    perror(err, "Couldn't resolve UDP listen")

	SerConn, err := net.ListenUDP("udp", Addr)
    perror(err, "Couldn't listen to udp")

    buf := make([]byte, 1024)
    defer SerConn.Close()
	for {
		// connect to this socket
		_,addr,err := SerConn.ReadFromUDP(buf)
        
        if testErr(err, "UDP read failed") || buf[0] >= myID {
            continue
        }

		ip,_,_ := net.SplitHostPort(addr.String())
        if !addToMap(ip) {
        	continue
        }

		var conn net.Conn
		for i := 0; i < 3; i++ {
			conn, err = net.Dial("tcp", ip + ":4487")

			if !testErr(err, "TCP dial failed") {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if testErr(err, "TCP dial couldn't reach client. Removing from map") {
			removeFromMap(ip)
			continue
		}

		fmt.Printf("Connection established, id: %d\n", buf[0])
		go ClientInit(conn, true)
	}
}

func UdpBroadcast(){
	for {
		ServerAddr,err := net.ResolveUDPAddr("udp","255.255.255.255:55087")
	    if testErr(err, "Couldn't resolve UDPAddr broadcast") {
	        continue
	    }

	    Conn, err := net.DialUDP("udp", nil, ServerAddr)
	    if testErr(err, "Couldn't establish UDP connection") {
	        continue
	    }
	 
	    defer Conn.Close()
	    for {
	        buf := []byte{myID}
	        _,err := Conn.Write(buf)
	        testErr(err, "UDP write failed")

	        time.Sleep(time.Second * 1)
	    }
	}
}

func TcpAccept(){
	for {
		ln, err := net.Listen("tcp", ":4487")
		if testErr(err, "TCP Listen failed") {
			continue
		}
		for {
			conn, err := ln.Accept()
			if testErr(err, "Accept TCP failed") {
				continue
			}
			ip,_,_ := net.SplitHostPort(conn.RemoteAddr().String())
			if !addToMap(ip) {
	        	continue
	        }
			
			fmt.Printf("Connected to %s\n", conn.RemoteAddr())
			go ClientInit(conn, false)
		}
	}
}


func endTalk(c *client, id uint32){
	talkTex.Lock()
	if c.talkDone_c != nil {
		c.talkDone_c <- id
	}
	talks--
	if talks == 0{
		fmt.Println("No talks active")
	}
	talkTex.Unlock()
}


func Ping_out(talkID uint32, c *client){
	msg := Msg_t{TalkID: talkID, Type: PING}
	for{
		select {
		case <- c.dc_c :
			//client dc
			return
		case <- time.After(30 * time.Millisecond) :
			send(&msg, c.conn)
		}
	}
}

func runProtocol(msg *Msg_t, talk_c <-chan *Msg_t, c *client, outgoing bool){
	if outgoing {
		switch msg.Type{
		default:
			go sendEvt(msg, talk_c, c)
		}

	} else {
		switch msg.Type{
		case PING:
			//nothing
		default:
			go recvEvt(msg, talk_c, c)
		}
	}
}

func sendEvt(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	send(msg, c.conn)
	if getACK(msg, talk_c, c) {
		sm.EvtAccepted(&msg.Evt, c.smIndex)
	} else {
		sm.EvtDismissed(&msg.Evt, c.smIndex)
	}
	endTalk(c,msg.TalkID)
	//fmt.Printf("Goroutine ended %d, %d\n", msg.Type, msg.TalkID)
}

func recvEvt(msg *Msg_t, talk_c <-chan *Msg_t, c *client){
	sm.EvtRegister(&msg.Evt, c.smIndex)
	sendACK(msg, talk_c, c)
	endTalk(c,msg.TalkID)
	//fmt.Printf("Goroutine ended %d, %d\n", msg.Type, msg.TalkID)
}


func getACK(msg *Msg_t, talk_c <-chan *Msg_t, c *client) bool {
	for {
		select {
		case rcvMsg := <- talk_c: 
			if rcvMsg.Type == ACK {
				return true
			} else {
				fmt.Printf("Talk : %d, Received unexpected message: %d\n", rcvMsg.TalkID, rcvMsg.Type)
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
			fmt.Printf("Talk : %d, Resending: %d\n", rcvMsg.TalkID, rcvMsg.Type)
		case <- c.dc_c :
			//client dc
			return false
		case <- time.After(100 * time.Millisecond) :
			//Ack assumed received
			return true
		}
	}
}