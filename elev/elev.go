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

const UP int = 0
const DOWN int = 1
const CAB int = 2
const STOP int = 3

const idle_s = 0
const init_s = 1
const open_s = 2
const stuck_s = 3
const executing_s = 4

var types [4]string = [4]string{"Up", "Down", "Cab", "Arrival"}

var currentFloor int = 0
var state int = init_s



func Init(){
	io_init()
	clear_all_lights()
	set_motor(STOP)
	go triggerEvents()
}

func io_init() bool{
	var i C.int = C.io_init()
	if i < 0 {
		return false
	}
	return true
}

func getInputs() bool{
	if C.get_signals() == C.int(1){
		return true
	}
	return false
}

func getEvent() (uint8,uint8) {
	var evt uint16 = uint16(C.getEvent())
	return uint8(evt >> 8), uint8(evt & 0xFF)
}

func set_button_light(floor uint8, buttonType uint8, value int){
	C.set_button_light(C.int(floor), C.int(buttonType), C.int(value))
}

func set_floor_light(floor uint8){
	C.set_floor_light(C.int(floor))
}

func clear_all_lights(){
	C.clear_all_lights()
}

func set_motor(dir uint8){
	C.set_motor(C.int(dir))
}


func evtButtonPressed(buttonType uint8, floor uint8){
	if Standalone()
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
