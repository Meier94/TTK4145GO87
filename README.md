# TTK4145GO87
Go heisprosjekt

https://blog.golang.org/c-go-cgo
https://github.com/TTK4145/Project

Working convert:
package main

import (
	"fmt"
	"encoding/binary"
	"bytes"
)

type myStruct struct {
	ID   int32
	Data int32
	B [10]byte
}

func main() {
	var bin_buf bytes.Buffer
	
	x := myStruct{1, 1, [10]byte{}}
	binary.Write(&bin_buf, binary.BigEndian, x)
	fmt.Printf("%q\n", bin_buf.Bytes())
	
	
	
	buf := bytes.NewReader(bin_buf.Bytes())
	var x2 myStruct
	err := binary.Read(buf, binary.BigEndian, &x2)
	if err != nil {
		panic(err)
	}
	if(x2.ID != x.ID){
		fmt.Printf("FAil\n")
	}

	bin_buf.Reset()
	binary.Write(&bin_buf, binary.BigEndian, x2)
	fmt.Printf("%q\n", bin_buf.Bytes())
}

Thoughts:
	Hva skjer dersom du er stuck med ordre og du kobler deg til din første node

	Hva hvis du får et kall fra en etasje og du allerede vet at ordren oppfylles

	Flytt C. funksjonene til io.go

	Er i 3. trykker ned 2. trykker opp 1. Heis stopper i 2. Dette intended?

	Hva hvis døra er åpen og du får et kall i samme etasje i riktig retning
