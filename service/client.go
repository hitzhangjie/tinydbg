package service

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/hitzhangjie/dlv/service/api"
)

// Client represents a client of a debugger service. All client methods are synchronous.
type Client interface {
	// ProcessPid Returns the pid of the process we are debugging.
	ProcessPid() int

	// LastModified returns the time that the process' executable was modified.
	LastModified() time.Time

	// Detach detaches the debugger, optionally killing the process.
	Detach(killProcess bool) error

	// Restarts program. Set true if you want to rebuild the process we are debugging.
	Restart(rebuild bool) ([]api.DiscardedBreakpoint, error)

	// GetState returns the current debugger state.
	GetState() (*api.DebuggerState, error)
	// GetStateNonBlocking returns the current debugger state, returning immediately if the target is already running.
	GetStateNonBlocking() (*api.DebuggerState, error)

	// Continue resumes process execution.
	Continue() <-chan *api.DebuggerState
	// DirectionCongruentContinue resumes process execution, if a next, step or stepout operation is in progress it will resume execution.
	DirectionCongruentContinue() <-chan *api.DebuggerState
	// Next continues to the next source line, not entering function calls.
	Next() (*api.DebuggerState, error)
	// Step continues to the next source line, entering function calls.
	Step() (*api.DebuggerState, error)
	// StepOut continues to the return address of the current function.
	StepOut() (*api.DebuggerState, error)
	// Call resumes process execution while making a function call.
	Call(goroutineID int, expr string, unsafe bool) (*api.DebuggerState, error)

	// SingleStep will step a single cpu instruction.
	StepInstruction() (*api.DebuggerState, error)
	// SwitchThread switches the current thread context.
	SwitchThread(threadID int) (*api.DebuggerState, error)
	// SwitchGoroutine switches the current goroutine (and the current thread as well)
	SwitchGoroutine(goroutineID int) (*api.DebuggerState, error)
	// Halt suspends the process.
	Halt() (*api.DebuggerState, error)

	// GetBreakpoint gets a breakpoint by ID.
	GetBreakpoint(id int) (*api.Breakpoint, error)
	// GetBreakpointByName gets a breakpoint by name.
	GetBreakpointByName(name string) (*api.Breakpoint, error)
	// CreateBreakpoint creates a new breakpoint.
	CreateBreakpoint(*api.Breakpoint) (*api.Breakpoint, error)
	// CreateWatchpoint creates a new watchpoint.
	CreateWatchpoint(api.EvalScope, string, api.WatchType) (*api.Breakpoint, error)
	// ListBreakpoints gets all breakpoints.
	ListBreakpoints(bool) ([]*api.Breakpoint, error)
	// ClearBreakpoint deletes a breakpoint by ID.
	ClearBreakpoint(id int) (*api.Breakpoint, error)
	// ClearBreakpointByName deletes a breakpoint by name
	ClearBreakpointByName(name string) (*api.Breakpoint, error)
	// ToggleBreakpoint toggles on or off a breakpoint by ID.
	ToggleBreakpoint(id int) (*api.Breakpoint, error)
	// ToggleBreakpointByName toggles on or off a breakpoint by name.
	ToggleBreakpointByName(name string) (*api.Breakpoint, error)
	// Allows user to update an existing breakpoint for example to change the information
	// retrieved when the breakpoint is hit or to change, add or remove the break condition
	AmendBreakpoint(*api.Breakpoint) error
	// Cancels a Next or Step call that was interrupted by a manual stop or by another breakpoint
	CancelNext() error

	// ListThreads lists all threads.
	ListThreads() ([]*api.Thread, error)
	// GetThread gets a thread by its ID.
	GetThread(id int) (*api.Thread, error)

	// ListPackageVariables lists all package variables in the context of the current thread.
	ListPackageVariables(filter string, cfg api.LoadConfig) ([]api.Variable, error)
	// EvalVariable returns a variable in the context of the current thread.
	EvalVariable(scope api.EvalScope, symbol string, cfg api.LoadConfig) (*api.Variable, error)

	// SetVariable sets the value of a variable
	SetVariable(scope api.EvalScope, symbol, value string) error

	// ListSources lists all source files in the process matching filter.
	ListSources(filter string) ([]string, error)
	// ListFunctions lists all functions in the process matching filter.
	ListFunctions(filter string) ([]string, error)
	// ListTypes lists all types in the process matching filter.
	ListTypes(filter string) ([]string, error)
	// ListLocals lists all local variables in scope.
	ListLocalVariables(scope api.EvalScope, cfg api.LoadConfig) ([]api.Variable, error)
	// ListFunctionArgs lists all arguments to the current function.
	ListFunctionArgs(scope api.EvalScope, cfg api.LoadConfig) ([]api.Variable, error)
	// ListThreadRegisters lists registers and their values, for the given thread.
	ListThreadRegisters(threadID int, includeFp bool) (api.Registers, error)
	// ListScopeRegisters lists registers and their values, for the given scope.
	ListScopeRegisters(scope api.EvalScope, includeFp bool) (api.Registers, error)

	// ListGoroutines lists all goroutines.
	ListGoroutines(start, count int) ([]*api.Goroutine, int, error)
	// ListGoroutinesWithFilter lists goroutines matching the filters
	ListGoroutinesWithFilter(start, count int, filters []api.ListGoroutinesFilter, group *api.GoroutineGroupingOptions) ([]*api.Goroutine, []api.GoroutineGroup, int, bool, error)

	// Returns stacktrace
	Stacktrace(goroutineID int, depth int, opts api.StacktraceOptions, cfg *api.LoadConfig) ([]api.Stackframe, error)

	// Returns ancestor stacktraces
	Ancestors(goroutineID int, numAncestors int, depth int) ([]api.Ancestor, error)

	// Returns whether we attached to a running process or not
	AttachedToExistingProcess() bool

	// Returns concrete location information described by a location expression
	// loc ::= <filename>:<line> | <function>[:<line>] | /<regex>/ | (+|-)<offset> | <line> | *<address>
	// * <filename> can be the full path of a file or just a suffix
	// * <function> ::= <package>.<receiver type>.<name> | <package>.(*<receiver type>).<name> | <receiver type>.<name> | <package>.<name> | (*<receiver type>).<name> | <name>
	// * <function> must be unambiguous
	// * /<regex>/ will return a location for each function matched by regex
	// * +<offset> returns a location for the line that is <offset> lines after the current line
	// * -<offset> returns a location for the line that is <offset> lines before the current line
	// * <line> returns a location for a line in the current file
	// * *<address> returns the location corresponding to the specified address
	// NOTE: this function does not actually set breakpoints.
	// If findInstruction is true FindLocation will only return locations that correspond to instructions.
	FindLocation(scope api.EvalScope, loc string, findInstruction bool, substitutePathRules [][2]string) ([]api.Location, error)

	// Disassemble code between startPC and endPC
	DisassembleRange(scope api.EvalScope, startPC, endPC uint64, flavour api.AssemblyFlavour) (api.AsmInstructions, error)
	// Disassemble code of the function containing PC
	DisassemblePC(scope api.EvalScope, pc uint64, flavour api.AssemblyFlavour) (api.AsmInstructions, error)

	// SetReturnValuesLoadConfig sets the load configuration for return values.
	SetReturnValuesLoadConfig(*api.LoadConfig)

	// FunctionReturnLocations return locations when function `fnName` returns
	FunctionReturnLocations(fnName string) ([]uint64, error)

	// IsMulticlien returns true if the headless instance is multiclient.
	IsMulticlient() bool

	// ListDynamicLibraries returns a list of loaded dynamic libraries.
	ListDynamicLibraries() ([]api.Image, error)

	// ExamineMemory returns the raw memory stored at the given address.
	// The amount of data to be read is specified by length which must be less than or equal to 1000.
	// This function will return an error if it reads less than `length` bytes.
	ExamineMemory(address uint64, length int) ([]byte, bool, error)

	// CoreDumpStart starts creating a core dump to the specified file
	CoreDumpStart(dest string) (api.DumpState, error)
	// CoreDumpWait waits for the core dump to finish, or for the specified amount of milliseconds
	CoreDumpWait(msec int) api.DumpState
	// CoreDumpCancel cancels a core dump in progress
	CoreDumpCancel() error

	// Disconnect closes the connection to the server without sending a Detach request first.
	// If cont is true a continue command will be sent instead.
	Disconnect(cont bool) error

	// CallAPI allows calling an arbitrary rpcv2 method (used by starlark bindings)
	CallAPI(method string, args, reply interface{}) error
}

// Ensure the implementation satisfies the interface.
var _ Client = &RPCClient{}

// RPCClient is a RPC service.Client.
type RPCClient struct {
	client        *rpc.Client
	retValLoadCfg *api.LoadConfig
}

// NewClient creates a new RPCClient.
func NewClient(addr string) *RPCClient {
	client, err := jsonrpc.Dial("tcp", addr)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	return &RPCClient{client: client}
}

// NewClientFromConn creates a new RPCClient from the given connection.
func NewClientFromConn(conn net.Conn) *RPCClient {
	return &RPCClient{client: jsonrpc.NewClient(conn)}
}

func (c *RPCClient) ProcessPid() int {
	out := new(ProcessPidOut)
	c.call("ProcessPid", ProcessPidIn{}, out)
	return out.Pid
}

func (c *RPCClient) LastModified() time.Time {
	out := new(LastModifiedOut)
	c.call("LastModified", LastModifiedIn{}, out)
	return out.Time
}

func (c *RPCClient) Detach(kill bool) error {
	defer c.client.Close()
	out := new(DetachOut)
	return c.call("Detach", DetachIn{kill}, out)
}

func (c *RPCClient) Restart(rebuild bool) ([]api.DiscardedBreakpoint, error) {
	out := new(RestartOut)
	err := c.call("Restart", RestartIn{rebuild}, out)
	return out.DiscardedBreakpoints, err
}

func (c *RPCClient) GetState() (*api.DebuggerState, error) {
	var out StateOut
	err := c.call("State", StateIn{NonBlocking: false}, &out)
	return out.State, err
}

func (c *RPCClient) GetStateNonBlocking() (*api.DebuggerState, error) {
	var out StateOut
	err := c.call("State", StateIn{NonBlocking: true}, &out)
	return out.State, err
}

func (c *RPCClient) Continue() <-chan *api.DebuggerState {
	return c.continueDir(api.Continue)
}

func (c *RPCClient) DirectionCongruentContinue() <-chan *api.DebuggerState {
	return c.continueDir(api.DirectionCongruentContinue)
}

func (c *RPCClient) continueDir(cmd string) <-chan *api.DebuggerState {
	ch := make(chan *api.DebuggerState)
	go func() {
		for {
			out := new(CommandOut)
			err := c.call("Command", &api.DebuggerCommand{Name: cmd, ReturnInfoLoadConfig: c.retValLoadCfg}, &out)
			state := out.State
			if err != nil {
				state.Err = err
			}
			if state.Exited {
				// Error types apparently cannot be marshalled by Go correctly. Must reset error here.
				//lint:ignore ST1005 backwards compatibility
				state.Err = fmt.Errorf("Process %d has exited with status %d", c.ProcessPid(), state.ExitStatus)
			}
			ch <- &state
			if err != nil || state.Exited {
				close(ch)
				return
			}

			isbreakpoint := false
			istracepoint := true
			for i := range state.Threads {
				if state.Threads[i].Breakpoint != nil {
					isbreakpoint = true
					istracepoint = istracepoint && (state.Threads[i].Breakpoint.Tracepoint || state.Threads[i].Breakpoint.TraceReturn)
				}
			}

			if !isbreakpoint || !istracepoint {
				close(ch)
				return
			}
		}
	}()
	return ch
}

func (c *RPCClient) Next() (*api.DebuggerState, error) {
	var out CommandOut
	err := c.call("Command", api.DebuggerCommand{Name: api.Next, ReturnInfoLoadConfig: c.retValLoadCfg}, &out)
	return &out.State, err
}

func (c *RPCClient) Step() (*api.DebuggerState, error) {
	var out CommandOut
	err := c.call("Command", api.DebuggerCommand{Name: api.Step, ReturnInfoLoadConfig: c.retValLoadCfg}, &out)
	return &out.State, err
}

func (c *RPCClient) StepOut() (*api.DebuggerState, error) {
	var out CommandOut
	err := c.call("Command", api.DebuggerCommand{Name: api.StepOut, ReturnInfoLoadConfig: c.retValLoadCfg}, &out)
	return &out.State, err
}

func (c *RPCClient) Call(goroutineID int, expr string, unsafe bool) (*api.DebuggerState, error) {
	var out CommandOut
	err := c.call("Command", api.DebuggerCommand{Name: api.Call, ReturnInfoLoadConfig: c.retValLoadCfg, Expr: expr, UnsafeCall: unsafe, GoroutineID: goroutineID}, &out)
	return &out.State, err
}

func (c *RPCClient) StepInstruction() (*api.DebuggerState, error) {
	var out CommandOut
	err := c.call("Command", api.DebuggerCommand{Name: api.StepInstruction}, &out)
	return &out.State, err
}

func (c *RPCClient) SwitchThread(threadID int) (*api.DebuggerState, error) {
	var out CommandOut
	cmd := api.DebuggerCommand{
		Name:     api.SwitchThread,
		ThreadID: threadID,
	}
	err := c.call("Command", cmd, &out)
	return &out.State, err
}

func (c *RPCClient) SwitchGoroutine(goroutineID int) (*api.DebuggerState, error) {
	var out CommandOut
	cmd := api.DebuggerCommand{
		Name:        api.SwitchGoroutine,
		GoroutineID: goroutineID,
	}
	err := c.call("Command", cmd, &out)
	return &out.State, err
}

func (c *RPCClient) Halt() (*api.DebuggerState, error) {
	var out CommandOut
	err := c.call("Command", api.DebuggerCommand{Name: api.Halt}, &out)
	return &out.State, err
}

func (c *RPCClient) GetBufferedTracepoints() ([]api.TracepointResult, error) {
	var out GetBufferedTracepointsOut
	err := c.call("GetBufferedTracepoints", GetBufferedTracepointsIn{}, &out)
	return out.TracepointResults, err
}

func (c *RPCClient) GetBreakpoint(id int) (*api.Breakpoint, error) {
	var out GetBreakpointOut
	err := c.call("GetBreakpoint", GetBreakpointIn{id, ""}, &out)
	return &out.Breakpoint, err
}

func (c *RPCClient) GetBreakpointByName(name string) (*api.Breakpoint, error) {
	var out GetBreakpointOut
	err := c.call("GetBreakpoint", GetBreakpointIn{0, name}, &out)
	return &out.Breakpoint, err
}

// CreateBreakpoint will send a request to the RPC server to create a breakpoint.
// Please refer to the documentation for `DebuggerConfig.CreateBreakpoint` for a description of how
// the requested breakpoint parameters are interpreted and used:
// https://pkg.go.dev/github.com/hitzhangjie/dlv/service/debugger#Debugger.CreateBreakpoint
func (c *RPCClient) CreateBreakpoint(breakPoint *api.Breakpoint) (*api.Breakpoint, error) {
	var out CreateBreakpointOut
	err := c.call("CreateBreakpoint", CreateBreakpointIn{*breakPoint}, &out)
	return &out.Breakpoint, err
}

func (c *RPCClient) CreateEBPFTracepoint(fnName string) error {
	var out CreateEBPFTracepointOut
	return c.call("CreateEBPFTracepoint", CreateEBPFTracepointIn{FunctionName: fnName}, &out)
}

func (c *RPCClient) CreateWatchpoint(scope api.EvalScope, expr string, wtype api.WatchType) (*api.Breakpoint, error) {
	var out CreateWatchpointOut
	err := c.call("CreateWatchpoint", CreateWatchpointIn{scope, expr, wtype}, &out)
	return out.Breakpoint, err
}

func (c *RPCClient) ListBreakpoints(all bool) ([]*api.Breakpoint, error) {
	var out ListBreakpointsOut
	err := c.call("ListBreakpoints", ListBreakpointsIn{all}, &out)
	return out.Breakpoints, err
}

func (c *RPCClient) ClearBreakpoint(id int) (*api.Breakpoint, error) {
	var out ClearBreakpointOut
	err := c.call("ClearBreakpoint", ClearBreakpointIn{id, ""}, &out)
	return out.Breakpoint, err
}

func (c *RPCClient) ClearBreakpointByName(name string) (*api.Breakpoint, error) {
	var out ClearBreakpointOut
	err := c.call("ClearBreakpoint", ClearBreakpointIn{0, name}, &out)
	return out.Breakpoint, err
}

func (c *RPCClient) ToggleBreakpoint(id int) (*api.Breakpoint, error) {
	var out ToggleBreakpointOut
	err := c.call("ToggleBreakpoint", ToggleBreakpointIn{id, ""}, &out)
	return out.Breakpoint, err
}

func (c *RPCClient) ToggleBreakpointByName(name string) (*api.Breakpoint, error) {
	var out ToggleBreakpointOut
	err := c.call("ToggleBreakpoint", ToggleBreakpointIn{0, name}, &out)
	return out.Breakpoint, err
}

func (c *RPCClient) AmendBreakpoint(bp *api.Breakpoint) error {
	out := new(AmendBreakpointOut)
	err := c.call("AmendBreakpoint", AmendBreakpointIn{*bp}, out)
	return err
}

func (c *RPCClient) CancelNext() error {
	var out CancelNextOut
	return c.call("CancelNext", CancelNextIn{}, &out)
}

func (c *RPCClient) ListThreads() ([]*api.Thread, error) {
	var out ListThreadsOut
	err := c.call("ListThreads", ListThreadsIn{}, &out)
	return out.Threads, err
}

func (c *RPCClient) GetThread(id int) (*api.Thread, error) {
	var out GetThreadOut
	err := c.call("GetThread", GetThreadIn{id}, &out)
	return out.Thread, err
}

func (c *RPCClient) EvalVariable(scope api.EvalScope, expr string, cfg api.LoadConfig) (*api.Variable, error) {
	var out EvalOut
	err := c.call("Eval", EvalIn{scope, expr, &cfg}, &out)
	return out.Variable, err
}

func (c *RPCClient) SetVariable(scope api.EvalScope, symbol, value string) error {
	out := new(SetOut)
	return c.call("Set", SetIn{scope, symbol, value}, out)
}

func (c *RPCClient) ListSources(filter string) ([]string, error) {
	sources := new(ListSourcesOut)
	err := c.call("ListSources", ListSourcesIn{filter}, sources)
	return sources.Sources, err
}

func (c *RPCClient) ListFunctions(filter string) ([]string, error) {
	funcs := new(ListFunctionsOut)
	err := c.call("ListFunctions", ListFunctionsIn{filter}, funcs)
	return funcs.Funcs, err
}

func (c *RPCClient) ListTypes(filter string) ([]string, error) {
	types := new(ListTypesOut)
	err := c.call("ListTypes", ListTypesIn{filter}, types)
	return types.Types, err
}

func (c *RPCClient) ListPackageVariables(filter string, cfg api.LoadConfig) ([]api.Variable, error) {
	var out ListPackageVarsOut
	err := c.call("ListPackageVars", ListPackageVarsIn{filter, cfg}, &out)
	return out.Variables, err
}

func (c *RPCClient) ListLocalVariables(scope api.EvalScope, cfg api.LoadConfig) ([]api.Variable, error) {
	var out ListLocalVarsOut
	err := c.call("ListLocalVars", ListLocalVarsIn{scope, cfg}, &out)
	return out.Variables, err
}

func (c *RPCClient) ListThreadRegisters(threadID int, includeFp bool) (api.Registers, error) {
	out := new(ListRegistersOut)
	err := c.call("ListRegisters", ListRegistersIn{ThreadID: threadID, IncludeFp: includeFp, Scope: nil}, out)
	return out.Regs, err
}

func (c *RPCClient) ListScopeRegisters(scope api.EvalScope, includeFp bool) (api.Registers, error) {
	out := new(ListRegistersOut)
	err := c.call("ListRegisters", ListRegistersIn{ThreadID: 0, IncludeFp: includeFp, Scope: &scope}, out)
	return out.Regs, err
}

func (c *RPCClient) ListFunctionArgs(scope api.EvalScope, cfg api.LoadConfig) ([]api.Variable, error) {
	var out ListFunctionArgsOut
	err := c.call("ListFunctionArgs", ListFunctionArgsIn{scope, cfg}, &out)
	return out.Args, err
}

func (c *RPCClient) ListGoroutines(start, count int) ([]*api.Goroutine, int, error) {
	var out ListGoroutinesOut
	err := c.call("ListGoroutines", ListGoroutinesIn{start, count, nil, api.GoroutineGroupingOptions{}}, &out)
	return out.Goroutines, out.Nextg, err
}

func (c *RPCClient) ListGoroutinesWithFilter(start, count int, filters []api.ListGoroutinesFilter, group *api.GoroutineGroupingOptions) ([]*api.Goroutine, []api.GoroutineGroup, int, bool, error) {
	if group == nil {
		group = &api.GoroutineGroupingOptions{}
	}
	var out ListGoroutinesOut
	err := c.call("ListGoroutines", ListGoroutinesIn{start, count, filters, *group}, &out)
	return out.Goroutines, out.Groups, out.Nextg, out.TooManyGroups, err
}

func (c *RPCClient) Stacktrace(goroutineId, depth int, opts api.StacktraceOptions, cfg *api.LoadConfig) ([]api.Stackframe, error) {
	var out StacktraceOut
	err := c.call("Stacktrace", StacktraceIn{goroutineId, depth, false, false, opts, cfg}, &out)
	return out.Locations, err
}

func (c *RPCClient) Ancestors(goroutineID int, numAncestors int, depth int) ([]api.Ancestor, error) {
	var out AncestorsOut
	err := c.call("Ancestors", AncestorsIn{goroutineID, numAncestors, depth}, &out)
	return out.Ancestors, err
}

func (c *RPCClient) AttachedToExistingProcess() bool {
	out := new(AttachedToExistingProcessOut)
	c.call("AttachedToExistingProcess", AttachedToExistingProcessIn{}, out)
	return out.Answer
}

func (c *RPCClient) FindLocation(scope api.EvalScope, loc string, findInstructions bool, substitutePathRules [][2]string) ([]api.Location, error) {
	var out FindLocationOut
	err := c.call("FindLocation", FindLocationIn{scope, loc, !findInstructions, substitutePathRules}, &out)
	return out.Locations, err
}

// DisassembleRange disassembles code between startPC and endPC
func (c *RPCClient) DisassembleRange(scope api.EvalScope, startPC, endPC uint64, flavour api.AssemblyFlavour) (api.AsmInstructions, error) {
	var out DisassembleOut
	err := c.call("Disassemble", DisassembleIn{scope, startPC, endPC, flavour}, &out)
	return out.Disassemble, err
}

// DisassemblePC disassembles function containing pc
func (c *RPCClient) DisassemblePC(scope api.EvalScope, pc uint64, flavour api.AssemblyFlavour) (api.AsmInstructions, error) {
	var out DisassembleOut
	err := c.call("Disassemble", DisassembleIn{scope, pc, 0, flavour}, &out)
	return out.Disassemble, err
}

func (c *RPCClient) SetReturnValuesLoadConfig(cfg *api.LoadConfig) {
	c.retValLoadCfg = cfg
}

func (c *RPCClient) FunctionReturnLocations(fnName string) ([]uint64, error) {
	var out FunctionReturnLocationsOut
	err := c.call("FunctionReturnLocations", FunctionReturnLocationsIn{fnName}, &out)
	return out.Addrs, err
}

func (c *RPCClient) IsMulticlient() bool {
	var out IsMulticlientOut
	c.call("IsMulticlient", IsMulticlientIn{}, &out)
	return out.IsMulticlient
}

func (c *RPCClient) Disconnect(cont bool) error {
	if cont {
		out := new(CommandOut)
		c.client.Go("RPCServer.Command", &api.DebuggerCommand{Name: api.Continue, ReturnInfoLoadConfig: c.retValLoadCfg}, &out, nil)
	}
	return c.client.Close()
}

func (c *RPCClient) ListDynamicLibraries() ([]api.Image, error) {
	var out ListDynamicLibrariesOut
	c.call("ListDynamicLibraries", ListDynamicLibrariesIn{}, &out)
	return out.List, nil
}

func (c *RPCClient) ExamineMemory(address uint64, count int) ([]byte, bool, error) {
	out := &ExaminedMemoryOut{}

	err := c.call("ExamineMemory", ExamineMemoryIn{Length: count, Address: address}, out)
	if err != nil {
		return nil, false, err
	}
	return out.Mem, out.IsLittleEndian, nil
}

func (c *RPCClient) CoreDumpStart(dest string) (api.DumpState, error) {
	out := &DumpStartOut{}
	err := c.call("DumpStart", DumpStartIn{Destination: dest}, out)
	return out.State, err
}

func (c *RPCClient) CoreDumpWait(msec int) api.DumpState {
	out := &DumpWaitOut{}
	_ = c.call("DumpWait", DumpWaitIn{Wait: msec}, out)
	return out.State
}

func (c *RPCClient) CoreDumpCancel() error {
	out := &DumpCancelOut{}
	return c.call("DumpCancel", DumpCancelIn{}, out)
}

// call sends any RPC request to debugger service
func (c *RPCClient) call(method string, args, reply interface{}) error {
	return c.client.Call("RPCServer."+method, args, reply)
}

func (c *RPCClient) CallAPI(method string, args, reply interface{}) error {
	return c.call(method, args, reply)
}
