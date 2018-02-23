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

Burde close hver talk_c istedenfor dc_c:
	Pros:
	En mindre entry i client
	En mindre entry i select for get/sendAck
	Mindre kompleksitet?

	Cons: (or not really)
	Må lukke channels en etter en mens de har timeout som:
		- Antar at ack er received (no prob)
		- Prøver å resend melding til closed socket (kan ha sjekk for dette though)
			- Kan lukke socket til sist? Don't like this
			- Kan lage id system i communication.go 
				- gjør den litt mindre generell, men id trenger ikke stemme overens med heisID.
				- Gjør communication litt mer selvstendig da client ikke trenger import net
				- id burde though stemme overens med heisID for å slippe å bruke map.
					- Hva skjer hvis connection closes og ny heis får samme id:
						- Et send til en closed client vil gå til ny client
						- Kan ha en key som må stemme for å sende (feks heisID)
							- liker ikke dette helt
		<--------------------
		- kan ha en key som autogenereres (increment) og legge inn i type:
		type connection struct{
			index //i lista over connections
			conn net.Conn
			key uint32
		}
		//burde være umulig å sende til closed connection 
		send(connection, buf){
			if connection.index < numConnections {

				if connList[connection.index].key == connection.key {
					connection.conn.Write(buf)
				}
			}
		}
		Dette vil ikke funke med dynamiske indexer though (som trengs)
			- kan ha voids i lista men prøve å fylle den fra start
			- holde track av voids og fylle inn de ved nye tilkoblinger
			- ikke superbra med mutex for connList access

		Alternativt:
			Sjekke errorverdi til send :P


		Kan ha en datatype connection:
		type connection struct {
			mutex *sync.Mutex
			closed bool
			conn net.Conn
		}

		Er det noe poeng å ha mutex og sjekke closedness dersom man ignorerer det uansett?

		Kan skreddersy com driveren til å funke med 256 noder (eller some other number (dyn array?)) Hvor stress er det egentlig å allokere 1mb minne (100k spots)

