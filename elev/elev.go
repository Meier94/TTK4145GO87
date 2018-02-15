package elev
/*
#include "io.h"
#cgo LDFLAGS: -L . -lcomedi
*/
import "C"

import (
	"time"
	"fmt"
)

const UP C.int = 0
const DOWN C.int = 1
const CAB C.int = 2
const STOP C.int = 3
var types [4]string = [4]string{"Up", "Down", "Cab", "Arrival"}

func Init(){
	var i C.int = C.io_init()
	i++
	C.clear_all_lights()
	C.set_motor(STOP)
	go triggerEvents()
}


func getInputs() bool{
	if C.get_signals() == C.int(1){
		return true
	}
	return false
}

func getEvent() (uint16,uint16) {
	var evt uint16 = uint16(C.getEvent())
	return (evt >> 8), evt & 0xFF
}


func triggerEvents(){
	for {

		if getInputs() {
			for {
				floor, evtType := getEvent()
				if(floor < 1){
					break
				}
				fmt.Printf("Event: %s, floor: %d\n",types[evtType],floor)
			}
		}
		time.Sleep(10*time.Millisecond)
	}
}
