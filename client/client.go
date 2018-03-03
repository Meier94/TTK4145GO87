package client

import (
	"87/statemap"
	"87/encode"
	"87/network"
	"87/print"
	"sync"
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
	dcCh chan bool
	talkDoneCh chan uint32
	msgCh chan []byte
	evtCh chan *sm.Evt
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
	BUFLEN = uint8(encode.Size(Msg_t{}))
	myID = id
	print.StaticVars("Active talks: ", &talks)
}

func testErr(err error, msg string) bool {
	if err != nil {
		//fmt.Println(msg,err)
		return true
	}
	return false
}

func NewClient(conn com.Connection, flag bool){
	msg := Msg_t{ClientID: myID, Type: INTRO}
	status := &msg.Evt
	status.Floor, status.Target, status.Stuck = sm.GetState(0)
	client{conn: conn}.send(&msg)

	in := conn.Read(BUFLEN)
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
	cli.talkDoneCh  = make(chan uint32)
	cli.dcCh 		= make(chan bool)
	cli.evtCh 		= make(chan *sm.Evt, 10)
	cli.msgCh 		= make(chan []byte, 10)
	cli.talks_m 	= make(map[uint32]chan *Msg_t)
	cli.smIndex		= sm.AddNode(cli.id, status.Floor, status.Target, status.Stuck, cli.evtCh)

	print.Format("Added node with id: %d\n", cli.id)
	go routeClient(&cli)
}

func closeClient(c *client){
	print.Format("Connection to %d closed.\n",c.id)

	sm.RemoveNode(c.smIndex)
	talkTex.Lock()
	close(c.talkDoneCh)
	c.talkDoneCh = nil
	close(c.dcCh)


	for _, ch := range c.talks_m {
		close(ch)
	}
	talkTex.Unlock()

	//Remove from communication module
	c.conn.Close()
}

func routeClient(c *client){
	//Unique id for each talk. Talks initiated by dialer are odd numbered and vice versa
	var TalkCounter uint32 = 0
	if myID < c.id {
		TalkCounter++
	}

	go c.conn.Listen(c.msgCh, BUFLEN)
	go pingClient(TalkCounter, c)
	TalkCounter+=2

	defer print.StaticVars("ID: ", &c.id, " TalkCounter: ", &TalkCounter).Remove()

	for {
		select {
			//TcpListen has received a message from client
			case data, ok := <- c.msgCh: {
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

			case evt := <- c.evtCh :
				newMsg := &Msg_t{Type: EVT, Evt: *evt}
				newTalk(newMsg, c, &TalkCounter, true)
		}
	}
}

func newTalk(msg *Msg_t, c *client, counter *uint32, outgoing bool){
	newCh := make(chan *Msg_t)
	talkTex.Lock()

	talks++
	if outgoing {
		msg.TalkID = *counter
		*counter += 2
	}

	c.talks_m[msg.TalkID] = newCh
	talkTex.Unlock()

	if outgoing {
		go sendEvt(msg, newCh, c)

	} else {
		if msg.Type != PING {
			go recvEvt(msg, newCh, c)
		}
	}
}


func notifyTalk(talks_m map[uint32]chan *Msg_t, msg *Msg_t) bool{
	//Ignore ping messages (TalkID <= 1)
	if msg.TalkID > 1 {
		talkTex.Lock()
		recvChan := talks_m[msg.TalkID]
		talkTex.Unlock()

		if recvChan == nil {
			return false
		}
		//Try to forward for *at least* 100us
		go func(){
			select {
			case recvChan <- msg:
			case <- time.After(100 * time.Microsecond):
				print.Format("Couldn't forward message: %d\n", msg.TalkID)
			}
		}()
	}
	return true
}


func endTalk(c *client, id uint32){
	talkTex.Lock()
	delete(c.talks_m, id)
	talks--
	talkTex.Unlock()
}


func pingClient(talkID uint32, c *client){
	msg := Msg_t{TalkID: talkID, Type: PING}
	for{
		select {
		case <- c.dcCh :
			//client dc
			return
		case <- time.After(40 * time.Millisecond) :
			c.send(&msg)
		}
	}
}


func sendEvt(msg *Msg_t, talkCh <-chan *Msg_t, c *client){
	c.send(msg)
	if getACK(msg, talkCh, c) {
		go sm.EvtAccepted(&msg.Evt, c.smIndex)
	} else {
		go sm.EvtDismissed(&msg.Evt, c.smIndex)
	}
	endTalk(c,msg.TalkID)
}


func recvEvt(msg *Msg_t, talkCh <-chan *Msg_t, c *client){
	go sm.EvtRegister(&msg.Evt, c.smIndex)
	sendACK(msg, talkCh, c)
	endTalk(c,msg.TalkID)
}


func getACK(msg *Msg_t, talkCh <-chan *Msg_t, c *client) bool {
	attempts := 0
	for {
		select {
		case rcvMsg, ok := <- talkCh:
			if !ok {
				//Client closed
				return false
			}
			if rcvMsg.Type == ACK {
				return true
			} else {
				print.Format("Talk : %d, received unexpected message\n", rcvMsg.TalkID)
			}

		case <- time.After(30 * time.Millisecond) :
			//Ack not received
			print.Format("Talk: %d, ack not received\n", msg.TalkID)
			if attempts == 3 {
				//Failed to receive ack 3 times, try to recover
				return false
			}
			attempts++
			c.send(msg)
		}
	}
}

func sendACK(msg *Msg_t, talkCh <-chan *Msg_t, c *client) bool {
	msg.Type = ACK
	c.send(msg)

	//Listen for further messages (in case the ack got lost)
	for {
		select {
		case rcvMsg, ok := <- talkCh:
			if !ok {
				//Client closed
				return false
			}
			//Ack not received (received duplicate message)
			c.send(msg)
			print.Format("Talk : %d, resending Ack\n", rcvMsg.TalkID)
		case <- time.After(100 * time.Millisecond) :
			//Ack assumed received (or 3-4 tcp messages lost?)
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
	encode.FromBytes(data, &msg)
	return &msg
}


func toBytes(data *Msg_t) []byte{
	return encode.ToBytes(*data)
}