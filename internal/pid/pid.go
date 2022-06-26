// Sourced from here: https://github.com/struCoder/pidusage (MIT) with a small
// refactor to enable requirements of the platform.

package pid

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"compile-and-run-sandbox/internal/memory"
)

const (
	statTypePS   = "ps"
	statTypeProc = "proc"
)

// SysInfo will record cpu and memory data
type SysInfo struct {
	CPU    float64
	Memory memory.Memory
}

type Statistics struct {
	uTime  float64
	sTime  float64
	cuTime float64
	csTime float64
	start  float64
	rss    float64
	uptime float64
}

var platform = runtime.GOOS
var eol = "\n"

// Linux platform
var clkTck float64 = 100    // default
var PageSize float64 = 4096 // default

var history = map[int]Statistics{}
var historyLock sync.Mutex

var fnMapping = map[string]string{
	"aix":     statTypePS,
	"darwin":  statTypePS,
	"freebsd": statTypePS,
	"sunos":   statTypePS,

	"linux":   statTypeProc,
	"netbsd":  statTypeProc,
	"openbsd": statTypeProc,

	"windows": "windows",
}

func init() {
	if platform == "windows" {
		eol = "\r\n"
	}

	if fnMapping[platform] == statTypeProc {
		initProc()
	}
}

func initProc() {
	if clkTckStdout, err := exec.Command("getconf", "CLK_TCK").Output(); err == nil {
		clkTck = parseFloat(formatStdOut(clkTckStdout, 0)[0])
	}

	if pageSizeStdout, err := exec.Command("getconf", "PAGESIZE").Output(); err == nil {
		PageSize = parseFloat(formatStdOut(pageSizeStdout, 0)[0])
	}
}

func formatStdOut(stdout []byte, splitIndex int) []string {
	infoArr := strings.Split(string(stdout), eol)[splitIndex]
	return strings.Fields(infoArr)
}

func parseFloat(val string) float64 {
	floatVal, _ := strconv.ParseFloat(val, 64)
	return floatVal
}

func statFromPS(pid int) (*SysInfo, error) {
	args := "-o pcpu,rss -p"

	if platform == "aix" {
		args = "-o pcpu,rssize -p"
	}

	stdout, _ := exec.Command("ps", args, strconv.Itoa(pid)).Output()
	ret := formatStdOut(stdout, 1)

	if len(ret) == 0 {
		return nil, errors.New(fmt.Sprintf("Can't find process with this PID: %d", pid))
	}

	return &SysInfo{
		CPU:    parseFloat(ret[0]),
		Memory: memory.Memory(parseFloat(ret[1]) * 1024),
	}, nil
}

func statFromProc(pid int) (*SysInfo, error) {
	uptimeFileBytes, err := ioutil.ReadFile(path.Join("/proc", "uptime"))

	if err != nil {
		return nil, err
	}

	uptime := parseFloat(strings.Split(string(uptimeFileBytes), " ")[0])
	procStatFileBytes, err := ioutil.ReadFile(path.Join("/proc", strconv.Itoa(pid), "stat"))

	if err != nil {
		return nil, err
	}

	splitAfter := strings.SplitAfter(string(procStatFileBytes), ")")

	if len(splitAfter) <= 1 {
		return nil, errors.New(fmt.Sprintf("Can't find process with this PID: %d", pid))
	}

	infos := strings.Split(splitAfter[1], " ")
	pidStatistics := &Statistics{
		uTime:  parseFloat(infos[12]),
		sTime:  parseFloat(infos[13]),
		cuTime: parseFloat(infos[14]),
		csTime: parseFloat(infos[15]),
		start:  parseFloat(infos[20]) / clkTck,
		rss:    parseFloat(infos[22]),
		uptime: uptime,
	}

	historyLock.Lock()
	defer historyLock.Unlock()

	// keep a history of the current processor time for the given pid so
	// that the computed value gets more and more accurate.
	pidHistory := history[pid]

	// use the history value or default back into the float default value
	// of 0.0.
	sTime := pidHistory.sTime
	uTime := pidHistory.uTime

	total := (pidStatistics.sTime - sTime + pidStatistics.uTime - uTime) / clkTck
	seconds := pidStatistics.start - uptime

	if pidHistory.uptime != 0 {
		seconds = uptime - pidHistory.uptime
	}

	seconds = math.Abs(seconds)
	if seconds == 0 {
		seconds = 1
	}

	history[pid] = *pidStatistics

	return &SysInfo{
		CPU:    (total / seconds) * 100,
		Memory: memory.Memory(pidStatistics.rss * PageSize),
	}, nil
}

func stat(pid int, statType string) (*SysInfo, error) {
	switch statType {
	case statTypePS:
		return statFromPS(pid)
	case statTypeProc:
		return statFromProc(pid)
	default:
		return nil, fmt.Errorf("unsupported OS %s", runtime.GOOS)
	}
}

// GetStat will return current system CPU and memory data
func GetStat(pid int) (*SysInfo, error) {
	return stat(pid, fnMapping[platform])
}
