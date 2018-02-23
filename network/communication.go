package com

import (
	"net"
	"fmt"
	"time"
	"sync"
	"../statemap"
)

type Connection struct{
	conn net.Conn
}

var connections_m map[string]int
var mapTex *sync.Mutex
var myID uint8
var connectionCallback func(Connection, bool)

func Start(id uint8, callback func(Connection, bool)){
	connectionCallback = callback
	mapTex = &sync.Mutex{}
	connections_m = make(map[string]int)
	myID = id

	go UdpListen()
	go TcpAccept()
	go UdpBroadcast()
}

func testErr(err error, msg string) bool {
	if err != nil {
		sm.Print(fmt.Sprintf("%v, %v", msg,err))
		return true
	}
	return false
}

func (c Connection) Send(msg []byte){
	_, err := c.conn.Write(msg)
	testErr(err, "Send")
}


func addToMap(ip string) bool {
	mapTex.Lock()
	defer mapTex.Unlock()
	//Test if already in map
	if _, ok := connections_m[ip]; ok {
		return false
	}
	connections_m[ip] = 1
	return true
}

func (c Connection) Close(){
	ip,_,_ := net.SplitHostPort(c.conn.RemoteAddr().String())
	removeFromMap(ip)
	c.conn.Close()
}

func removeFromMap(ip string){
	mapTex.Lock()
	delete(connections_m, ip)
	mapTex.Unlock()
}


func (c Connection) Listen(msg_c chan<- []byte, bufLen uint8){
	for {
		buf := make([]byte, bufLen)
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil || n != int(bufLen){

			if err, ok := err.(net.Error); ok && err.Timeout() {
        		sm.Print(fmt.Sprintf("Timeout caused disconnect"))
    		}
			close(msg_c)
			return
		}
		//Translate into msg format
		msg_c <- buf
	}
}

func (c Connection) TcpRead(bufLen uint8) []byte{
	buf := make([]byte, bufLen)
	c.conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
	n, err := c.conn.Read(buf)
	if err != nil || n != int(bufLen){

		if err, ok := err.(net.Error); ok && err.Timeout() {
    		sm.Print(fmt.Sprintf("Tcp read timed out"))
		} else {
			sm.Print(fmt.Sprintf("Tcp read failed"))
		}
		return nil
	}
	return buf
}


func UdpListen(){
	for {
		Addr,err := net.ResolveUDPAddr("udp",":55087")
	    if testErr(err, "Couldn't resolve UDP listen") {
	    	continue
	    }

		SerConn, err := net.ListenUDP("udp", Addr)
	    if testErr(err, "Couldn't listen to udp") {
	    	continue
	    }

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

			sm.Print(fmt.Sprintf("Connection established, id: %d", buf[0]))
			go connectionCallback(Connection{conn}, true)
		}
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
			
			sm.Print(fmt.Sprintf("Connected to %s", conn.RemoteAddr()))
			go connectionCallback(Connection{conn}, false)
		}
	}
}
