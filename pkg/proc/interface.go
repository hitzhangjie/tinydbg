package proc

import (
	"github.com/hitzhangjie/dlv/pkg/elfwriter"
	"github.com/hitzhangjie/dlv/pkg/proc/internal/ebpf"
)

// Process represents the target of the debugger. This
// target could be a system process, core file, etc.
//
// Implementations of Process are not required to be thread safe and users
// of Process should not assume they are.
// There is one exception to this rule: it is safe to call RequestManualStop
// concurrently with ContinueOnce.
type Process interface {
	// ResumeNotify specifies a channel that will be closed the next time
	// ContinueOnce finishes resuming the target.
	ResumeNotify(chan<- struct{})
	BinInfo() *BinaryInfo
	EntryPoint() (uint64, error)

	// RequestManualStop attempts to stop all the process' threads.
	RequestManualStop() error
	// CheckAndClearManualStopRequest returns true the first time it's called
	// after a call to RequestManualStop.
	CheckAndClearManualStopRequest() bool

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
	Detach(bool) error
	ContinueOnce() (trapthread Thread, stopReason StopReason, err error)

	WriteBreakpoint(*Breakpoint) error
	EraseBreakpoint(*Breakpoint) error

	SupportsBPF() bool
	SetUProbe(string, int64, []ebpf.UProbeArgMap) error
	GetBufferedTracepoints() []ebpf.RawUProbeParams

	// DumpProcessNotes returns ELF core notes describing the process and its threads.
	// Implementing this method is optional.
	DumpProcessNotes(notes []elfwriter.Note, threadDone func()) (bool, []elfwriter.Note, error)
	// MemoryMap returns the memory map of the target process. This method must be implemented if CanDump is true.
	MemoryMap() ([]MemoryMapEntry, error)

	// StartCallInjection notifies the backend that we are about to inject a function call.
	StartCallInjection() (func(), error)
}

// Direction is the direction of execution for the target process.
type Direction int8

const (
	// Forward direction executes the target normally.
	Forward Direction = 0
	// Backward direction executes the target in reverse.
	Backward Direction = 1
)

// Checkpoint is a checkpoint
type Checkpoint struct {
	ID    int
	When  string
	Where string
}
