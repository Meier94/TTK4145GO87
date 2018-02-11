package tcpmod

import (
	"net"
	"fmt"
//	"bufio"
//	"os"
//	"strings"
	"time"
//	"io"
	"bytes"
	"encoding/binary"
)

//felt må ha stor forbokstav for å kunne bli konvertert fra []byte
type msg_t struct{
	TalkId uint32	//talk-id
	MsgId uint8
	Type uint8
	Data uint16 //is allowed (Y) for serialization
}

type client struct{
	conn net.Conn
	dc_c chan bool
	talkDone_c chan uint32	
}



func ClientListen(c *client){
//	num_rcvd := 0
//	num_sent := 0

	msg_c := make(chan *msg_t)
	talks_m := make(map[uint32]chan *msg_t)

	c.talkDone_c = make(chan uint32)
	c.dc_c = make(chan bool)

	//Tcp receiver, passes incoming messages to msg_c
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

					//Distribute his orders (should maybe be done in the talks?)

					//issue that it returns before talks are finished cleaning up?
					return
				}
				//Notify correct protocol
				if !notifyTalk(talks_m, newMsg){
					new_c := make(chan *msg_t)
					go runProtocol(newMsg, new_c, c)
					talks_m[newMsg.TalkId] = new_c
				}
			}
			case id, ok := <- c.talkDone_c:{
				if !ok {
					fmt.Printf("TalkDone channel closed for some odd reason\n")
				}
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
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
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


//placeholder for compilation
func addOrder(data uint16) chan bool{
	data = 0
	c := make(chan bool)
	return c
}
func removeOrder() chan uint16{
	return make(chan uint16)
}

//flags
//Message Types
const COMPLETE uint8 = 200
const ACK uint8 = 201
const HANDLE_CALL uint8 = 202
const FAILED uint8 = 203
const BUFLEN uint8 = 8

func runProtocol(msg *msg_t, talk_c <-chan *msg_t, c *client){
	switch msg.Type{
		case HANDLE_CALL:{
			HandleCall(msg, talk_c, c)
			//Legg det over i en egen funksjon prob

		}
	}
}

//Ikke fornøyd med denne måten å gjøre ting på
//Vil helst at man skal anta at tcp meldinger går gjennom og heller sørge for ack med timeout
func HandleCall(msg *msg_t, talk_c <-chan *msg_t, c *client){
	//Keep track of unique msgs sent/rcvd
	rcvMsgId := msg.MsgId
	var sndMsgId uint8 = 0

	//Add order to map
	callStatus_c := addOrder(msg.Data)
	call := msg.Data
	//Send ack
	*msg = msg_t{msg.TalkId, sndMsgId, ACK, 0}
	send(msg, c)

	//Wait for call to be handled / request for new ack if prev failed
	for {
		select {
		case status := <- callStatus_c:
			if status {
				*msg = msg_t{msg.TalkId, sndMsgId, COMPLETE, call}
				sndMsgId++
				send(msg, c)
				//Channel gets closed in other end, therefore omit it from select
				callStatus_c = nil
			} else {
				//Call couldn't be completed
				*msg = msg_t{msg.TalkId, sndMsgId, FAILED, call}
				sndMsgId++
				send(msg, c)
			}
			//TODO: wait for ack
		
		case rcvMsg := <- talk_c: 
			if rcvMsg.MsgId == rcvMsgId {
				//Ack not received
				send(msg, c)
			} else if rcvMsg.Type == ACK {
				removeOrder() <- msg.Data
				return
			}
		case <- c.dc_c :
			//client dc
			//TODO - redistribute order
			return
		}
	}
}