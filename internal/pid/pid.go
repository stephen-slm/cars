// Sourced from here: https://github.com/struCoder/pidusage (MIT) with a small
// refactor to enable requirements of the platform.
//
// PROC Reference - https://man7.org/linux/man-pages/man5/proc.5.html

package pid

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"compile-and-run-sandbox/internal/memory"
)

const (
	statTypePS   = "ps"
	statTypeProc = "proc"
)

// SysInfo will record cpu and memory data
type SysInfo struct {
	CPU    int64
	Memory memory.Memory
}

type ProcPidState string

const DEBUG = false

const (
	ProcPidRunning             ProcPidState = "R" // R  Running
	ProcPidSleeping            ProcPidState = "S" // S  Sleeping in an interruptible wait
	ProcPidWaiting             ProcPidState = "D" // D  Waiting in uninterruptible disk sleep
	ProcPidZombie              ProcPidState = "Z" // Z  Zombie
	ProcPidStopped             ProcPidState = "T" // T  Stopped (on a signal) or (before Linux 2.6.33) trace stopped
	ProcPidTracing             ProcPidState = "t" // t  Tracing stop (Linux 2.6.33 onward)
	ProcPidPaging              ProcPidState = "W" // W  Paging (only before Linux 2.6.0)
	ProcPidDeadLinuxNew        ProcPidState = "X" // X  Dead (from Linux 2.6.0 onward)
	ProcPidDeadLinux26633To333 ProcPidState = "x" // x  Dead (Linux 2.6.33 to 3.13 only)
	ProcPidWakeKill            ProcPidState = "K" // K  Wakekill (Linux 2.6.33 to 3.13 only)
	ProcPidWaking              ProcPidState = "W" // W  Waking (Linux 2.6.33 to 3.13 only)
	ProcPidPark                ProcPidState = "P" // P  Parked (Linux 3.9 to 3.13 only)
)

type ProcPidStatistics struct {
	// The process ID.
	Pid int64 `json:"pid"`
	// The filename of the executable, in parentheses. Strings longer than
	// TASK_COMM_LEN (16) characters (including the terminating null byte) are
	// silently truncated.  This is visible whether or not the executable is
	// swapped out.
	Comm string `json:"comm"`
	//  One of the following characters, indicating process State
	State ProcPidState `json:"state"`
	//  The PID of the parent of this process.
	Ppid int64 `json:"ppid"`
	// The process group ID of the process.
	Pgrp int64 `json:"pgrp"`
	// The Session ID of the process.
	Session int64 `json:"session"`
	// The controlling terminal of the process.  (The minor device number is
	// contained in the combination of bits 31 to 20 and 7 to 0; the major
	// device number is in bits 15 to 8.)
	TtyNr int64 `json:"ttyNr"`
	// The ID of the foreground process group of the controlling terminal of the
	// process.
	Tpgid int64 `json:"tpgid"`
	// The kernel flags word of the process.  For bit meanings, see the PF_*
	// defines in the Linux kernel source file include/linux/sched.h.  Details
	//  depend on the kernel version.
	//
	// The format for this field was %lu before Linux 2.6.
	Flags int64 `json:"flags"`
	// The number of minor faults the process has made which have not required
	// loading a memory page from disk.
	Minflt int64 `json:"minflt"`
	// The number of minor faults that the process's waited-for children have
	// made.
	Cminflt int64 `json:"cminflt"`
	// The number of major faults the process has made which have required
	// loading a memory page from disk.
	Majflt int64 `json:"majflt"`
	// The number of major faults that the process's waited-for children have
	// made.
	Cmajflt int64 `json:"cmajflt"`
	// Amount of time that this process has been scheduled in user mode,
	// measured in clock ticks (divide by sysconf(_SC_CLK_TCK)).  This includes
	// guest time, GuestTime (time spent running a virtual CPU, see below), so
	// that applications that are not aware of the guest time field do not lose
	// that time from their calculations.
	Utime int64 `json:"utime"`
	// Amount of time that this process has been scheduled in kernel mode,
	// measured in clock ticks (divide by sysconf(_SC_CLK_TCK)).
	Stime int64 `json:"stime"`
	// Amount of time that this process's waited-for children have been
	// scheduled in user mode, measured in clock ticks (divide by
	// sysconf(_SC_CLK_TCK)). (See also times(2).)  This includes guest time,
	// CguestTime (time spent running a virtual CPU, see below).
	Cutime int64 `json:"cutime"`
	// Amount of time that this process's waited-for children have been
	// scheduled in kernel mode, measured in clock ticks (divide by
	// sysconf(_SC_CLK_TCK)).
	Cstime int64 `json:"cstime"`
	// (Explanation for Linux 2.6) For processes running a real-time scheduling
	// Policy (Policy below; see sched_setscheduler(2)), this is the negated
	// scheduling Priority, minus one; that is, a number in the range -2 to -100,
	// corresponding to real-time priorities 1 to 99.  For processes running
	// under a non-real-time scheduling Policy, this is the raw Nice value
	// (setpriority(2)) as represented in the kernel.  The kernel stores Nice
	// values as numbers in the range 0 (high) to 39 (low), corresponding to the
	// user-visible Nice range of -20 to 19.
	//
	// Before Linux 2.6, this was a scaled value based on
	// the scheduler weighting given to this process.
	Priority int64 `json:"priority"`
	// The Nice value (see setpriority(2)), a value in the range 19 (low
	// priority) to -20 (high priority).
	Nice int64 `json:"nice"`
	// Number of threads in this process (since Linux 2.6).  Before kernel 2.6,
	// this field was hard coded to 0 as a placeholder for an earlier removed
	// field.
	NumThreads int64 `json:"numThreads"`
	// The time in jiffies before the next SIGALRM is sent to the process due to
	// an interval timer.  Since kernel 2.6.17, this field is no longer
	// maintained, and is hard coded as 0.
	Itrealvalue int64 `json:"itrealvalue"`
	// The time the process started after system boot.  In kernels before Linux
	// 2.6, this value was expressed in jiffies.  Since Linux 2.6, the value is
	// expressed in clock ticks (divide by sysconf(_SC_CLK_TCK)).
	Starttime int64 `json:"starttime"`
	// Virtual memory size in bytes.
	Vsize int64 `json:"vsize"`
	// Resident Set Size: number of pages the process has in real memory.  This
	// is just the pages which count toward text, data, or stack space.
	// This does not include pages which have not been demand-loaded in, or
	// which are swapped out.  This value is inaccurate; see /proc/[pid]/statm
	// below.
	Rss int64 `json:"rss"`
	// Current soft limit in bytes on the rss of the process; see the
	// description of RLIMIT_RSS in getrlimit(2).
	Rsslim int64 `json:"rsslim"`
	// The address above which program text can run.
	Startcode int64 `json:"startcode"`
	// The address below which program text can run.
	Endcode int64 `json:"endcode"`
	// The address of the start (i.e., bottom) of the stack.
	Startstack int64 `json:"startstack"`
	// The current value of ESP (stack pointer), as found in the kernel stack
	// page for the process.
	Kstkesp int64 `json:"kstkesp"`
	// The current EIP (instruction pointer).
	Kstkeip int64 `json:"kstkeip"`
	// The bitmap of pending signals, displayed as a decimal number.  Obsolete,
	// because it does not provide information on real-time signals; use
	// /proc/[pid]/status instead.
	Signal int64 `json:"signal"`
	// The bitmap of Blocked signals, displayed as a decimal number.  Obsolete,
	// because it does not provide information on real-time signals; use
	// /proc/[pid]/status instead.
	Blocked int64 `json:"blocked"`
	// The bitmap of ignored signals, displayed as a decimal number.  Obsolete,
	// because it does not provide information on real-time signals; use
	// /proc/[pid]/status instead.
	Sigignore int64 `json:"sigignore"`
	// The bitmap of caught signals, displayed as a decimal number.  Obsolete,
	// because it does not provide information on real-time signals; use
	// /proc/[pid]/status instead.
	Sigcatch int64 `json:"sigcatch"`
	// This is the "channel" in which the process is waiting.  It is the address
	// of a location in the kernel where the process is sleeping.  The
	// corresponding symbolic name can be found in /proc/[pid]/wchan.
	Wchan int64 `json:"wchan"`
	// Number of pages swapped (not maintained).
	Nswap int64 `json:"nswap"`
	// Cumulative nswap for child processes (not maintained).
	Cnswap int64 `json:"cnswap"`
	// Signal to be sent to parent when we die.
	ExitSignal int64 `json:"exitSignal"`
	// CPU number last executed on.
	Processor int64 `json:"processor"`
	// Real-time scheduling priority, a number in the range 1 to 99 for
	// processes scheduled under a real- time Policy, or 0, for non-real-time
	// processes (see sched_setscheduler(2)).
	RtPriority int64 `json:"rtPriority"`
	// Scheduling Policy (see sched_setscheduler(2)). Decode using the SCHED_*
	// constants in linux/sched.h.
	Policy int64 `json:"policy"`
	// The format for this field was int64 before Linux Aggregated block I/O
	// delays, measured in clock ticks (centiseconds).
	DelayacctBlkioTicks int64 `json:"delayacctBlkioTicks"`
	// Guest time of the process (time spent running a virtual CPU for a guest
	// operating system), measured in clock ticks (divide by sysconf(_SC_CLK_TCK)).
	GuestTime int64 `json:"guestTime"`
	// Guest time of the process's children, measured in clock ticks (divide
	// by sysconf(_SC_CLK_TCK)).
	CguestTime int64 `json:"cguestTime"`
	// Address above which program initialized and uninitialized (BSS) data are
	// placed.
	StartData int64 `json:"startData"`
	// Address below which program initialized and uninitialized (BSS) data are
	// placed.
	EndData int64 `json:"endData"`
	// Address above which program heap can be expanded with brk(2).
	StartBrk int64 `json:"startBrk"`
	// Address above which program command-line arguments (argv) are placed.
	ArgStart int64 `json:"argStart"`
	// Address below program command-line arguments (argv) are placed.
	ArgEnd int64 `json:"argEnd"`
	// Address above which program environment is placed.
	EnvStart int64 `json:"envStart"`
	// Address below which program environment is placed.
	EnvEnd int64 `json:"envEnd"`
	// The thread's exit status in the form reported by
	// waitpid(2).
	ExitCode int64 `json:"exitCode"`
}

var platform = runtime.GOOS
var eol = "\n"

// Linux platform
// nolint // allow for future development
var clkTck int64 = 100    // default
var PageSize int64 = 4096 // default

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
		log.Debug().Str("CLK_TCK", string(clkTckStdout)).Msg("getconf")
		clkTck = parseInt64(formatStdOut(clkTckStdout, 0)[0])
	}

	if pageSizeStdout, err := exec.Command("getconf", "PAGESIZE").Output(); err == nil {
		log.Debug().Str("PAGESIZE", string(pageSizeStdout)).Msg("getconf")
		PageSize = parseInt64(formatStdOut(pageSizeStdout, 0)[0])
	}
}

func formatStdOut(stdout []byte, splitIndex int) []string {
	infoArr := strings.Split(string(stdout), eol)[splitIndex]
	return strings.Fields(infoArr)
}

func parseInt64(val string) int64 {
	value, _ := strconv.ParseInt(val, 10, 0)
	return value
}

func statFromProc(pid int) (*SysInfo, error) {
	procStatFileBytes, err := os.ReadFile(path.Join("/proc", strconv.Itoa(pid), "stat"))

	if err != nil {
		return nil, err
	}

	infos := strings.Split(string(procStatFileBytes), " ")

	pidStatistics := &ProcPidStatistics{
		Pid:                 parseInt64(infos[0]),
		Comm:                infos[1],
		State:               ProcPidState(infos[2]),
		Ppid:                parseInt64(infos[3]),
		Pgrp:                parseInt64(infos[4]),
		Session:             parseInt64(infos[5]),
		TtyNr:               parseInt64(infos[6]),
		Tpgid:               parseInt64(infos[7]),
		Flags:               parseInt64(infos[8]),
		Minflt:              parseInt64(infos[9]),
		Cminflt:             parseInt64(infos[10]),
		Majflt:              parseInt64(infos[11]),
		Cmajflt:             parseInt64(infos[12]),
		Utime:               parseInt64(infos[13]),
		Stime:               parseInt64(infos[14]),
		Cutime:              parseInt64(infos[15]),
		Cstime:              parseInt64(infos[16]),
		Priority:            parseInt64(infos[17]),
		Nice:                parseInt64(infos[18]),
		NumThreads:          parseInt64(infos[19]),
		Itrealvalue:         parseInt64(infos[20]),
		Starttime:           parseInt64(infos[21]),
		Vsize:               parseInt64(infos[22]),
		Rss:                 parseInt64(infos[23]),
		Rsslim:              parseInt64(infos[24]),
		Startcode:           parseInt64(infos[25]),
		Endcode:             parseInt64(infos[26]),
		Startstack:          parseInt64(infos[27]),
		Kstkesp:             parseInt64(infos[28]),
		Kstkeip:             parseInt64(infos[29]),
		Signal:              parseInt64(infos[30]),
		Blocked:             parseInt64(infos[31]),
		Sigignore:           parseInt64(infos[32]),
		Sigcatch:            parseInt64(infos[33]),
		Wchan:               parseInt64(infos[34]),
		Nswap:               parseInt64(infos[35]),
		Cnswap:              parseInt64(infos[36]),
		ExitSignal:          parseInt64(infos[37]),
		Processor:           parseInt64(infos[38]),
		RtPriority:          parseInt64(infos[39]),
		Policy:              parseInt64(infos[40]),
		DelayacctBlkioTicks: parseInt64(infos[41]),
		GuestTime:           parseInt64(infos[42]),
		CguestTime:          parseInt64(infos[43]),
		StartData:           parseInt64(infos[44]),
		EndData:             parseInt64(infos[45]),
		StartBrk:            parseInt64(infos[46]),
		ArgStart:            parseInt64(infos[47]),
		ArgEnd:              parseInt64(infos[48]),
		EnvStart:            parseInt64(infos[49]),
		EnvEnd:              parseInt64(infos[50]),
		ExitCode:            parseInt64(infos[51]),
	}

	if pidStatistics.Rss >= 0 && DEBUG {
		log.Debug().
			Interface("pid-statistics", pidStatistics).
			Msg("pid states")
	}

	return &SysInfo{
		CPU:    0,
		Memory: memory.Memory(pidStatistics.Rss * PageSize),
	}, nil
}

func stat(pid int, statType string) (*SysInfo, error) {
	switch statType {
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

func StreamPid(done <-chan any, pid int) <-chan *SysInfo {
	value := make(chan *SysInfo)

	go func() {
		defer close(value)

		for {
			state, err := GetStat(pid)

			if err != nil {
				log.Err(err).
					Int("pid", pid).
					Msg("failed to get stats for pid")

				select {
				case <-done:
					return
				default:
					continue
				}
			}

			select {
			case <-done:
				return
			case value <- state:
			}

			time.Sleep(time.Millisecond * 10)
		}
	}()

	return value
}
