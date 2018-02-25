package print


import (
	"fmt"
	"sync"
	"reflect"
	"container/list"
)

//Print buffer
const numStrings = 80

//Allowed static print functions active at a time
const numStatic = 10


var mut *sync.Mutex

var strings []string
var static *list.List


func Init(){
	strings = make([]string, 0, numStrings)
	static = list.New()
	mut = &sync.Mutex{}
	fmt.Printf("\n")
}

//Printf
func Format(f string, args ...interface{}){
	mut.Lock()
	defer mut.Unlock()
	if len(strings) < numStrings {
		strings  = append(strings, fmt.Sprintf(f, args...))
	}
}

//Println
func Line(args ...interface{}){
	mut.Lock()
	defer mut.Unlock()
	if len(strings) < numStrings {
		strings = append(strings, fmt.Sprintln(args...))
	}
}

//Only one line allowed
func StaticVars(args ...interface{}) *list.Element {
	mut.Lock()
	f := func() int {
		for _, v := range args {
			switch t := v.(type){
			case string:
				fmt.Print(t)
			default:
				fmt.Printf("%v", reflect.Indirect(reflect.ValueOf(v)))
			}
		}
		fmt.Printf("\n")
		return 1
	}
	mut.Unlock()
	return AddStatic(f)
}

//Should preferably have fixed print height
func AddStatic(f func() int) *list.Element {
	mut.Lock()
	defer mut.Unlock()
	return static.PushFront(f)
}

func RemoveStatic(entry *list.Element){
	mut.Lock()
	defer mut.Unlock()
	static.Remove(entry)
}


var linesPrinted = 0
//Prints buffered prints and calls static print functions
func Display(){
	mut.Lock()
	defer mut.Unlock()

	fmt.Printf("\x1b[%dA\r",linesPrinted) 	//up n lines
	fmt.Printf("\x1b[J\r")			//Clear untill end of screen
	linesPrinted = 0

	for _,s := range strings {
		fmt.Printf(s)
	}
	strings = strings[:0]

	for e := static.Front(); e != nil; e = e.Next() {
		printFunc := e.Value.(func() int)
		linesPrinted += printFunc()
	}
}