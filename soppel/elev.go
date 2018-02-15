package elev
/*
#include "io.h"
#cgo LDFLAGS: -L . -lcomedi
*/
import "C"

import (
	"fmt"
	"time"
)

func Init(){
	C.io_init()

	C.set_motor(0)
	time.Sleep(1*time.Second)
	C.set_motor(2)
	time.Sleep(1*time.Second)
	C.set_motor(1)
	time.Sleep(1*time.Second)
	C.set_motor(2)

	for{
		var i C.int = 0
		for i < 4{
			time.Sleep(1*time.Second)
			C.set_floor_light(i)
			C.set_button_light(i,0,0)
			C.set_button_light(i,1,0)
			C.set_button_light(i,2,0)
			i++
		}
		C.set_button_light(3,0,1)
		time.Sleep(1*time.Second)
		C.set_button_light(3,0,0)
	}
}