package client

import (
	"bytes"
	"encoding/binary"
	"../statemap"
	"../network"
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
		fmt.Println(msg,err)
		return true
	}
	return false
}

func ClientInit(conn com.Connection, flag bool){
	msg := Msg_t{ClientID: myID, Type: INTRO}
	status := &msg.Evt
	status.Floor, status.Target, status.Stuck = sm.GetState(0)
	client{conn: conn}.send(&msg)

	intro := toMsg(conn.TcpRead(BUFLEN))
	if intro == nil || intro.Type != INTRO {
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
	cli.msg_c 		= make(chan []byte)
	cli.talks_m 	= make(map[uint32]chan *Msg_t)
	cli.smIndex		= sm.AddNode(cli.id, status.Floor, status.Target, status.Stuck, cli.evt_c)

	fmt.Printf("Node added, Floor: %d, Target: %d, Stuck: %t, ID: %d\n", status.Floor, status.Target, status.Stuck, cli.id)

	go ClientListen(&cli)
}

func closeClient(c *client){
	fmt.Printf("Connection to %d closed.\n",c.id)

	sm.RemoveNode(c.smIndex)
	talkTex.Lock()
	close(c.talkDone_c)
	c.talkDone_c = nil
	talkTex.Unlock()
	close(c.dc_c)


	for _, ch := range c.talks_m {
		close(ch)
	}
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

			//Needs to be exclusive from closeClient 
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
		//Try to forward for 5 ms.
		select {
		case recvChan <- msg:
		case <- time.After(5 * time.Millisecond):
			fmt.Println("Couldn't forward message")
			//This should only happen if an ack message is assumed received
			//but two tcp messages got lost, and the third message is actually
			//received as the getAck times out. Precautinary
			//Helps with fault tolerance in any case
		}
	}
	return true
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
		case <- time.After(30 * time.Millisecond) :
			c.send(&msg)
		}
	}
}


func sendEvt(msg *Msg_t, talk_c <-chan *Msg_t, c *client){

	c.send(msg)
	if getACK(msg, talk_c, c) {
		sm.EvtAccepted(&msg.Evt, c.smIndex)
	} else {
		sm.EvtDismissed(&msg.Evt, c.smIndex)
	}
	endTalk(c,msg.TalkID)
}

func recvEvt(msg *Msg_t, talk_c <-chan *Msg_t, c *client){
	sm.EvtRegister(&msg.Evt, c.smIndex)
	sendACK(msg, talk_c, c)
	endTalk(c,msg.TalkID)
}


func getACK(msg *Msg_t, talk_c <-chan *Msg_t, c *client) bool {
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
				fmt.Printf("Talk : %d, Received unexpected message: %d\n", rcvMsg.TalkID, rcvMsg.Type)
			}

		case <- time.After(40 * time.Millisecond) :
			//Ack not received
			fmt.Printf("Ack not received\n")
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
			//Ack not received
			c.send(msg)
			fmt.Printf("Talk : %d, Resending: %d\n", rcvMsg.TalkID, rcvMsg.Type)
		case <- time.After(1000 * time.Millisecond) :
			//Ack assumed received (or 25 tcp messages lost?)
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