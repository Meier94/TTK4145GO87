package com

import (
	"net"
	"fmt"
	"time"
	"sync"
)



var connections_m map[string]int
var mapTex *sync.Mutex
var myID uint8
var connectionCallback func(net.Conn, bool)

func Start(id uint8, callback func(net.Conn, bool)){
	connectionCallback = callback
	mapTex = &sync.Mutex{}
	connections_m = make(map[string]int)
	myID = id
}

func testErr(err error, msg string) bool {
	if err != nil {
		fmt.Println(msg,err)
		return true
	}
	return false
}

func Send(msg []byte, conn net.Conn){
	_, err := conn.Write(msg)
	if testErr(err, "") {
		panic(err)
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

func Close(conn net.Conn){
	ip,_,_ := net.SplitHostPort(conn.RemoteAddr().String())
	removeFromMap(ip)
	conn.Close()
}

func removeFromMap(ip string){
	mapTex.Lock()
	delete(connections_m, ip)
	mapTex.Unlock()
}


func TcpListen(conn net.Conn, msg_c chan<- []byte, bufLen uint8){
	for {
		buf := make([]byte, bufLen)
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil || n != int(bufLen){

			if err, ok := err.(net.Error); ok && err.Timeout() {
        		fmt.Printf("Timeout caused disconnect\n")
    		}
			close(msg_c)
			return
		}
		//Translate into msg format
		msg_c <- buf
	}
}

func TcpRead(conn net.Conn, bufLen uint8) []byte{
	buf := make([]byte, bufLen)
	conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
	n, err := conn.Read(buf)
	if err != nil || n != int(bufLen){

		if err, ok := err.(net.Error); ok && err.Timeout() {
    		fmt.Printf("Tcp read timed out\n")
		} else {
			fmt.Printf("Tcp read failed\n")
		}
		return nil
	}
	return buf
}


//Legg inn i for loop dersom starten failer
func UdpListen(){

	Addr,err := net.ResolveUDPAddr("udp",":55087")
    testErr(err, "Couldn't resolve UDP listen")

	SerConn, err := net.ListenUDP("udp", Addr)
    testErr(err, "Couldn't listen to udp")

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
		go connectionCallback(conn, true)
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
			go connectionCallback(conn, false)
		}
	}
}
