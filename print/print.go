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
//Static prints are always visible
const numStatic = 10

type staticPrint struct{
	e *list.Element
}

var printTex *sync.Mutex

var strings []string
var static *list.List


func Init(){
	strings = make([]string, 0, numStrings)
	static = list.New()
	printTex = &sync.Mutex{}
	fmt.Printf("\n")
}

//Printf
func Format(f string, args ...interface{}){
	printTex.Lock()
	defer printTex.Unlock()
	if len(strings) < numStrings {
		strings  = append(strings, fmt.Sprintf(f, args...))
	}
}

//Println
func Line(args ...interface{}){
	printTex.Lock()
	defer printTex.Unlock()
	if len(strings) < numStrings {
		strings = append(strings, fmt.Sprintln(args...))
	}
}

//Only one line allowed
func StaticVars(args ...interface{}) staticPrint {
	printTex.Lock()
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
	printTex.Unlock()
	return AddStatic(f)
}

func AddStatic(f func() int) staticPrint {
	printTex.Lock()
	defer printTex.Unlock()
	return staticPrint{static.PushFront(f)}
}

func (print staticPrint) Remove(){
	printTex.Lock()
	static.Remove(print.e)
	printTex.Unlock()
}


var linesPrinted = 0
//Prints buffered prints and calls static print functions
func Display(){
	printTex.Lock()
	defer printTex.Unlock()

	if linesPrinted > 0 {
		fmt.Printf("\x1b[%dA\r",linesPrinted) 	//up n lines
		fmt.Printf("\x1b[J\r")					//Clear untill end of screen
	}
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