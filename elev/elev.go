package elev

import (
	"87/elev/io"
	"87/statemap"
	"87/print"
	"time"
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

//current
var cFloor int16 = NONE
var	cTarget int16 = NONE
var	cDir uint8 = UP


var orders[m][3] bool
var openTimer *time.Timer
var stuckTimer *timer.Timer
var evt_c chan sm.ButtonPress



func Init(id uint8) bool {
	if !io.Init(){
		return false
	}
	io.ClearAllLights()

	openTimer = time.NewTimer(0 * time.Millisecond)
	<- openTimer.C

	evt_c = make(chan sm.ButtonPress, m*3)
	sm.Init(id, evt_c)

	io.SetMotor(UP)
	go triggerEvents()
	return true
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
					clearedOrders := evtFloorReached(floor)
					go sm.StatusUpdate(cFloor, cTarget, false, clearedOrders)
					continue
				}
				go sm.AddButtonPress(floor, evtType)
			}
		}
		for data := true; data;{
			select {
			case Press := <- evt_c:
				clearedOrders := evtExternalInput(Press.Floor, Press.Type)
				go sm.StatusUpdate(cFloor, cTarget, false, clearedOrders)
			case <- openTimer.C:
				evtTimeout()
			default:
				data = false
			}
		}
		time.Sleep(10*time.Millisecond)
	}
}


func evtExternalInput(floor int16, buttonType uint8) [3]bool {
	print.Format("New Order %d, %s\n",floor, types[buttonType])

	orders[floor][buttonType] = true
	nTarget, nDir := newTarget(cFloor, cDir)
	
	clearedOrders := [3]bool{}
	if nTarget == NONE {
		openDoor()
		clearedOrders = orderComplete(cFloor, cDir, nDir)
	}

	switch state {
	case idle_s:
		if nTarget != NONE {
			io.SetMotor(nDir)
			state = executing_s
		}
	}
	cDir = nDir
	cTarget = nTarget
	return clearedOrders
}


func evtTimeout(){

	io.SetDoorLight(0)
	if cTarget != NONE{
		cDir := UP
		if cFloor > cTarget {
			cDir = DOWN
		}
		io.SetMotor(cDir)
		
		state = executing_s
		return
	}
	
	state = idle_s
}


func evtFloorReached(nFloor int16) [3]bool  {
	nTarget, nDir := newTarget(nFloor, cDir)

	clearedOrders := [3]bool{}
	switch state {
	case open_s:
	case idle_s:
	default:
		if cTarget == nFloor{
			openDoor()
			clearedOrders = orderComplete(nFloor, cDir, nDir)
			break
		}
		if cTarget == NONE {
			if nTarget == NONE {
				io.SetMotor(STOP)
				state = idle_s
				break
			}
			io.SetMotor(nDir)
			state = executing_s
		}
	}
	cDir = nDir
	cFloor = nFloor
	cTarget = nTarget
	return clearedOrders
}



func openDoor(){
	state = open_s
	io.SetMotor(STOP)
	io.SetDoorLight(1)

	//Try to stop openTimer
	if !openTimer.Stop() {
		select {
		case <-openTimer.C: //it just completed (evtTimeout() will not run)
		default:		//it was not running
		}
	}

	openTimer.Reset(time.Second*3)
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
func orderComplete(floor int16, dir uint8, newDir uint8) [3]bool {
	cleared := [3]bool{}
	if orders[floor][CAB] {
		orders[floor][CAB] = false
		cleared[CAB] = true
	}
	if orders[floor][dir] {
		orders[floor][dir] = false
		cleared[dir] = true
	}
	if dir != newDir {
		if orders[floor][1^dir] {
			orders[floor][1^dir] = false
			cleared[1^dir] = true
		}
	}
	return cleared
}
