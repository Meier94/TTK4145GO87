package client

import (
	"xx/encode"
	"xx/network"
	"xx/print"
	"xx/statemap"
	"sync"
	"time"
)

//Struct fields must be public for marshalling
type Msg_t struct {
	TalkID   uint32
	ClientID uint8
	Type     uint8
	Evt      sm.Evt
}

type client struct {
	id         uint8						//Client ID
	smIndex    *int16						//Clients index in statemap (not constant)
	conn       com.Connection				//Network handle
	dcCh       chan bool					//Channel to notify goroutines of dc
	msgCh      chan []byte 					//Channel for incoming messages
	evtCh      chan *sm.Evt 				//Channel for outgoing events from statemap
	talks_m    map[uint32]chan *Msg_t		//Map of active talks for the client
	mapTex	   *sync.Mutex					//Mutex for the talk map
}

//flags
//Message Types
const ACK uint8 = 200
const PING uint8 = 201
const INTRO uint8 = 202
const EVT uint8 = 203

var BUFLEN uint8							
var talks uint32 = 0						//Active talk counter for debugging
var myID uint8

func Init(id uint8) {
	BUFLEN = uint8(encode.Size(Msg_t{}))
	myID = id
	print.StaticVars("Active talks: ", &talks)
}

func testErr(err error, msg string) bool {
	if err != nil {
		return true
	}
	return false
}

func NewClient(conn com.Connection, flag bool) {
	msg := Msg_t{ClientID: myID, Type: INTRO}
	status := &msg.Evt
	status.Floor, status.Target, status.Stuck = sm.GetState(0)
	client{conn: conn}.send(&msg)

	rcvData := conn.Read(BUFLEN)
	if rcvData == nil {
		conn.Close()
		return
	}
	introMsg := toMsg(rcvData)
	if introMsg.Type != INTRO {
		conn.Close()
		return
	}
	status = &introMsg.Evt

	var cli client
	cli.id = introMsg.ClientID
	cli.conn = conn
	cli.dcCh = make(chan bool)
	cli.evtCh = make(chan *sm.Evt, 10)
	cli.msgCh = make(chan []byte, 10)
	cli.talks_m = make(map[uint32]chan *Msg_t)
	cli.mapTex = &sync.Mutex{}
	cli.smIndex = sm.AddNode(cli.id, status.Floor, status.Target, status.Stuck, cli.evtCh)

	print.Format("Added node with id: %d\n", cli.id)
	go routeClient(&cli)
}

func closeClient(c *client) {
	print.Format("Connection to %d closed.\n", c.id)

	sm.RemoveNode(c.smIndex)
	c.mapTex.Lock()
	close(c.dcCh)

	for _, ch := range c.talks_m {
		close(ch)
	}
	c.mapTex.Unlock()

	//Remove from communication module
	c.conn.Close()
}

//Goroutine that routes messages to/from a client
func routeClient(c *client) {
	//Unique id for each talk. Talks initiated by dialer are odd numbered and vice versa
	var TalkCounter uint32 = 0
	if myID < c.id {
		TalkCounter++
	}

	go c.conn.Listen(c.msgCh, BUFLEN)
	go pingClient(TalkCounter, c)
	TalkCounter += 2

	defer print.StaticVars("ID: ", &c.id, " TalkCounter: ", &TalkCounter).Remove()

	for {
		select {
		//TcpListen has received a message from client
		case data, ok := <-c.msgCh:
			{
				if !ok {
					//Client is non responsive
					closeClient(c)
					return
				}
				newMsg := toMsg(data)
				//Notify correct protocol
				if !notifyTalk(c, newMsg) {
					newTalk(newMsg, c, nil, false)
				}
			}

		case evt := <-c.evtCh:
			newMsg := &Msg_t{Type: EVT, Evt: *evt}
			newTalk(newMsg, c, &TalkCounter, true)
		}
	}
}

func newTalk(msg *Msg_t, c *client, counter *uint32, outgoing bool) {
	newCh := make(chan *Msg_t)
	c.mapTex.Lock()

	talks++
	if outgoing {
		msg.TalkID = *counter
		*counter += 2
	}

	c.talks_m[msg.TalkID] = newCh
	c.mapTex.Unlock()

	if outgoing {
		go sendEvt(msg, newCh, c)

	} else {
		if msg.Type != PING {
			go recvEvt(msg, newCh, c)
		}
	}
}

func notifyTalk(c *client, msg *Msg_t) bool {
	//Ignore ping messages (TalkID <= 1)
	if msg.TalkID > 1 {
		c.mapTex.Lock()
		recvChan := c.talks_m[msg.TalkID]
		c.mapTex.Unlock()

		if recvChan == nil {
			return false
		}
		//Try to forward for *at least* 100us
		go func() {
			select {
			case recvChan <- msg:
			case <-time.After(100 * time.Microsecond):
				print.Format("Couldn't forward message: %d\n", msg.TalkID)
			}
		}()
	}
	return true
}

func endTalk(c *client, id uint32) {
	c.mapTex.Lock()
	delete(c.talks_m, id)
	talks--
	c.mapTex.Unlock()
}

func pingClient(talkID uint32, c *client) {
	msg := Msg_t{TalkID: talkID, Type: PING}
	for {
		select {
		case <-c.dcCh:
			//client dc
			return
		case <-time.After(20 * time.Millisecond):
			c.send(&msg)
		}
	}
}

func sendEvt(msg *Msg_t, talkCh <-chan *Msg_t, c *client) {
	c.send(msg)
	if getACK(msg, talkCh, c) {
		go sm.EvtAccepted(&msg.Evt, c.smIndex)
	} else {
		go sm.EvtDismissed(&msg.Evt, c.smIndex)
	}
	endTalk(c, msg.TalkID)
}

func recvEvt(msg *Msg_t, talkCh <-chan *Msg_t, c *client) {
	go sm.EvtRegister(&msg.Evt, c.smIndex)
	sendACK(msg, talkCh, c)
	endTalk(c, msg.TalkID)
}

func getACK(msg *Msg_t, talkCh <-chan *Msg_t, c *client) bool {
	attempts := 0
	for {
		select {
		case rcvMsg, ok := <-talkCh:
			if !ok {
				//Client closed
				return false
			}
			if rcvMsg.Type == ACK {
				return true
			} else {
				print.Format("Talk : %d, received unexpected message\n", rcvMsg.TalkID)
			}

		case <-time.After(30 * time.Millisecond):
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
		case rcvMsg, ok := <-talkCh:
			if !ok {
				//Client closed
				return false
			}
			//Ack not received (received duplicate message)
			c.send(msg)
			print.Format("Talk : %d, resending Ack\n", rcvMsg.TalkID)
		case <-time.After(100 * time.Millisecond):
			//Ack assumed received (or 3-4 tcp messages lost?)
			return true
		}
	}
}

func (c client) send(msg *Msg_t) {
	buf := toBytes(msg)
	c.conn.Send(buf)
}

func toMsg(data []byte) *Msg_t {
	var msg Msg_t
	encode.FromBytes(data, &msg)
	return &msg
}

func toBytes(data *Msg_t) []byte {
	return encode.ToBytes(*data)
}
