package client

import (
	"bytes"
	"encoding/binary"
	"87/statemap"
	"87/network"
	"sync"
	"fmt"
	"time"
)

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
	conn com.Connection
	dc_c chan bool
	talkDone_c chan uint32
	msg_c chan []byte
	evt_c chan *sm.Evt
	talks_m map[uint32]chan *Msg_t
}

//flags
//Message Types
const ACK uint8 = 200
const PING uint8 = 201
const INTRO uint8 = 202
const EVT uint8 = 203

var BUFLEN uint8
var talks uint32 = 0

var talkTex *sync.Mutex
var myID uint8


func Init(id uint8){
	talkTex = &sync.Mutex{}
	BUFLEN = uint8(binary.Size(Msg_t{}))
	myID = id
}

func testErr(err error, msg string) bool {
	if err != nil {
		//fmt.Println(msg,err)
		return true
	}
	return false
}

func ClientInit(conn com.Connection, flag bool){
	msg := Msg_t{ClientID: myID, Type: INTRO}
	status := &msg.Evt
	status.Floor, status.Target, status.Stuck = sm.GetState(0)
	client{conn: conn}.send(&msg)

	in := conn.TcpRead(BUFLEN)
	if in == nil {
		conn.Close()
		return
	}
	intro := toMsg(in)
	if intro.Type != INTRO {
		conn.Close()
		return
	}
	status = &intro.Evt

	var cli client
	cli.id 			= intro.ClientID
	cli.conn 		= conn
	cli.talkDone_c  = make(chan uint32)
	cli.dc_c 		= make(chan bool)
	cli.evt_c 		= make(chan *sm.Evt, 10)
	cli.msg_c 		= make(chan []byte, 10)
	cli.talks_m 	= make(map[uint32]chan *Msg_t)
	cli.smIndex		= sm.AddNode(cli.id, status.Floor, status.Target, status.Stuck, cli.evt_c)

	sm.Print(fmt.Sprintf("Node added, Floor: %d, Target: %d, Stuck: %t, ID: %d", status.Floor, status.Target, status.Stuck, cli.id))

	go ClientListen(&cli)
}

func closeClient(c *client){
	sm.Print(fmt.Sprintf("Connection to %d closed.",c.id))

	sm.RemoveNode(c.smIndex)
	talkTex.Lock()
	close(c.talkDone_c)
	c.talkDone_c = nil
	close(c.dc_c)


	for _, ch := range c.talks_m {
		close(ch)
	}
	talkTex.Unlock()
	//issue that it returns before talks are finished cleaning up?
	//remove itself from map
	c.conn.Close()
}

func ClientListen(c *client){
	var TalkCounter uint32 = 0
	if myID < c.id {
		TalkCounter++
	}

	go c.conn.Listen(c.msg_c, BUFLEN)
	go Ping_out(TalkCounter, c)
	TalkCounter+=2
	

	for {
		select {
			//TcpListen has received a message from client
			case data, ok := <- c.msg_c: {
				if !ok {
					//Client is non responsive
					closeClient(c)
					return
				}
				newMsg := toMsg(data)
				//Notify correct protocol
				if !notifyTalk(c.talks_m, newMsg){
					newTalk(newMsg, c, nil, false)
				}
			}

			case evt := <- c.evt_c :
				newMsg := &Msg_t{Type: EVT, Evt: *evt}
				newTalk(newMsg, c, &TalkCounter, true)
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
		//Try to forward for at least 100 us.
		go func(){
			select {
			case recvChan <- msg:
			case <- time.After(100 * time.Microsecond):
				sm.Print(fmt.Sprintf("Couldn't forward message: %d", msg.TalkID))
				//This should only happen if an ack message is assumed received
				//but two tcp messages got lost, and the third message is actually
				//received as the getAck times out. Precautinary
				//Helps with fault tolerance in any case
			}
		}()
	}
	return true
}


func endTalk(c *client, id uint32){
	talkTex.Lock()
	delete(c.talks_m, id)
	talks--
	if talks == 0{
		//fmt.Println("No talks active")
	}
	sm.Print(fmt.Sprintf("Talk ended2 %d", id))
	talkTex.Unlock()
}

func runProtocol(msg *Msg_t, talk_c <-chan *Msg_t, c *client, outgoing bool){
	if outgoing {
		switch msg.Type{
		default:
			sendEvt(msg, talk_c, c)
		}

	} else {
		switch msg.Type{
		case PING:
			//nothing
		default:
			recvEvt(msg, talk_c, c)
		}
	}
}

func Ping_out(talkID uint32, c *client){
	msg := Msg_t{TalkID: talkID, Type: PING}
	for{
		select {
		case <- c.dc_c :
			//client dc
			return
		case <- time.After(40 * time.Millisecond) :
			c.send(&msg)
		}
	}
}


func sendEvt(msg *Msg_t, talk_c <-chan *Msg_t, c *client){
	sm.Print(fmt.Sprintf("Talk started send %d", msg.TalkID))
	c.send(msg)
	if getACK(msg, talk_c, c) {
		go sm.EvtAccepted(&msg.Evt, c.smIndex)
	} else {
		go sm.EvtDismissed(&msg.Evt, c.smIndex)
	}
	endTalk(c,msg.TalkID)
}

func recvEvt(msg *Msg_t, talk_c <-chan *Msg_t, c *client){
	sm.Print(fmt.Sprintf("Talk started recv %d", msg.TalkID))
	sm.EvtRegister(&msg.Evt, c.smIndex)
	sendACK(msg, talk_c, c)
	endTalk(c,msg.TalkID)
}


func getACK(msg *Msg_t, talk_c <-chan *Msg_t, c *client) bool {
	missed := false
	for {
		//dc_c and talk_c gets filled by same routine.
		//No messages will be received after dc_c closes
		select {
		case rcvMsg, ok := <- talk_c:
			if !ok {
				return false
			}
			if rcvMsg.Type == ACK {
				return true
			} else {
				sm.Print(fmt.Sprintf("Talk : %d, Received unexpected message: %d", rcvMsg.TalkID, rcvMsg.Type))
			}

		case <- time.After(30 * time.Millisecond) :
			//Ack not received
			if !missed {
				sm.Print(fmt.Sprintf("Ack not received %d, Type: %d, Evt: %d, Floor:%d, Target%d, Button: %d", msg.TalkID, msg.Type, msg.Evt.Type, msg.Evt.Floor, msg.Evt.Target, msg.Evt.Button))
				missed = true
			}
			c.send(msg)
		}
	}
}

func sendACK(msg *Msg_t, talk_c <-chan *Msg_t, c *client) bool {
	//Wait for call to be handled / request for new ack if prev failed
	msg.Type = ACK
	c.send(msg)
	for {
		select {
		case rcvMsg, ok := <- talk_c:
			if !ok {
				return false
			}
			//Ack not received (received duplicate message)
			c.send(msg)
			sm.Print(fmt.Sprintf("Talk : %d, resending Ack", rcvMsg.TalkID))
		case <- time.After(100 * time.Millisecond) :
			//Ack assumed received (or 3 tcp messages lost?)
			sm.Print(fmt.Sprintf("Talk : %d, assuming ack received", rcvMsg.TalkID))
			return true
		}
	}
}

func (c client) send(msg *Msg_t){
	buf := toBytes(msg)
	c.conn.Send(buf)
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