package sm


import (
	"fmt"
//	"bytes"
//	"encoding/binary"
	"sync"
)

const m = 4
const n = 256

const UP uint8 = 0
const DOWN uint8 = 1
const CAB uint8 = 2
const STOP uint8 = 3

const NONE = int16(-1)

const COMPLETE_CALL uint8 = 200
const CALL uint8 = 201
const FAILED_CALL uint8 = 204

type nodeInfo struct{
	id uint8
	floor uint8
	target uint8
	stuck bool
	send chan *Evt
}

type Evt struct{
	Type uint8
	Floor uint8
	Target uint8
	Button uint8
	Stuck bool
	Supervise bool
}

type orderParticipants struct{
	id int16
	buddy int16
}

type stateMap struct{
	mutex 		*sync.Mutex
	orders 		[m][3]int16
	supervisors [m][3]int16
	nodes 		[n]nodeInfo
	numNodes 	uint8
}

var sm = stateMap{}


func sm_init(id uint8){
	sm.mutex = &sync.Mutex{}
	sm.mutex.Lock()
	sm.numNodes = 1

	sm.nodes[0].id = id
	sm.nodes[0].floor = 0
	sm.nodes[0].target = 0
	sm.nodes[0].stuck = false

	//AddOrdersFromFile(&sm)
	sm.mutex.Unlock()
}




//Denne må skrives om til å fungere med startindex 1
func sm_cost_function(floor uint8, buttonType uint8, index int) (int, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	node := &(sm.nodes[index]);

	var cost uint8

	if node.stuck {
		return -1, true
	}
	if node.target == 0 {
		if node.floor < floor {
			cost = floor - node.floor
		} else {
			cost = node.floor - floor
		}
		return int(cost), true
	}

	dir := DOWN
	if node.floor < node.target {
		dir = UP
	}
	if buttonType == UP && dir == DOWN {                          	   //Kall oppover, men heisen går nedover
		cost =  node.floor + floor

	}else if buttonType == UP && dir == UP && floor < node.floor {       //Kall oppover, men heisen er over kallet
		cost =  (m - 1) - node.floor + (m - 1) - floor

	}else if buttonType == DOWN && dir == UP {                           //Kall nedover, men heisen går oppover
		cost =  (m - 1) - node.floor + (m - 1) - floor

	}else if buttonType == DOWN && dir == DOWN && floor > node.floor {   //Kall nedover, men heisen er under kallet
		cost =  node.floor + floor

	}else {
		if node.floor < floor {
			cost = floor - node.floor
		} else {
			cost = node.floor - floor
		}
	}
	return int(cost), false
}



//helper function
func sm_redistribute(index int16) {
	for f := uint8(0); f < m; f++ {
		if sm.orders[f][UP] == index{
			sm.orders[f][UP] = NONE
			sm.supervisors[f][UP] = NONE
			DelegateButtonPress(f, UP)
		}
		if sm.orders[f][DOWN] == index{
			sm.orders[f][DOWN] = NONE
			sm.supervisors[f][DOWN] = NONE
			DelegateButtonPress(f, DOWN)
		}
	}
}

func AddOrder(floor uint8, buttonType uint8, index int16, supervisor int16){
	sm.orders[floor][buttonType] = index
	sm.supervisors[floor][buttonType] = supervisor
}

func sm_remove_order(floor uint8, buttonType uint8, index int16){
	sm.orders[floor][buttonType] = NONE
	sm.supervisors[floor][buttonType] = NONE
}

//external function
func AddNode(id uint8, floor uint8, target uint8, stuck bool, send chan *Evt) int16{
	sm.mutex.Lock()
	index := int16(sm.numNodes)
	sm.nodes[index].id = id
	sm.nodes[index].floor = floor
	sm.nodes[index].target = target
	sm.nodes[index].stuck = stuck
	sm.nodes[index].send = send
	sm.numNodes++
	sm.mutex.Unlock()
	return index
}

func GetState(index int16) (uint8, uint8, bool){
	return sm.nodes[index].floor, sm.nodes[index].target, sm.nodes[index].stuck
}

//external function
func RemoveNode(index int16){
	sm.mutex.Lock()
	for i := index; i < int16(sm.numNodes); i++{
		sm.nodes[i] = sm.nodes[i + 1]
	}
	sm.numNodes--
	sm_redistribute(index)
	sm.mutex.Unlock()
}

//external function
func sm_set_node_stuck(index int16, status bool) {
	sm.mutex.Lock()
	sm.nodes[index].stuck = status
	sm.mutex.Unlock()
}


//burde kanskje returnere channel bare så fikser elev evt biffen?
func DelegateButtonPress(floor uint8, buttonType uint8) {
	if buttonType == CAB {
		AddOrder(floor, CAB, 0, -1)
		return
	}

	index := -1
	lowestCost := 1000
	for i := 0; i < int(sm.numNodes); i++ {
		nodeCost, nodeIdle := sm_cost_function(floor, buttonType, i)
		if nodeCost < lowestCost && nodeCost != -1 {
			index = i
			lowestCost = nodeCost
		} else if nodeCost == lowestCost && nodeIdle {
			index = i
		}
	}

	if index == -1 {
		return
	}

	evt := Evt{Type: CALL, Floor: floor, Button: buttonType}
	if index == 0{
		if sm.numNodes > 1 {
			evt.Supervise = true
			sm.nodes[index].send <- &evt
		}
	}else{
		evt.Supervise = false
		sm.nodes[index].send <- &evt
	}
}


//Sjekker om lyset av en gitt type skal på i en gitt etasje.
func sm_check_light(floor uint8, buttonType int) bool{

	if sm.orders[floor][buttonType] != NONE {
		return true
	}

	return false
}



//Would have preferred to lock the map while printing but it is slow
//Any concurrent write to the statemap will manifest only in the printed output
func smPrintMap(){
	num := int(sm.numNodes)
	fmt.Printf("\n F  - | U , D , C | \n");
	for f := m-1; f >= 0; f--{
		fmt.Printf("%3d - |%3d,%3d,%3d|\n",f, sm.orders[f][UP],
								      		  sm.orders[f][DOWN],
								      		  sm.orders[f][CAB]);
	}
	fmt.Printf("Connected nodes")

	fmt.Printf("\nid     | ")
	for n := 0; n < num; n++ {
		fmt.Printf("%3d |", sm.nodes[n].id)
	}

	fmt.Printf("\nfloor  | ")
	for n := 0; n < num; n++ {
		fmt.Printf("%3d |", sm.nodes[n].floor)
	}

	fmt.Printf("\ntarget | ")
	for n := 0; n < num; n++ {
		fmt.Printf("%3d |", sm.nodes[n].target)
	}

	fmt.Printf("\nstuck  | ")
	for n := 0; n < num; n++ {
		t := 0
		if sm.nodes[n].stuck{
			t = 1
		}
		fmt.Printf("%3d |", t)
	}

	fmt.Printf("\n");
}