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
	"errors"
	"flag"
	"fmt"
	"github.com/rainycape/dl"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// go build -o ./driver1/main ./driver1/main.go
// go build -o ./driver1/libdriver1.so -buildmode=c-shared ./driver1/main.go

type driver struct {
	path string
	lib *dl.DL
	run func()
	read func(mode string) *C.res
	connect func(d int64)
	disconnect func()
}
func (d *driver) load() {
	lib, err := dl.Open(d.path, 0)
	if err != nil {
		log.Fatal(err)
	}
	d.lib = lib
	d.symRun()
	d.symRead()
	d.symConnect()
	d.symDisconnect()
	if err := lib.Close(); err != nil {
		log.Fatal(err)
	}
}
func (d *driver) symRun() {
	if err := d.lib.Sym("Run", &d.run); err != nil {
		log.Fatal(err)
	}
}
func (d *driver) symConnect() {
	if err := d.lib.Sym("Connect", &d.connect); err != nil {
		log.Fatal(err)
	}
}
func (d *driver) symDisconnect() {
	if err := d.lib.Sym("Disconnect", &d.disconnect); err != nil {
		log.Fatal(err)
	}
}
func (d *driver) symRead() {
	if err := d.lib.Sym("Read", &d.read); err != nil {
		log.Fatal(err)
	}
}

type DriverResult struct {
	Value float64
	Iteration int64
	CreateTimestamp int64
	ReadTimestamp int64
}

type durationTest struct {
	durations []int64
}
func (t durationTest) average() int64 {
	var total int64 = 0
	for _, v := range t.durations {
		total += v
	}
	return total / int64(len(t.durations))
}
func (t *durationTest) addDuration(duration time.Duration) {
	t.durations = append(t.durations, duration.Nanoseconds())
}
func (t durationTest) endLog() {
	fmt.Println("len", len(t.durations))
	fmt.Println("average", time.Duration(t.average()))
	fmt.Println("end")
}

var driverBinPath string = "./driver1/main"
var updateDelay = time.Millisecond * 50
var readDelay = time.Millisecond * 100
var runMode string = "json"
var iteration int64 = 50
var driver1 driver = driver{
	path: "./driver1/libdriver1.so",
}
var durations durationTest

func main() {
	parseFlags()
	fmt.Printf("r: %v, d: %v, i: %v\n", runMode, updateDelay, iteration)
	run()
}

func parseFlags() {
	runPtr := flag.String("r", runMode, "run mode")
	delayPtr := flag.Int64("d", int64(updateDelay), "update delay")
	iterationPtr := flag.Int64("i", iteration, "iteration")
	flag.Parse()
	updateDelay = time.Duration(*delayPtr)
	iteration = *iterationPtr
	runMode = *runPtr
}

func run() {
	switch runMode {
	case "json":
		runIO(decodeJson)
	case "bin":
		runIO(decodeBin)
	case "str":
		runIO(decodeStr)
	case "cgo":
		runDl()
	default:
		log.Fatal(errors.New("invalid runMode"))
	}
}

func newDriverResult(res *C.res) DriverResult {
	return DriverResult{
		Value: float64(res.Value),
		Iteration: int64(res.Iteration),
		CreateTimestamp: int64(res.CreateTimestamp),
		ReadTimestamp: int64(res.ReadTimestamp),
	}
}

func newDriverResultFromString(str string) DriverResult {
	s := strings.Split(str, " ")
	value, err := strconv.ParseFloat(s[0], 64)
	if err != nil {
		log.Fatal(err)
	}
	iteration, err := strconv.ParseInt(s[1], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	createTimestamp, err := strconv.ParseInt(s[2], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	readTimestamp, err := strconv.ParseInt(s[3], 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return DriverResult{
		Value: value,
		Iteration: iteration,
		CreateTimestamp: createTimestamp,
		ReadTimestamp: readTimestamp,
	}
}

func decodeStr(stdout io.ReadCloser, stdin io.WriteCloser) {
	buf := bufio.NewReader(stdout)
	for {
		if _, err := stdin.Write([]byte("r str\n")); err != nil {
			log.Fatal(err)
		}
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			fmt.Println(err)
			break
		} else if err != nil {
			log.Fatal(err)
		}
		ls := string(line)
		data := newDriverResultFromString(ls)
		fmt.Println(data)
		duration := time.Since(time.Unix(0, data.ReadTimestamp))
		fmt.Println(duration)
		durations.addDuration(duration)
		if data.Iteration >= iteration {
			break
		}
		time.Sleep(readDelay)
	}
}

func decodeJson(stdout io.ReadCloser, stdin io.WriteCloser) {
	var data DriverResult
	decoder := json.NewDecoder(stdout)
	for {
		if _, err := stdin.Write([]byte("r json\n")); err != nil {
			log.Fatal(err)
		}
		if err := decoder.Decode(&data); err == io.EOF {
			fmt.Println(err)
			break
		} else if err != nil {
			log.Fatal(err)
		}
		fmt.Println(data)
		duration := time.Since(time.Unix(0, data.ReadTimestamp))
		fmt.Println(duration)
		durations.addDuration(duration)
		if data.Iteration >= iteration {
			break
		}
		time.Sleep(readDelay)
	}
}

func decodeBin(stdout io.ReadCloser, stdin io.WriteCloser) {
	var res C.res
	var data DriverResult
	buf := bufio.NewReader(stdout)
	for {
		if _, err := stdin.Write([]byte("r bin\n")); err != nil {
			log.Fatal(err)
		}
		line, _, err := buf.ReadLine()
		if err != nil {
			log.Fatal(err)
		}
		r := bytes.NewReader(line)
		if err := binary.Read(r, binary.LittleEndian, &res); err == io.ErrUnexpectedEOF {
			fmt.Println(err)
			break
		} else if err == io.EOF {
			fmt.Println(err)
			break
		} else if err != nil {
			log.Fatal(err)
		}
		data = newDriverResult(&res)
		fmt.Println(data)
		duration := time.Since(time.Unix(0, data.ReadTimestamp))
		fmt.Println(duration)
		durations.addDuration(duration)
		if data.Iteration >= iteration {
			break
		}
		time.Sleep(readDelay)
	}
}

func runIO(handler func(stdout io.ReadCloser, stdin io.WriteCloser)) {
	args := []string{
		"-r",
		fmt.Sprintf("-d=%v", int64(updateDelay)),
	}
	fmt.Println(driverBinPath, args)
	cmd := exec.Command(driverBinPath, args...)
	cmd.Stderr = os.Stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if _, err := stdin.Write([]byte("run\n")); err != nil {
		log.Fatal(err)
	}
	handler(stdout, stdin)
	if _, err := stdin.Write([]byte("exit\n")); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	durations.endLog()
}

func runDl() {
	driver1.load()
	fmt.Println(driver1.path, "- loaded")
	driver1.connect(int64(updateDelay))
	defer driver1.disconnect()
	driver1.run()
	for {
		res1 := driver1.read("") // TODO баг в передаче стринговых аргументов
		data := newDriverResult(res1)
		fmt.Println(data)
		duration := time.Since(time.Unix(0, data.ReadTimestamp))
		fmt.Println(duration)
		durations.addDuration(duration)
		if data.Iteration >= iteration {
			break
		}
		time.Sleep(readDelay)
	}
	durations.endLog()
}
