package tcpmod

import (
	"net"
	"fmt"
//	"bufio"
//	"os"
//	"strings"
	"time"
	"bytes"
)


func Tcp_client(){
	fmt.Println("Launching server test...")

	// listen on all interfaces
	ln, _ := net.Listen("tcp", ":4487")

	// accept connection on port
	conn, _ := ln.Accept()
	fmt.Printf("Connected to %q\n", conn.RemoteAddr())
	// run loop forever (or until ctrl-c)
	count := 0
	for {
		// will listen for message to process ending in newline (\n)
		buf := []byte{'g','o','l','a','n','g'}
		buf2 := []byte{'g','o','l','a','n','g'}
		conn.Write([]byte(buf))
		for {
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, err := conn.Read([]byte(buf))
			if err != nil {
				fmt.Printf("Read fail: %d,%s, count: %d\n", n, err, count)
				return
			}
			if bytes.Compare(buf,buf2) != 0 {
				fmt.Printf("Byte arrays different\n")
				return;
			}
			count++
			break
		}
	}
}



//Klarer å sende ~215k meldinger frem og tilbake med annen maskin på 30 sek (139 us pr meldingsutveksling (mld + ack))
func Tcp_server(){
	// connect to this socket
	var err error
	var conn net.Conn

	//loops infinitely until it manages to connect
	for {
		conn, err = net.Dial("tcp", "129.241.187.152:4487")
		if err == nil {
			break
		}
		//fmt.Println(err)
		time.Sleep(100 * time.Millisecond)
	}
	count := 0
	for {
		buf := []byte{'g','o','l','a','n','g'}
		buf2 := []byte{'g','o','l','a','n','g'}
		for {
			conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
			n, err := conn.Read([]byte(buf))
			if err != nil {
				fmt.Printf("Read fail: %d,%s, count: %d\n", n, err, count)
				return
			}
			if bytes.Compare(buf,buf2) != 0 {
				fmt.Printf("Byte arrays different\n")
				return
			}
			count++
			break
		}
		conn.Write([]byte(buf))
	}
}



/*
import (
    "encoding/base64"
    "encoding/gob"
    "bytes"
)

type SX map[string]interface{}

// go binary encoder
func ToGOB64(m SX) string {
    b := bytes.Buffer{}
    e := gob.NewEncoder(&b)
    err := e.Encode(m)
    if err != nil { fmt.Println(`failed gob Encode`, err) }
    return base64.StdEncoding.EncodeToString(b.Bytes())
}

// go binary decoder
func FromGOB64(str string) SX {
    m := SX{}
    by, err := base64.StdEncoding.DecodeString(str)
    if err != nil { fmt.Println(`failed base64 Decode`, err); }
    b := bytes.Buffer{}
    b.Write(by)
    d := gob.NewDecoder(&b)
    err = d.Decode(&m)
    if err != nil { fmt.Println(`failed gob Decode`, err); }
    return m
}*/