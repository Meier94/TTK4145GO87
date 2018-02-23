package elev

import (
	"./io"
	"../statemap"
	"time"
	"sync"
//	"fmt"
)

const m int16 = 4

//Event types
const UP uint8 = 0
const DOWN uint8 = 1
const CAB uint8 = 2
const STOP uint8 = 3
const FLOOR uint8 = 3

const NONE int16 = int16(-1)
var currentFloor int16 = NONE
var currentTarget int16 = NONE
var currentDir uint8 = UP

//States
const idle_s = 0
const init_s = 1
const open_s = 2
const stuck_s = 3
const executing_s = 4
var state int = init_s

//Print helper
var types [4]string = [4]string{"Up", "Down", "Cab", "Arrival"}


var orders[m][3] bool
var mutex *sync.Mutex

var timer *time.Timer
var timeTex *sync.Mutex
var stopped bool = false
var open bool = false


func Init(id uint8) bool {
	if !io.Init(){
		return false
	}
	io.ClearAllLights()
	sm.AddFunction(evtExternalInput)

	timer = time.NewTimer(0 * time.Millisecond)
	<- timer.C
	mutex = &sync.Mutex{}
	timeTex = &sync.Mutex{}
	mutex.Lock()
	for i := 0 ; i < int(m) * 3; i++{
		orders[i / 3][i % 3] = false
	}
	mutex.Unlock()

	sm.Init(id)

	io.SetMotor(UP)
	go triggerEvents()
	return true
}

func evtExternalInput(floor int16, buttonType uint8){
	mutex.Lock()
	defer mutex.Unlock()
	sm.Printf(fmt.SprintF("New Order %d, %s",floor, types[buttonType]))
	orders[floor][buttonType] = true
	newTarget, newDir := newTarget(currentFloor, currentDir)
	defer updateCurrent(currentFloor, newTarget, newDir)

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
	mutex.Lock()
	mutex.Unlock()
	sm.AddButtonPress(floor, buttonType)
}


func evtTimeout(){
	timeTex.Lock()
	defer timeTex.Unlock()
	if stopped {
		stopped = false
		return
	}
	open = false
	mutex.Lock()
	defer mutex.Unlock()
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

func updateCurrent(newFloor int16, newTarget int16, newDir uint8){
	currentTarget = newTarget
	currentFloor = newFloor
	currentDir = newDir
	//fmt.Println("New state, floor: ", newFloor, " target: ", newTarget, " dir: ", newDir )
}

func evtFloorReached(floor int16){
	mutex.Lock()

	newTarget, newDir := newTarget(floor, currentDir)
	defer sm.StatusUpdate(floor, newTarget, false)
	defer mutex.Unlock()

	defer updateCurrent(floor, newTarget, newDir)
	

	switch state {
	case open_s:
	case idle_s:
	default:
		if currentTarget == floor{
			io.SetMotor(STOP)
			openDoor()
			orderComplete(floor, currentDir, newDir)
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


func triggerEvents(){
	for {
		if io.GetInputs() {
			for {
				floor, evtType := io.GetEvent()
				if(evtType > 3){
					break
				}
				////fmt.Printf("Event: %s, floor: %d\n",types[evtType],floor)
				if evtType == FLOOR {
					evtFloorReached(floor)
					continue
				}
				evtButtonPressed(floor, evtType)
			}
		}
		time.Sleep(10*time.Millisecond)
	}
}

func openDoor(){
	state = open_s
	io.SetDoorLight(1)
	timeTex.Lock()
	if open {
		stopped = !timer.Stop()
	}
	open = true
	timer = time.AfterFunc(time.Second*3, evtTimeout)
	timeTex.Unlock()
	io.SetMotor(STOP)
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
		sm.CallComplete(floor, CAB)
	}
	if orders[floor][dir] {
		orders[floor][dir] = false
		sm.CallComplete(floor, dir)
	}
	if dir != newDir {
		if orders[floor][1^dir] {
			orders[floor][1^dir] = false
			sm.CallComplete(floor, 1^dir)
		}
	}
}
