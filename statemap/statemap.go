package sm


import (
	"fmt"
	"sync"
//	"bytes"
//	"encoding/binary"
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

type ButtonPress struct {
	Floor int16
	Type uint8
}

type Evt struct{
	Type uint8
	Floor int16
	Target int16
	Button uint8
	Stuck bool
	Supervise bool
	Cleared [3]bool
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
var elevCh chan<- ButtonPress
var binit bool = false


func Init(id uint8, elev_c chan<- ButtonPress){
	mut = &sync.Mutex{}

	elevCh = elev_c
	sm.mutex = &sync.Mutex{}
	sm.mutex.Lock()
	sm.numNodes = 1

	sm.nodes[ME].id = id
	sm.nodes[ME].floor = NONE
	sm.nodes[ME].target = NONE
	sm.nodes[ME].stuck = false

	for i := 0; i < int(m) * 3; i++{
		sm.orders[i / 3][i % 3] = NONE
		sm.supervisors[i / 3][i % 3] = NONE
	}


	//AddOrdersFromFile(&sm)
	binit = true
	sm.mutex.Unlock()
}


//External
func EvtAccepted(evt *Evt, index int16){
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	switch evt.Type {
	case CALL :
		if evt.Supervise {
			addOrder(evt.Floor, evt.Button, 0, index)
		} else {
			addOrder(evt.Floor, evt.Button, index, 0)
		}
	}
}

//External
//Only happens if node is dc
func EvtDismissed(evt *Evt, index int16){
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	switch evt.Type {
	case CALL :
		delegateButtonPress(evt.Floor, evt.Button)
	}
}

//External
func EvtRegister(evt *Evt, index int16){
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	switch evt.Type {
	case CALL :
		if evt.Supervise {
			addOrder(evt.Floor, evt.Button, index, 0)
			
		} else {
			addOrder(evt.Floor, evt.Button, 0, index)
		}

	case STATE :
		sm.nodes[index].floor = evt.Floor
		sm.nodes[index].target = evt.Target
		sm.nodes[index].stuck = evt.Stuck

		evt.Cleared[CAB] = false
		removeOrders(evt.Floor, evt.Cleared)

		if evt.Stuck {
			redistributeOrders(index, false)
		}
	}
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
	redistributeOrders(index, true)
	sm.mutex.Unlock()
}

//External
func GetState(index int16) (int16, int16, bool){
	sm.mutex.Lock()
	floor := sm.nodes[index].floor
	target := sm.nodes[index].target
	stuck := sm.nodes[index].stuck
	sm.mutex.Unlock()
	return floor, target, stuck
}


//External
func StatusUpdate(floor int16, target int16, stuck bool, cleared [3]bool){
	sm.mutex.Lock()

	sm.nodes[ME].floor = floor
	sm.nodes[ME].target = target
	sm.nodes[ME].stuck = stuck

	removeOrders(floor, cleared)

	evt := &Evt{Type: STATE, Floor: floor, Target: target, Stuck: stuck, Cleared: cleared}
	for i := uint8(1); i < sm.numNodes; i++ {
		sm.nodes[i].send <- evt
	}
	sm.mutex.Unlock()
}

//External
func AddButtonPress(floor int16, buttonType uint8){
	sm.mutex.Lock()
	delegateButtonPress(floor, buttonType)
	sm.mutex.Unlock()
}

//internal
//burde kanskje returnere channel bare så fikser elev evt biffen?
func delegateButtonPress(floor int16, buttonType uint8) {
	if buttonType == CAB || sm.numNodes == 1{
		addOrder(floor, buttonType, 0, NONE)
		return
	}

	index := -1
	lowestCost := 1000
	for i := 0; i < int(sm.numNodes); i++ {
		nodeCost, nodeIdle := costFunction(floor, buttonType, i)
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
			sm.nodes[1].send <- &evt
		}
	}else{
		evt.Supervise = false
		sm.nodes[index].send <- &evt
	}
}


//internal
func costFunction(floor int16, buttonType uint8, index int) (int, bool) {
	node := &(sm.nodes[index]);
	var cost int16

	if node.stuck {
		return -1, true
	}
	if node.target == NONE {
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



//internal
func redistributeOrders(index int16, removed bool) {
	//Stuck or removed, redistribute orders
	for f := int16(0); f < m; f++ {
		if sm.orders[f][UP] == index{
			sm.orders[f][UP] = NONE
			sm.supervisors[f][UP] = NONE
			delegateButtonPress(f, UP)
		}
		if sm.orders[f][DOWN] == index{
			sm.orders[f][DOWN] = NONE
			sm.supervisors[f][DOWN] = NONE
			delegateButtonPress(f, DOWN)
		}
	}
	//if removed
	//TODO: maybe add new supervisor (Not really necessary according to spec)
}

//internal
func addOrder(floor int16, buttonType uint8, index int16, supervisor int16){
	sm.orders[floor][buttonType] = index
	sm.supervisors[floor][buttonType] = supervisor
	if index == ME {
		elevCh <- ButtonPress{floor, buttonType}
	}
}

//internal
func removeOrders(floor int16, clear [3]bool){
	if clear[UP] {
		sm.orders[floor][UP] = NONE
		sm.supervisors[floor][UP] = NONE
	}
	if clear[DOWN] {
		sm.orders[floor][DOWN] = NONE
		sm.supervisors[floor][DOWN] = NONE
	}
	if clear[CAB] {
		sm.orders[floor][CAB] = NONE
		sm.supervisors[floor][CAB] = NONE
	}
}


//Would have preferred to lock the map while printing but it is slow
//Any concurrent write to the statemap will manifest only in the printed output
var first int = -1
const numstrings = 80
var mut *sync.Mutex
var strings [numstrings]string
var firstRun = true

func Print(s string){
	if binit {
		mut.Lock()
		first++
		if first < numstrings {
			strings[first] = s
		}
		mut.Unlock()
	}
}

func PrintMap(){
	mut.Lock()

	if !firstRun {
		fmt.Printf("%c[%dA\r",27, 6 + m) 	//up 6+m lines
		fmt.Printf("%c[J\r",27)				//Clear untill end of screen
	}
	firstRun = false

	for i := 0; i <= first; i++ {
		fmt.Println(strings[i])
	}
	first = -1

	num := int(sm.numNodes)
	fmt.Printf(" F  - | U , D , C | \n");
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
	mut.Unlock()
}