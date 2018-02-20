package elev
/*
#include "io.h"
#cgo LDFLAGS: -L . -lcomedi
*/
import "C"

import (
	"../statemap"
	"time"
	"fmt"
)

const UP uint8 = 0
const DOWN uint8 = 1
const CAB uint = 2
const STOP uint8 = 3
const FLOOR uint8 = 3

const idle_s = 0
const init_s = 1
const open_s = 2
const stuck_s = 3
const executing_s = 4

var types [4]string = [4]string{"Up", "Down", "Cab", "Arrival"}

var state int = init_s

const NONE int16 = int16(-1)
var currentFloor int16 = NONE
var currentTarget int16 = NONE



func Init(id uint8){
	io_init()
	clear_all_lights()
	set_motor(UP)
	sm.ElevMapUpdate = MapUpdate
	sm.Init(id)
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

func getEvent() (int16, uint8) {
	var evt uint16 = uint16(C.getEvent())
	return int16(evt >> 8), uint8(evt & 0xFF)
}

func set_button_light(floor int16, buttonType uint8, value int){
	C.set_button_light(C.int(floor), C.int(buttonType), C.int(value))
}

func set_floor_light(floor int16){
	C.set_floor_light(C.int(floor))
}

func clear_all_lights(){
	C.clear_all_lights()
}

func set_motor(dir uint8){
	C.set_motor(C.int(dir))
}

func setDoorLight(value int){
	C.set_door_light(C.int(value))
}


func evtButtonPressed(floor int16, buttonType uint8){
	sm.DelegateButtonPress(floor, buttonType)
}

func openDoor(){
	
	state = open_s
	println("state: ",state)
	setDoorLight(1)
	time.AfterFunc(time.Second*3, timeout)
	set_motor(STOP)
}

func timeout(){
	setDoorLight(0)
	if currentTarget != NONE && state != init_s{
		println(currentTarget)
		dir := UP
		if currentFloor > currentTarget {
			dir = DOWN
		}
		set_motor(dir)
		
		state = executing_s
		println("state: ",state)
		return
	}
	
	state = idle_s
	println("state: ",state)
}



func evtFloorReached(floor int16){
	currentFloor = floor
	target := sm.StatusUpdate(floor, false)

	switch state {
	case open_s:
	case idle_s:
	default:
		if currentTarget == floor{
			openDoor()
			currentTarget = target
			return
		}
		if currentTarget == NONE {
			if target == NONE {
				println("test")
				set_motor(STOP)
				
				state = idle_s
				println("state: ",state)
				return
			}
			currentTarget = target
			
			state = executing_s
			println("state: ",state)
		}
		return
	}
	fmt.Println("Floor reached in wrong state")
}

func MapUpdate(floor int16){
	if currentTarget == NONE {
		currentTarget = floor
	}
	if state == idle_s {
		if currentFloor == floor {
			openDoor()
			currentTarget = sm.StatusUpdate(floor, false)
			return
		}
		dir := UP
		if currentFloor > currentTarget {
			dir = DOWN
		}
		println("mapupdate going")
		set_motor(dir)
		
		state = executing_s
		println("state: ",state)
	}
}

func triggerEvents(){
	for {
		if getInputs() {
			for {
				floor, evtType := getEvent()
				if(evtType > 3){
					break
				}
				fmt.Printf("Event: %s, floor: %d\n",types[evtType],floor)
				if evtType == FLOOR {
					println("arrival")
					evtFloorReached(floor)
					continue
				}
				evtButtonPressed(floor, evtType)
			}
		}
		time.Sleep(10*time.Millisecond)
	}
}
