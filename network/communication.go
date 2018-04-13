package com

import (
	"87/print"
	"net"
	"sync"
	"time"
)

type Connection struct {
	conn net.Conn
}

var connections_m map[string]int
var mapTex *sync.Mutex
var myID uint8
var connectionCallback func(Connection, bool)

func Start(id uint8, callback func(Connection, bool)) {
	connectionCallback = callback
	mapTex = &sync.Mutex{}
	connections_m = make(map[string]int)
	myID = id

	go udpListenForNodes()
	go tcpAcceptConnections()
	go udpBroadcastExistence()
}

func testErr(err error, msg string) bool {
	if err != nil {
		print.Format("%v, %v\n", msg, err)
		return true
	}
	return false
}

//No error test here because it is expected to fail repeatedly when disconnected
func (c Connection) Send(msg []byte) {
	c.conn.Write(msg)
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

func removeFromMap(ip string) {
	mapTex.Lock()
	delete(connections_m, ip)
	mapTex.Unlock()
}

func (c Connection) Close() {
	ip, _, _ := net.SplitHostPort(c.conn.RemoteAddr().String())
	removeFromMap(ip)
	c.conn.Close()
}

func (c Connection) Listen(msg_c chan<- []byte, bufLen uint8) {
	for {
		buf := make([]byte, bufLen)
		c.conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil || n != int(bufLen) {

			if err, ok := err.(net.Error); ok && err.Timeout() {
				print.Format("Timeout caused disconnect\n")
			}
			close(msg_c)
			return
		}
		//Translate into msg format
		msg_c <- buf
	}
}

func (c Connection) Read(bufLen uint8) []byte {
	buf := make([]byte, bufLen)
	c.conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
	n, err := c.conn.Read(buf)
	if err != nil || n != int(bufLen) {

		if err, ok := err.(net.Error); ok && err.Timeout() {
			print.Format("Tcp read timed out\n")
		} else {
			print.Format("Tcp read failed\n")
		}
		return nil
	}
	return buf
}

func udpListenForNodes() {
	for {
		Addr, err := net.ResolveUDPAddr("udp", ":55087")
		if testErr(err, "Couldn't resolve UDP listen") {
			continue
		}

		ListenConn, err := net.ListenUDP("udp", Addr)
		if testErr(err, "Couldn't listen to udp") {
			continue
		}

		buf := make([]byte, 1024)
		defer ListenConn.Close()
		for {
			_, addr, err := ListenConn.ReadFromUDP(buf)

			//Only connect if id of sender is higher
			if testErr(err, "UDP read failed") || buf[0] >= myID {
				continue
			}

			ip, _, _ := net.SplitHostPort(addr.String())
			if !addToMap(ip) {
				continue
			}

			var conn net.Conn
			//Try to dial 3 times
			for i := 0; i < 3; i++ {
				conn, err = net.Dial("tcp", ip+":4487")

				if !testErr(err, "TCP dial failed") {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			if testErr(err, "TCP dial couldn't reach client. Removing from map") {
				removeFromMap(ip)
				continue
			}

			print.Format("Connection established, id: %d\n", buf[0])
			go connectionCallback(Connection{conn}, true)
		}
	}
}

func udpBroadcastExistence() {
	for {
		Addr, err := net.ResolveUDPAddr("udp", "255.255.255.255:55087")
		if testErr(err, "Couldn't resolve UDPAddr broadcast") {
			continue
		}

		conn, err := net.DialUDP("udp", nil, Addr)
		if testErr(err, "Couldn't establish UDP connection") {
			continue
		}

		defer conn.Close()
		buf := []byte{myID}
		for {
			_, err := conn.Write(buf)
			testErr(err, "UDP write failed")

			time.Sleep(time.Second * 1)
		}
	}
}

func tcpAcceptConnections() {
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
			ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
			if !addToMap(ip) {
				continue
			}

			print.Format("Connected to %s\n", conn.RemoteAddr())
			go connectionCallback(Connection{conn}, false)
		}
	}
}
