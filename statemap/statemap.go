package sm


import (
	"fmt"
//	"bytes"
//	"encoding/binary"
	"sync"
)

const m int16 = 4
const n int16 = 256

const UP uint8 = 0
const DOWN uint8 = 1
const CAB uint8 = 2
const STOP uint8 = 3

const NONE = int16(-1)
const ME = int16(0)

const CALL_COMPLETE uint8 = 200
const CALL uint8 = 201
const FAILED_CALL uint8 = 202
const STATE uint8 = 203

type nodeInfo struct{
	id uint8
	floor int16
	target int16
	stuck bool
	send chan *Evt
}

type Evt struct{
	Type uint8
	Floor int16
	Target int16
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

var ElevMapUpdate func(int16)

func Init(id uint8){
	sm.mutex = &sync.Mutex{}
	sm.mutex.Lock()
	sm.numNodes = 1

	sm.nodes[ME].id = id
	sm.nodes[ME].floor = 0
	sm.nodes[ME].target = 0
	sm.nodes[ME].stuck = false



	//AddOrdersFromFile(&sm)
	sm.mutex.Unlock()
}


func EvtAccepted(evt *Evt, index int16){
	switch evt.Type {
	case CALL :
		if evt.Supervise {
			addOrder(evt.Floor, evt.Button, index, 0)
		} else {
			addOrder(evt.Floor, evt.Button, 0, index)
		}
	}
}

//Only happens if node is dc
func EvtDismissed(evt *Evt, index int16){
	switch evt.Type {
	case CALL :
		DelegateButtonPress(evt.Floor, evt.Button)
	}
}

func EvtRegister(evt *Evt, index int16){
	switch evt.Type {
	case CALL :
		if evt.Supervise {
			addOrder(evt.Floor, evt.Button, index, 0)
		} else {
			addOrder(evt.Floor, evt.Button, 0, index)
		}

	case CALL_COMPLETE :
		removeOrder(evt.Floor, evt.Button)

	case STATE :
		sm.nodes[index].floor = evt.Floor
		sm.nodes[index].target = evt.Target
		sm.nodes[index].stuck = evt.Stuck
		if evt.Stuck {
			Redistribute(index, false)
		}
	}
}

//Denne må skrives om til å fungere med startindex 1
func sm_cost_function(floor int16, buttonType uint8, index int) (int, bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	node := &(sm.nodes[index]);

	var cost int16

	if node.stuck {
		return -1, true
	}
	if node.target == 0 {
		cost = node.floor - floor
		if node.floor < floor {
			cost = -cost
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
		cost = node.floor - floor
		if node.floor < floor {
			cost = -cost
		}
	}
	return int(cost), false
}



//helper function
func Redistribute(index int16, removed bool) {
	//Stuck or removed, redistribute orders
	for f := int16(0); f < m; f++ {
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
	//if removed
	//TODO: maybe add new supervisor (Not really necessary according to spec)
}

func addOrder(floor int16, buttonType uint8, index int16, supervisor int16){
	sm.orders[floor][buttonType] = index
	sm.supervisors[floor][buttonType] = supervisor
	if index == ME {
		ElevMapUpdate(floor)
	}
}

func removeOrder(floor int16, buttonType uint8){
	sm.orders[floor][buttonType] = NONE
	sm.supervisors[floor][buttonType] = NONE
}

//external function
func AddNode(id uint8, floor int16, target int16, stuck bool, send chan *Evt) int16{
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

//external function
func RemoveNode(index int16){
	sm.mutex.Lock()
	for i := index; i < int16(sm.numNodes); i++{
		sm.nodes[i] = sm.nodes[i + 1]
	}
	sm.numNodes--
	Redistribute(index, true)
	sm.mutex.Unlock()
}

func GetState(index int16) (int16, int16, bool){
	sm.mutex.Lock()
	floor := sm.nodes[index].floor
	target := sm.nodes[index].target
	stuck := sm.nodes[index].stuck
	sm.mutex.Unlock()
	return floor, target, stuck
}

//Returns first in current direction
//If non existant, returns first in other direction
func newTarget(floor int16, dir uint8) int16{
	above := NONE
	below := NONE
	for i := (floor + 1) * 3 ; i < m * 3; i++{
		if sm.orders[i / m][i % 3] == ME {
			above = i / m
			break
		}
	}
	for i := int16(0); i < floor * 3; i++{
		if sm.orders[i / m][i % 3] == ME {
			below = i / m
		}
	}
	if dir == UP { 
		if above != NONE {
			return above
		}
		return below
	}else if below != NONE {
		return below
	}
	return above
}


func StatusUpdate(floor int16, stuck bool) int16{
	sm.mutex.Lock()
	target := sm.nodes[ME].target
	//Order completed
	if floor == target {
		dir := UP
		if sm.nodes[ME].floor > target {
			dir = DOWN
		}
		evt := &Evt{Type: CALL_COMPLETE, Floor: floor, Button: dir}
		if id := sm.supervisors[floor][dir]; id != -1 {
			sm.nodes[id].send <- evt
		}
		removeOrder(floor, dir)
		target = newTarget(floor, dir)
	}

	sm.nodes[ME].floor = floor
	sm.nodes[ME].target = target
	sm.nodes[ME].stuck = stuck

	evt := &Evt{Type: STATE, Floor: floor, Target: target, Stuck: stuck}
	for i := uint8(0); i < sm.numNodes; i++ {
		sm.nodes[i].send <- evt
	}
	sm.mutex.Unlock()
	return target
}


//burde kanskje returnere channel bare så fikser elev evt biffen?
func DelegateButtonPress(floor int16, buttonType uint8) {
	if buttonType == CAB {
		addOrder(floor, CAB, 0, -1)
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
func sm_check_light(floor int16, buttonType int) bool{

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