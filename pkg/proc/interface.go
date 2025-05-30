package proc

import (
	"sync"
	"time"

	"github.com/hitzhangjie/tinydbg/pkg/elfwriter"
)

// ProcessGroup is a group of processes that are resumed at the same time.
type ProcessGroup interface {
	ContinueOnce(*ContinueOnceContext) (Thread, StopReason, error)
	StepInstruction(int) error
	Detach(int, bool) error
	Close() error
}

// Process represents the target of the debugger. This
// target could be a system process, core file, etc.
//
// Implementations of Process are not required to be thread safe and users
// of Process should not assume they are.
// There is one exception to this rule: it is safe to call RequestManualStop
// concurrently with ContinueOnce.
type Process interface {
	BinInfo() *BinaryInfo
	EntryPoint() (uint64, error)

	FindThread(threadID int) (Thread, bool)
	ThreadList() []Thread

	Breakpoints() *BreakpointMap

	// Memory returns a memory read/writer for this process's memory.
	Memory() MemoryReadWriter
}

// ProcessInternal holds a set of methods that need to be implemented by a
// Delve backend. Methods in the Process interface are safe to be called by
// clients of the 'proc' library, while all other methods are only called
// directly within 'proc'.
type ProcessInternal interface {
	Process
	// Valid returns true if this Process can be used. When it returns false it
	// also returns an error describing why the Process is invalid (either
	// ErrProcessExited or ErrProcessDetached).
	Valid() (bool, error)

	// RequestManualStop attempts to stop all the process' threads.
	RequestManualStop(cctx *ContinueOnceContext) error

	WriteBreakpoint(*Breakpoint) error
	EraseBreakpoint(*Breakpoint) error

	// DumpProcessNotes returns ELF core notes describing the process and its threads.
	// Implementing this method is optional.
	DumpProcessNotes(notes []elfwriter.Note, threadDone func()) (bool, []elfwriter.Note, error)
	// MemoryMap returns the memory map of the target process. This method must be implemented if CanDump is true.
	MemoryMap() ([]MemoryMapEntry, error)

	// StartCallInjection notifies the backend that we are about to inject a function call.
	StartCallInjection() (func(), error)

	// FollowExec enables (or disables) follow exec mode
	FollowExec(bool) error
}

// ContinueOnceContext is an object passed to ContinueOnce that the backend
// can use to communicate with the target layer.
type ContinueOnceContext struct {
	ResumeChan chan<- struct{}
	StopMu     sync.Mutex
	// manualStopRequested is set if all the threads in the process were
	// signalled to stop as a result of a Halt API call. Used to disambiguate
	// why a thread is found to have stopped.
	manualStopRequested bool
}

// CheckAndClearManualStopRequest will check for a manual
// stop and then clear that state.
func (cctx *ContinueOnceContext) CheckAndClearManualStopRequest() bool {
	cctx.StopMu.Lock()
	defer cctx.StopMu.Unlock()
	msr := cctx.manualStopRequested
	cctx.manualStopRequested = false
	return msr
}

func (cctx *ContinueOnceContext) GetManualStopRequested() bool {
	cctx.StopMu.Lock()
	defer cctx.StopMu.Unlock()
	return cctx.manualStopRequested
}

// WaitFor is passed to native.Attach and gdbserver.LLDBAttach to wait for a
// process to start before attaching.
type WaitFor struct {
	Name               string
	Interval, Duration time.Duration
}
