package sm


import (
	"net"
	"fmt"

//	"bytes"
//	"encoding/binary"
	"sync"
)

const m = 4
const n = 256

type status struct{
	id uint8
	floor uint8
	target uint8
	stuck bool
}

type floorOrders struct{
	up uint8
	down uint8
	cab uint8
}

type stateMap struct{
	mutex *sync.Mutex
	orders [m]floorOrders
	nodes [n]status
	numNodes uint8
}

var sm stateMap
var myID uint8

//TODO - sørg for at IP blir satt til riktig, samt de andre variablene
void sm_init(int floors, id uint8){
	sm.mutex = &sync.Mutex{}
	sm.mutec.Lock()
	sm.numNodes = 1
	myID = id

	sm.nodes[0].id = myID
	sm.nodes[0].floor = 0
	sm.nodes[0].target = 0
	sm.nodes[0].stuck = 0

	//AddOrdersFromFile(&sm)
	sm.mutex.Unlock()
}




//Returner total avstand en node må bruke før den kan betjene et kall.
int sm_cost_function(floor uint8, buttonType uint8, id uint8) {
	sm.mutex.Lock()
	node := &(sm.nodes[id]);

	if node->stuck {
		return -1;
	}
	if node->target == 0 {
		return abs(node->floor - floor);
	}

	var dir := DOWN
	if node->floor < node->target {
		dir = UP
	}

	if (buttonType == UP && dir == DOWN) {                           //Kall oppover, men heisen går nedover
		return node.floor + floor;
	}
	if (buttonType == UP && dir == UP && floor < node.floor) {       //Kall oppover, men heisen er over kallet
		return (m - 1) - node.floor + (m - 1) - floor;
	}
	if (buttonType == DOWN && dir == UP) {                           //Kall nedover, men heisen går oppover
		return (m - 1) - node.floor + (m - 1) - floor;
	}
	if (buttonType == DOWN && dir == DOWN && floor > node.floor) {   //Kall nedover, men heisen er under kallet
		return node.floor + floor;
	}
	sm.mutex.Unlock()
	return abs(node.floor - floor);
}



//helper function
func sm_redistribute(id uint8) {
	for (int f = 0; f < m; f++) {
		if sm.orders[f].up == id{
			sm.orders[f].up = 0
			sm_delegate_button_press(f, UP)
		}
		if sm.orders[f].down == id{
			sm.orders[f].down = 0
			sm_delegate_button_press(f, DOWN)
		}
		if sm.orders[f].cab == id{
			sm.orders[f].cab = 0
			sm_delegate_button_press(f, CAB)
		}
	}
}

//external function
func sm_add_node(id uint8, floor uint8, target uint8, stuck bool) int{
	sm.mutex.Lock()
	index := sm.numNodes
	sm.nodes[index].id = id
	sm.nodes[index].floor = floor
	sm.nodes[index].target = target
	sm.nodes[index].stuck = stuck
	sm.numNodes++
	sm.mutex.Unlock()
	return index
}

//external function
func sm_remove_node(index int){
	sm.mutex.Lock()
	node := sm.nodes[index]
	for i := index; i < numNodes; i++{
		sm.orders[i] = sm.orders[i + 1]
	}
	sm.numNodes--
	sm_redistribute(node.id)
	sm.mutex.Unlock()
}

//external function
void sm_set_node_stuck(index int, status bool) {
	sm.mutex.Lock()
	sm.nodes[index].stuck = stuck
	sm.mutex.Unlock()
}


//Would have preferred to lock the map while printing but it is slow
//Any concurrent write to the statemap will manifest only in the printed output
void smPrintMap(){
	fmt.Printf("\n F  - | U , D , C | \n");
	for(int f = m-1; f >= 0; f--){
		fmt.Printf("%3d - |%3d,%3d,%3d|\n",f, sm.orders[f].up,
								      		  sm.orders[f].down,
								      		  sm.orders[f].cab);
	}
	fmt.Printf("Connected nodes")

	fmt.Printf("\nid     | ")
	for n := 0; n < sm.numNodes; n++){
		fmt.Printf("%3d |", sm.nodes[n].id)
	}
	fmt.Printf("\nfloor  | ")
	for n := 0; n < sm.numNodes; n++){
		fmt.Printf("%3d |", sm.nodes[n].floor)
	}
	fmt.Printf("\ntarget | ")
	for n := 0; n < sm.numNodes; n++){
		fmt.Printf("%3d |", sm.nodes[n].target)
	}
	fmt.Printf("\nstuck  | ")
	for n := 0; n < sm.numNodes; n++){
		t := 0
		if sm.nodes[n].stuck{
			t = 1
		}
		fmt.Printf("%3d |", t)
	}

	fmt.Printf("\n");
}


//Finner best egnede node og legger til den ordren i den nodens ordreliste.
void sm_delegate_button_press(int node_index, int floor, elev_button_type_t buttonType) {
	if (buttonType == BUTTON_COMMAND) {
		sm_add_order(floor, BUTTON_COMMAND, node_index);
		return;
	}

	int chosenNodeIndex = 0;
	int lowestCost = sm_cost_function(floor, buttonType, 0);
	for (int i = 1; i < localMap.nodes; i++) {
		int nodeCost = sm_cost_function(floor, buttonType, i);																//Finner den noden med lavest cost
		if (nodeCost < lowestCost && nodeCost != -1) {
			chosenNodeIndex = i;
			lowestCost = nodeCost;
		}
		else if (nodeCost == lowestCost && localMap.nodeList[i].idleStatus && !localMap.nodeList[chosenNodeIndex].idleStatus){			//Ved lik cost brukes heller en heis som ikke utfører noen kall for øyeblikket
			chosenNodeIndex = i;
		}
	}
	sm_add_order(floor, buttonType, chosenNodeIndex);
}


//Sjekker om lyset av en gitt type skal på i en gitt etasje.
int sm_check_light(elev_button_type_t buttonType, int floor) {
	if (buttonType == BUTTON_COMMAND) {
		if (localMap.orderList[self_index].floor[floor].cm == 1) {
			return 1;
		}
	}
	else {
		for (int i = 0; i < localMap.nodes; i++) {
			if (localMap.orderList[i].floor[floor].type[buttonType] == 1) {
				return 1;
			}
		}
	}
	return 0;
}


int sm_get_num_nodes() {
	return sm.numNodes;
}

void sm_service_all_external_orders() {
	for (int i = 0; i < localMap.nodes; i++) {
		for (int j = 0; j < N_FLOORS; j++) {
			if (localMap.orderList[i].floor[j].up == 1) {
				sm_add_order(j, BUTTON_CALL_UP, self_index);
			}
			if (localMap.orderList[i].floor[j].dn == 1) {
				sm_add_order(j, BUTTON_CALL_DOWN, self_index);
			}
		}
	}
}

void sm_send_internal_calls_as_orders() {
	for (int i = 0; i < N_FLOORS; i++) {
		if (localMap.orderList[self_index].floor[i].cm == 1) {
			mq_addButtonPress(i, BUTTON_COMMAND);
		}
	}
}

void sm_remove_all_external_orders() {
	for (int i = 0; i < N_FLOORS; i++) {
		sm_remove_order(i, BUTTON_CALL_UP, self_index);
		sm_remove_order(i, BUTTON_CALL_DOWN, self_index);
	}
}


void sm_add_order(int floor, elev_button_type_t buttonType, int node_index){
	localMap.orderList[node_index].floor[floor].type[buttonType] = 1;
}

void sm_remove_order(int floor, elev_button_type_t buttonType, int node_index){
	localMap.orderList[node_index].floor[floor].type[buttonType] = 0;
}

void sm_clear_floor(int floor, int node_index){
	for (int i = 0; i < numNodes; i++){
		localMap.orderList[i].floor[floor].up = 0;
		localMap.orderList[i].floor[floor].dn = 0;
	}
	localMap.orderList[node_index].floor[floor].cm = 0;
}
