package io

/*
#include "io.h"
#cgo LDFLAGS: -L . -lcomedi
*/
import "C"


func Init() bool{
	var i C.int = C.io_init()
	if i < 0 {
		return false
	}
	return true
}

func GetInputs() bool{
	if C.get_signals() == C.int(1){
		return true
	}
	return false
}

func GetEvent() (int16, uint8) {
	var evt uint16 = uint16(C.getEvent())
	return int16(evt >> 8), uint8(evt & 0xFF)
}

func SetButtonLight(floor int16, buttonType uint8, value int){
	C.set_button_light(C.int(floor), C.int(buttonType), C.int(value))
}

func SetFloorLight(floor int16){
	C.set_floor_light(C.int(floor))
}

func ClearAllLights(){
	C.clear_all_lights()
}

func SetMotor(dir uint8){
	C.set_motor(C.int(dir))
}

func SetDoorLight(value int){
	C.set_door_light(C.int(value))
}