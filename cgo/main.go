package cgo

/*
#cgo CFLAGS: -I./driver1/
#cgo LDFLAGS: -L./driver1/ -ldriver1
#cgo CFLAGS: -I./driver2/
#cgo LDFLAGS: -L./driver2/ -ldriver2
#include <libdriver1.h>
#include <libdriver2.h>
*/
import "C"
import (
	"fmt"
	"time"
)

// Для запуска через cgo
// libdriver1.so должна лежать в /usr/lib/

// Команда чтоб создать ссылку:
// sudo ln -s /home/notus/go/src/scada/driver1/libdriver1.so /usr/lib/

// TODO переписать все

type driverResult struct {
	value float64
	iteration int64
}

type Driver struct {
	result driverResult
}

var drivers []Driver

func init() {
	drivers = append(drivers, Driver{result: driverResult{0, 0.0}})
}

func main() {
	C.Connect()
	C.Run()
	for {
		result := C.Read()
		drivers[0].result = driverResult{float64(result.value), int64(result.iteration)}
		fmt.Println(drivers[0].result)
		time.Sleep(time.Millisecond * 100)
	}
}
