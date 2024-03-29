package native

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"runtime"
	"strings"
	"sync"

	sys "golang.org/x/sys/unix"

	"github.com/hitzhangjie/dlv/pkg/proc"
	"github.com/hitzhangjie/dlv/pkg/proc/internal/ebpf"
)

// Process statuses
const (
	statusSleeping  = 'S'
	statusRunning   = 'R'
	statusTraceStop = 't'
	statusZombie    = 'Z'

	// Kernel 2.6 has TraceStop as T
	// TODO(derekparker) Since this means something different based on the
	// version of the kernel ('T' is job control stop on modern 3.x+ kernels) we
	// may want to differentiate at some point.
	statusTraceStopT = 'T'

	personalityGetPersonality = 0xffffffff // argument to pass to personality syscall to get the current personality
	_ADDR_NO_RANDOMIZE        = 0x0040000  // ADDR_NO_RANDOMIZE linux constant
)

// Process represents all of the information the debugger
// is holding onto regarding the process we are debugging.
type nativeProcess struct {
	bi *proc.BinaryInfo

	pid int // Process Pid

	// Breakpoint table, holds information on breakpoints.
	// Maps instruction address to Breakpoint struct.
	breakpoints proc.BreakpointMap

	// List of threads mapped as such: pid -> *Thread
	threads map[int]*nativeThread

	// Thread used to read and write memory
	memthread *nativeThread

	os             *osProcessDetails
	firstStart     bool
	resumeChan     chan<- struct{}
	ptraceChan     chan func()
	ptraceDoneChan chan interface{}
	childProcess   bool       // this process was launched, not attached to
	stopMu         sync.Mutex // protects manualStopRequested
	// manualStopRequested is set if all the threads in the process were
	// signalled to stop as a result of a Halt API call. Used to disambiguate
	// why a thread is found to have stopped.
	manualStopRequested bool

	iscgo bool

	exited, detached bool
}

// newProcess returns an initialized Process struct. Before returning,
// it will also launch a goroutine in order to handle ptrace(2)
// functions. For more information, see the documentation on
// `handlePtraceFuncs`.
func newProcess(pid int) *nativeProcess {
	dbp := &nativeProcess{
		pid:            pid,
		threads:        make(map[int]*nativeThread),
		breakpoints:    proc.NewBreakpointMap(),
		firstStart:     true,
		os:             new(osProcessDetails),
		ptraceChan:     make(chan func()),
		ptraceDoneChan: make(chan interface{}),
		bi:             proc.NewBinaryInfo(runtime.GOOS, runtime.GOARCH),
	}
	go dbp.handlePtraceFuncs()
	return dbp
}

// initialize will ensure that all relevant information is loaded
// so the process is ready to be debugged.
func (dbp *nativeProcess) initialize(path string) (*proc.Target, error) {
	if err := initialize(dbp); err != nil {
		return nil, err
	}
	if err := dbp.updateThreadList(); err != nil {
		return nil, err
	}
	stopReason := proc.StopLaunched
	if !dbp.childProcess {
		stopReason = proc.StopAttached
	}
	return proc.NewTarget(dbp, dbp.pid, dbp.memthread, proc.NewTargetConfig{
		Path:                path,
		DisableAsyncPreempt: false,
		StopReason:          stopReason,
		CanDump:             true})
}

func initialize(dbp *nativeProcess) error {
	comm, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/comm", dbp.pid))
	if err == nil {
		// removes newline character
		comm = bytes.TrimSuffix(comm, []byte("\n"))
	}

	if comm == nil || len(comm) <= 0 {
		stat, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", dbp.pid))
		if err != nil {
			return fmt.Errorf("could not read proc stat: %v", err)
		}
		expr := fmt.Sprintf("%d\\s*\\((.*)\\)", dbp.pid)
		rexp, err := regexp.Compile(expr)
		if err != nil {
			return fmt.Errorf("regexp compile error: %v", err)
		}
		match := rexp.FindSubmatch(stat)
		if match == nil {
			return fmt.Errorf("no match found using regexp '%s' in /proc/%d/stat", expr, dbp.pid)
		}
		comm = match[1]
	}
	dbp.os.comm = strings.ReplaceAll(string(comm), "%", "%%")

	return nil
}

// BinInfo will return the binary info struct associated with this process.
func (dbp *nativeProcess) BinInfo() *proc.BinaryInfo {
	return dbp.bi
}

// StartCallInjection notifies the backend that we are about to inject a function call.
func (dbp *nativeProcess) StartCallInjection() (func(), error) { return func() {}, nil }

// Detach from the process being debugged, optionally killing it.
func (dbp *nativeProcess) Detach(kill bool) (err error) {
	if dbp.exited {
		return nil
	}
	if kill && dbp.childProcess {
		err := dbp.kill()
		if err != nil {
			return err
		}
		dbp.bi.Close()
		return nil
	}
	dbp.execPtraceFunc(func() {
		err = dbp.detach(kill)
		if err != nil {
			return
		}
		if kill {
			err = killProcess(dbp.pid)
		}
	})
	dbp.detached = true
	dbp.postExit()
	return
}

func killProcess(pid int) error {
	return sys.Kill(pid, sys.SIGINT)
}

// Valid returns whether the process is still attached to and
// has not exited.
func (dbp *nativeProcess) Valid() (bool, error) {
	if dbp.detached {
		return false, proc.ErrProcessDetached
	}
	if dbp.exited {
		return false, proc.ErrProcessExited{Pid: dbp.pid}
	}
	return true, nil
}

// ResumeNotify specifies a channel that will be closed the next time
// ContinueOnce finishes resuming the target.
func (dbp *nativeProcess) ResumeNotify(ch chan<- struct{}) {
	dbp.resumeChan = ch
}

// ThreadList returns a list of threads in the process.
func (dbp *nativeProcess) ThreadList() []proc.Thread {
	r := make([]proc.Thread, 0, len(dbp.threads))
	for _, v := range dbp.threads {
		r = append(r, v)
	}
	return r
}

// FindThread attempts to find the thread with the specified ID.
func (dbp *nativeProcess) FindThread(threadID int) (proc.Thread, bool) {
	th, ok := dbp.threads[threadID]
	return th, ok
}

// Memory returns the process memory.
func (dbp *nativeProcess) Memory() proc.MemoryReadWriter {
	return dbp.memthread
}

// Breakpoints returns a list of breakpoints currently set.
func (dbp *nativeProcess) Breakpoints() *proc.BreakpointMap {
	return &dbp.breakpoints
}

// RequestManualStop sets the `manualStopRequested` flag and
// sends SIGSTOP to all threads.
func (dbp *nativeProcess) RequestManualStop() error {
	if dbp.exited {
		return proc.ErrProcessExited{Pid: dbp.pid}
	}
	dbp.stopMu.Lock()
	defer dbp.stopMu.Unlock()
	dbp.manualStopRequested = true
	return dbp.requestManualStop()
}

// CheckAndClearManualStopRequest checks if a manual stop has
// been requested, and then clears that state.
func (dbp *nativeProcess) CheckAndClearManualStopRequest() bool {
	dbp.stopMu.Lock()
	defer dbp.stopMu.Unlock()

	msr := dbp.manualStopRequested
	dbp.manualStopRequested = false

	return msr
}

func (dbp *nativeProcess) WriteBreakpoint(bp *proc.Breakpoint) error {
	if bp.WatchType != 0 {
		for _, thread := range dbp.threads {
			err := thread.writeHardwareBreakpoint(bp.Addr, bp.WatchType, bp.HWBreakIndex)
			if err != nil {
				return err
			}
		}
		return nil
	}

	bp.Orig = make([]byte, dbp.bi.Arch.BreakpointSize())
	_, err := dbp.memthread.ReadMemory(bp.Orig, bp.Addr)
	if err != nil {
		return err
	}
	return dbp.writeSoftwareBreakpoint(dbp.memthread, bp.Addr)
}

func (dbp *nativeProcess) EraseBreakpoint(bp *proc.Breakpoint) error {
	if bp.WatchType != 0 {
		for _, thread := range dbp.threads {
			err := thread.clearHardwareBreakpoint(bp.Addr, bp.WatchType, bp.HWBreakIndex)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return dbp.memthread.clearSoftwareBreakpoint(bp)
}

// ContinueOnce will continue the target until it stops.
// This could be the result of a breakpoint or signal.
func (dbp *nativeProcess) ContinueOnce() (proc.Thread, proc.StopReason, error) {
	if dbp.exited {
		return nil, proc.StopExited, proc.ErrProcessExited{Pid: dbp.pid}
	}

	for {

		if err := dbp.resume(); err != nil {
			return nil, proc.StopUnknown, err
		}

		for _, th := range dbp.threads {
			th.CurrentBreakpoint.Clear()
		}

		if dbp.resumeChan != nil {
			close(dbp.resumeChan)
			dbp.resumeChan = nil
		}

		trapthread, err := dbp.trapWait(-1)
		if err != nil {
			return nil, proc.StopUnknown, err
		}
		trapthread, err = dbp.stop(trapthread)
		if err != nil {
			return nil, proc.StopUnknown, err
		}
		if trapthread != nil {
			dbp.memthread = trapthread
			return trapthread, proc.StopUnknown, nil
		}
	}
}

// FindBreakpoint finds the breakpoint for the given pc.
func (dbp *nativeProcess) FindBreakpoint(pc uint64, adjustPC bool) (*proc.Breakpoint, bool) {
	if adjustPC {
		// Check to see if address is past the breakpoint, (i.e. breakpoint was hit).
		if bp, ok := dbp.breakpoints.M[pc-uint64(dbp.bi.Arch.BreakpointSize())]; ok {
			return bp, true
		}
	}
	// Directly use addr to lookup breakpoint.
	if bp, ok := dbp.breakpoints.M[pc]; ok {
		return bp, true
	}
	return nil, false
}

func (dbp *nativeProcess) handlePtraceFuncs() {
	// We must ensure here that we are running on the same thread during
	// while invoking the ptrace(2) syscall. This is due to the fact that ptrace(2) expects
	// all commands after PTRACE_ATTACH to come from the same thread.
	runtime.LockOSThread()

	for fn := range dbp.ptraceChan {
		fn()
		dbp.ptraceDoneChan <- nil
	}
}

func (dbp *nativeProcess) execPtraceFunc(fn func()) {
	dbp.ptraceChan <- fn
	<-dbp.ptraceDoneChan
}

func (dbp *nativeProcess) postExit() {
	dbp.exited = true
	close(dbp.ptraceChan)
	close(dbp.ptraceDoneChan)
	dbp.bi.Close()
	dbp.os.Close()
}

func (dbp *nativeProcess) writeSoftwareBreakpoint(thread *nativeThread, addr uint64) error {
	_, err := thread.WriteMemory(addr, dbp.bi.Arch.BreakpointInstruction())
	return err
}

// linux && amd64 && cgo && go1.16
func (dbp *nativeProcess) SupportsBPF() bool {
	return true
}

// osProcessDetails contains Linux specific process details.
type osProcessDetails struct {
	comm string

	ebpf *ebpf.EBPFContext
}

func (os *osProcessDetails) Close() {
	if os.ebpf != nil {
		os.ebpf.Close()
	}
}
