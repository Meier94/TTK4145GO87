package print


import (
	"fmt"
	"sync"
)

//Print buffer
const numStrings = 80

//Allowed static print functions active at a time
const numStatic = 5


var mut *sync.Mutex

type staticPrint struct {
	f func() int
	linesPrinted int
}

var strings []string
var stringCount = 0

var static []staticPrint
var staticCount int = 0

var running = false

func Init(){
	running = true
	strings = make([]string, 0, numStrings)
	static = make([]string, 0, numStatic)
	mut = &sync.Mutex{}
}

func Format(f string, args ...interface{}){
	if binit {
		mut.Lock()
		if stringCount < numStrings {
			strings[stringCount] = fmt.Sprintf(f, args)
			stringCount++
		}
		mut.Unlock()
	}
}

func Line(args ...interface{}){
	if binit {
		mut.Lock()
		if len(strings) < numStrings {
			strings.append(fmt.Sprintln(args))
			stringCount++
		}
		mut.Unlock()
	}
}

func AddStatic(f func() int) bool{
	mut.Lock()
	defer mut.Unlock()
	if len(static) < numStatic {
		static.append(staticPrint{f, 0})
		staticCount++
		return true
	}
	return false
}


func PrintMap() int{
	mut.Lock()
	for s := range static {
			fmt.Printf("%c[%dA\r",27, s.lines) 	//up n lines
	}
	fmt.Printf("%c[J\r",27)			//Clear untill end of screen


	for s := range strings {
		fmt.Printf(s)
	}
	numStrings = 0

	for s := range static {
			s.lines = s.f()
	}


	mut.Unlock()
	return numlines
}




