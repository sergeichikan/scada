package main

/*
typedef struct {
  double Value;
  long long int Iteration;
  long long int CreateTimestamp;
  long long int ReadTimestamp;
} res;
*/
import "C"
import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

// go build -o ./driver1/main ./driver1/main.go
// go build -o ./driver1/libdriver1.so -buildmode=c-shared ./driver1/main.go

type res struct {
	c C.res
}

func (r *res) update() {
	rand.Seed(time.Now().UnixNano())
	r.c.Value = C.double(1 + rand.Float64()*(99))
	r.c.Iteration = C.longlong(int64(r.c.Iteration) + 1)
	r.c.CreateTimestamp = C.longlong(time.Now().UnixNano())
}

func (r *res) updateReadTimestamp() {
	r.c.ReadTimestamp = C.longlong(time.Now().UnixNano())
}

func (r *res) run() {
	for {
		r.update()
		time.Sleep(updateDelay)
	}
}

func (r res) json() string {
	res, err := json.Marshal(r.c)
	if err != nil {
		log.Fatal(err)
	}
	return string(res)
}

func (r res) string() string {
	return fmt.Sprintf("%v %v %v %v", r.c.Value, r.c.Iteration, r.c.CreateTimestamp, r.c.ReadTimestamp)
}

func (r res) bin() string {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, r.c); err != nil {
		log.Fatal(err)
	}
	return buf.String()
}

var updateDelay = time.Millisecond * 50
var value res

//export Connect
func Connect(d int64) {
	updateDelay = time.Duration(d)
}

//export Disconnect
func Disconnect() {

}

//export Read
func Read(mode string) *C.res {
	value.updateReadTimestamp()
	switch mode {
	case "json":
		fmt.Println(value.json())
	case "str":
		fmt.Println(value.string())
	case "bin":
		fmt.Println(value.bin())
	}
	return &value.c
}

//export Run
func Run() {
	go value.run()
}

// We need the main function to make possible
// CGO compiler to compile the package as C shared library

// Для подкл через CGO main должна быть (хотябы пустая)

func main() {
	runModePtr := flag.Bool("r", false, "cmd mode on")
	updateDelayPtr := flag.Int64("d", int64(updateDelay), "update delay")
	flag.Parse()
	Connect(*updateDelayPtr)
	if *runModePtr {
		runCmdMode()
	}
}

// Режим работы через консоль
// run - запускает цикл обновлений
// r <mode> - выводит в консоль данные (mode - json или str или bin)
func runCmdMode() {
	reader := bufio.NewReader(os.Stdin)
	readArg := "json"
	for {
		text, err := reader.ReadString('\n') // ковычки обязательно одинарные
		if err != nil {
			log.Fatal(err)
		}
		args := strings.Split(strings.TrimSpace(text), " ")
		//fmt.Fprintf(os.Stderr, "%v\n", args)
		switch args[0] {
		case "run":
			Run()
		case "r":
			if len(args) >= 2 {
				readArg = args[1]
			}
			Read(readArg)
		default:
			return
		}
	}
}
