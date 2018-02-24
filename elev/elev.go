package elev

import (
	"87/elev/io"
	"87/statemap"
	"time"
	"sync"
	"fmt"
)

const m int16 = 4

//Event types
const UP uint8 = 0
const DOWN uint8 = 1
const CAB uint8 = 2
const STOP uint8 = 3
const FLOOR uint8 = 3

const NONE int16 = int16(-1)

//States
const idle_s = 0
const init_s = 1
const open_s = 2
const stuck_s = 3
const executing_s = 4

//Print helper
var types [4]string = [4]string{"Up", "Down", "Cab", "Arrival"}

//variables
var state int = init_s
var orders[m][3] bool
var timer time.Timer



func Init(id uint8) bool {
	if !io.Init(){
		return false
	}
	io.ClearAllLights()
	sm.AddFunction(evtExternalInput)

	timer = time.NewTimer(0 * time.Millisecond)
	<- timer.C


	for i := 0 ; i < int(m) * 3; i++{
		orders[i / 3][i % 3] = false
	}

	sm.Init(id)

	io.SetMotor(UP)
	go triggerEvents()
	return true
}


func triggerEvents(){
	var Floor int16 = NONE
	var Target int16 = NONE
	var Dir int16 = UP
	var doorOpen = false

	for {
		if io.GetInputs() {
			for {
				floor, evtType := io.GetEvent()
				if(evtType > 3){
					break
				}
				////fmt.Printf("Event: %s, floor: %d\n",types[evtType],floor)
				if evtType == FLOOR {
					Floor, Target, Dir = evtFloorReached(floor, Target, Dir)
					continue
				}
				evtButtonPressed(floor, evtType)
			}
		}
		for data := true; data;{
			select {
			case <- evt:
				Target, Dir = evtExternalInput(evt)
			case <- timer.C:
				evtTimeout(Target, Dir)
			default:
				data = false
			}
		}
		time.Sleep(10*time.Millisecond)
	}
}



func evtExternalInput(floor int16, buttonType uint8) int16, uint8{
	sm.Print(fmt.Sprintf("New Order %d, %s",floor, types[buttonType]))

	orders[floor][buttonType] = true
	newTarget, newDir := newTarget(currentFloor, currentDir)
	
	go sm.StatusUpdate(currentFloor, newTarget, false)

	if newTarget == NONE {
		openDoor()
		go orderComplete(floor, currentDir, newDir)
	}

	switch state {
	case idle_s:
		if newTarget != NONE {
			io.SetMotor(newDir)
			state = executing_s
		}
	}
}

func evtButtonPressed(floor int16, buttonType uint8){
	go sm.AddButtonPress(floor, buttonType)
}


func evtTimeout(open bool){
	if stopped {
		stopped = false
		return
	}
	open = false

	io.SetDoorLight(0)
	if currentTarget != NONE{
		currentDir := UP
		if currentFloor > currentTarget {
			currentDir = DOWN
		}
		io.SetMotor(currentDir)
		
		state = executing_s
		return
	}
	
	state = idle_s
}


func evtFloorReached(Floor int16, Target int16, Dir uint8){

	newTarget, newDir := newTarget(Floor, Dir)
	sm.Print(fmt.Sprintf("Reached floor: %d New target: %d", Floor, newTarget))
	go sm.StatusUpdate(Floor, newTarget, false)

	switch state {
	case open_s:
	case idle_s:
	default:
		if currentTarget == Floor{
			io.SetMotor(STOP)
			openDoor()
			orderComplete(Floor, Dir, newDir)
			return
		}
		if currentTarget == NONE {
			if newTarget == NONE {
				io.SetMotor(STOP)
				state = idle_s
				return
			}
			io.SetMotor(newDir)
			state = executing_s
		}
		return
	}
	//fmt.Println("Floor reached in wrong state")
}



func openDoor(open bool)bool{
	state = open_s
	io.SetMotor(STOP)
	io.SetDoorLight(1)

	//Check if timer returned before 
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}

	timer.Reset(time.Second*3, evtTimeout)
	return true
}

//Returns first in current direction
//If non existant, returns first in other direction
//Assumes it swaps direction if no orders in current direction
func newTarget(floor int16, dir uint8) (int16, uint8){
	above := NONE
	below := NONE
	for i := (floor + 1) * 3 ; i < m * 3; i++{
		if orders[i / 3][i % 3] {
			above = i / 3
			break
		}
	}
	for i := int16(0); i < floor * 3; i++{
		if orders[i / 3][i % 3] {
			below = i / 3
		}
	}
	if dir == UP { 
		if above != NONE {
			return above, UP
		}
		return below, DOWN

	//Dir down
	}else if below != NONE {
		return below, DOWN
	}
	return above, UP
}

//may not do anything
func orderComplete(floor int16, dir uint8, newDir uint8){
	if orders[floor][CAB] {
		orders[floor][CAB] = false
		go sm.CallComplete(floor, CAB)
	}
	if orders[floor][dir] {
		orders[floor][dir] = false
		go sm.CallComplete(floor, dir)
	}
	if dir != newDir {
		if orders[floor][1^dir] {
			orders[floor][1^dir] = false
			go sm.CallComplete(floor, 1^dir)
		}
	}
}
