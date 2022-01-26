package rpcx

import (
	"github.com/hitzhangjie/dlv/service/api"
	"time"
)

// rpc ProcessPid

type ProcessPidIn struct {
}

type ProcessPidOut struct {
	Pid int
}

// rpc LastModified

type LastModifiedIn struct {
}

type LastModifiedOut struct {
	Time time.Time
}

// rpc Detach

type DetachIn struct {
	Kill bool
}

type DetachOut struct {
}

// rpc Restart

type RestartIn struct {
	// Position to restart from, if it starts with 'c' it's a checkpoint ID,
	// otherwise it's an event number. Only valid for recorded targets.
	Position string

	// ResetArgs tell whether NewArgs and NewRedirects should take effect.
	ResetArgs bool
	// NewArgs are arguments to launch a new process.  They replace only the
	// argv[1] and later. Argv[0] cannot be changed.
	NewArgs []string

	// When Rerecord is set the target will be rerecorded
	Rerecord bool

	// When Rebuild is set the process will be build again
	Rebuild bool
}

type RestartOut struct {
	DiscardedBreakpoints []api.DiscardedBreakpoint
}

// rpc State

type StateIn struct {
	// If NonBlocking is true State will return immediately even if the target process is running.
	NonBlocking bool
}

type StateOut struct {
	State *api.DebuggerState
}

// rpc Command

type CommandOut struct {
	State api.DebuggerState
}

// rpc GetBufferedTracepoints

type GetBufferedTracepointsIn struct {
}

type GetBufferedTracepointsOut struct {
	TracepointResults []api.TracepointResult
}

// rpc GetBreakpoint

type GetBreakpointIn struct {
	Id   int
	Name string
}

type GetBreakpointOut struct {
	Breakpoint api.Breakpoint
}

// rpc Stacktrace

type StacktraceIn struct {
	Id     int
	Depth  int
	Full   bool
	Defers bool // read deferred functions (equivalent to passing StacktraceReadDefers in Opts)
	Opts   api.StacktraceOptions
	Cfg    *api.LoadConfig
}

type StacktraceOut struct {
	Locations []api.Stackframe
}

// rpc Ancestors

type AncestorsIn struct {
	GoroutineID  int
	NumAncestors int
	Depth        int
}

type AncestorsOut struct {
	Ancestors []api.Ancestor
}

// rpc ListBreakpoints

type ListBreakpointsIn struct {
	All bool
}

type ListBreakpointsOut struct {
	Breakpoints []*api.Breakpoint
}

// rpc CreateBreakpoint

type CreateBreakpointIn struct {
	Breakpoint api.Breakpoint
}

type CreateBreakpointOut struct {
	Breakpoint api.Breakpoint
}

// rpc CreateEBPFTracepoint

type CreateEBPFTracepointIn struct {
	FunctionName string
}

type CreateEBPFTracepointOut struct {
	Breakpoint api.Breakpoint
}

// rpc ClearBreakpoint

type ClearBreakpointIn struct {
	Id   int
	Name string
}

type ClearBreakpointOut struct {
	Breakpoint *api.Breakpoint
}

// rpc ToggleBreakpoint

type ToggleBreakpointIn struct {
	Id   int
	Name string
}

type ToggleBreakpointOut struct {
	Breakpoint *api.Breakpoint
}

// rpc AmendBreakpoint

type AmendBreakpointIn struct {
	Breakpoint api.Breakpoint
}

type AmendBreakpointOut struct {
}

// rpc CancelNext

type CancelNextIn struct {
}

type CancelNextOut struct {
}

// rpc ListThreads

type ListThreadsIn struct {
}

type ListThreadsOut struct {
	Threads []*api.Thread
}

// rpc GetThread

type GetThreadIn struct {
	Id int
}

type GetThreadOut struct {
	Thread *api.Thread
}

// rpc ListPackageVars

type ListPackageVarsIn struct {
	Filter string
	Cfg    api.LoadConfig
}

type ListPackageVarsOut struct {
	Variables []api.Variable
}

// rpc ListRegisters

type ListRegistersIn struct {
	ThreadID  int
	IncludeFp bool
	Scope     *api.EvalScope
}

type ListRegistersOut struct {
	Registers string
	Regs      api.Registers
}

// rpc ListLocalVars

type ListLocalVarsIn struct {
	Scope api.EvalScope
	Cfg   api.LoadConfig
}

type ListLocalVarsOut struct {
	Variables []api.Variable
}

// rpc ListFunctionArgs

type ListFunctionArgsIn struct {
	Scope api.EvalScope
	Cfg   api.LoadConfig
}

type ListFunctionArgsOut struct {
	Args []api.Variable
}

// rpc Eval

type EvalIn struct {
	Scope api.EvalScope
	Expr  string
	Cfg   *api.LoadConfig
}

type EvalOut struct {
	Variable *api.Variable
}

// rpc Set

type SetIn struct {
	Scope  api.EvalScope
	Symbol string
	Value  string
}

type SetOut struct {
}

// rpc ListSources

type ListSourcesIn struct {
	Filter string
}

type ListSourcesOut struct {
	Sources []string
}

// rpc ListFunctions

type ListFunctionsIn struct {
	Filter string
}

type ListFunctionsOut struct {
	Funcs []string
}

// rpc ListTypes

type ListTypesIn struct {
	Filter string
}

type ListTypesOut struct {
	Types []string
}

// rpc ListGoroutines

type ListGoroutinesIn struct {
	Start int
	Count int

	Filters []api.ListGoroutinesFilter
	api.GoroutineGroupingOptions
}

type ListGoroutinesOut struct {
	Goroutines    []*api.Goroutine
	Nextg         int
	Groups        []api.GoroutineGroup
	TooManyGroups bool
}

// rpc AttachedToExistingProcess

type AttachedToExistingProcessIn struct {
}

type AttachedToExistingProcessOut struct {
	Answer bool
}

// rpc FindLocation

type FindLocationIn struct {
	Scope                     api.EvalScope
	Loc                       string
	IncludeNonExecutableLines bool

	// SubstitutePathRules is a slice of source code path substitution rules,
	// the first entry of each pair is the path of a directory as it appears in
	// the executable file (i.e. the location of a source file when the program
	// was compiled), the second entry of each pair is the location of the same
	// directory on the client system.
	SubstitutePathRules [][2]string
}

type FindLocationOut struct {
	Locations []api.Location
}

// rpc Disassemble

type DisassembleIn struct {
	Scope          api.EvalScope
	StartPC, EndPC uint64
	Flavour        api.AssemblyFlavour
}

type DisassembleOut struct {
	Disassemble api.AsmInstructions
}

// rpc Recorded

type RecordedIn struct {
}

type RecordedOut struct {
	Recorded       bool
	TraceDirectory string
}

// rpc Checkpoint

type CheckpointIn struct {
	Where string
}

type CheckpointOut struct {
	ID int
}

// rpc ListCheckpoints

type ListCheckpointsIn struct {
}

type ListCheckpointsOut struct {
	Checkpoints []api.Checkpoint
}

// rpc ClearCheckpoint

type ClearCheckpointIn struct {
	ID int
}

type ClearCheckpointOut struct {
}

// rpc IsMulticlient

type IsMulticlientIn struct {
}

type IsMulticlientOut struct {
	// IsMulticlient returns true if the headless instance was started with --accept-multiclient
	IsMulticlient bool
}

// rpc FunctionReturnLocations

// FunctionReturnLocationsIn holds arguments for the
// FunctionReturnLocationsRPC call. It holds the name of
// the function for which all return locations should be
// given.
type FunctionReturnLocationsIn struct {
	// FnName is the name of the function for which all
	// return locations should be given.
	FnName string
}

// FunctionReturnLocationsOut holds the result of the FunctionReturnLocations
// RPC call. It provides the list of addresses that the given function returns,
// for example with a `RET` instruction or `CALL runtime.deferreturn`.
type FunctionReturnLocationsOut struct {
	// Addrs is the list of all locations where the given function returns.
	Addrs []uint64
}

// rpc ListDynamicLibraries

// ListDynamicLibrariesIn holds the arguments of ListDynamicLibraries
type ListDynamicLibrariesIn struct {
}

// ListDynamicLibrariesOut holds the return values of ListDynamicLibraries
type ListDynamicLibrariesOut struct {
	List []api.Image
}

// rpc ListPackagesBuildInfo

// ListPackagesBuildInfoIn holds the arguments of ListPackages.
type ListPackagesBuildInfoIn struct {
	IncludeFiles bool
}

// ListPackagesBuildInfoOut holds the return values of ListPackages.
type ListPackagesBuildInfoOut struct {
	List []api.PackageBuildInfo
}

// rpc ExamineMemory

// ExamineMemoryIn holds the arguments of ExamineMemory
type ExamineMemoryIn struct {
	Address uint64
	Length  int
}

// ExaminedMemoryOut holds the return values of ExamineMemory
type ExaminedMemoryOut struct {
	Mem            []byte
	IsLittleEndian bool
}

// rpc StopRecording

type StopRecordingIn struct {
}

type StopRecordingOut struct {
}

// rpc DumpStart

type DumpStartIn struct {
	Destination string
}

type DumpStartOut struct {
	State api.DumpState
}

// rpc DumpWait

type DumpWaitIn struct {
	Wait int
}

type DumpWaitOut struct {
	State api.DumpState
}

// rpc DumpCancel

type DumpCancelIn struct {
}

type DumpCancelOut struct {
}

// rpc CreateWatchpoint

type CreateWatchpointIn struct {
	Scope api.EvalScope
	Expr  string
	Type  api.WatchType
}

type CreateWatchpointOut struct {
	*api.Breakpoint
}
