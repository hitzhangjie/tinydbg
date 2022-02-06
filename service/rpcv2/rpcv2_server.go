package rpcv2

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/hitzhangjie/dlv/pkg/dwarf/op"
	"github.com/hitzhangjie/dlv/pkg/proc"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// RPCServer RPC server serves the RPC reqeust from RPC client.
//
// RPCService will interact with the inner debugger service to control and query
// the states of the tracee, then responds the RPC client.
type RPCServer struct {
	config   *service.Config    // all the information necessary to start the debugger and server.
	debugger *debugger.Debugger // the debugger service
}

// NewServer returns a new RPC server.
func NewServer(config *service.Config, debugger *debugger.Debugger) *RPCServer {
	return &RPCServer{config, debugger}
}

// ProcessPid returns the pid of the process we are debugging.
func (s *RPCServer) ProcessPid(arg ProcessPidIn, out *ProcessPidOut) error {
	out.Pid = s.debugger.ProcessPid()
	return nil
}

// LastModified returns the last modified time of the debugged program.
func (s *RPCServer) LastModified(arg LastModifiedIn, out *LastModifiedOut) error {
	out.Time = s.debugger.LastModified()
	return nil
}

// Detach detaches the debugger, optionally killing the process.
func (s *RPCServer) Detach(arg DetachIn, out *DetachOut) error {
	return s.debugger.Detach(arg.Kill)
}

// Restart restarts program.
func (s *RPCServer) Restart(arg RestartIn, cb service.RPCCallback) {
	close(cb.SetupDoneChan())
	if s.config.DebuggerConfig.AttachPid != 0 {
		cb.Return(nil, errors.New("cannot restart process Delve did not create"))
		return
	}
	var out RestartOut
	var err error
	out.DiscardedBreakpoints, err = s.debugger.Restart(arg.Rerecord, arg.Position, arg.ResetArgs, arg.NewArgs, arg.Rebuild)
	cb.Return(out, err)
}

// State returns the current debugger's state.
func (s *RPCServer) State(arg StateIn, cb service.RPCCallback) {
	close(cb.SetupDoneChan())
	var out StateOut
	st, err := s.debugger.State(arg.NonBlocking)
	if err != nil {
		cb.Return(nil, err)
		return
	}
	out.State = st
	cb.Return(out, nil)
}

// Command interrupts, continues and steps through the program.
func (s *RPCServer) Command(command api.DebuggerCommand, cb service.RPCCallback) {
	st, err := s.debugger.Command(&command, cb.SetupDoneChan())
	if err != nil {
		cb.Return(nil, err)
		return
	}
	var out CommandOut
	out.State = *st
	cb.Return(out, nil)
}

// GetBufferedTracepoints returns the buffered tracepoints? todo so what does it actually do?
func (s *RPCServer) GetBufferedTracepoints(arg GetBufferedTracepointsIn, out *GetBufferedTracepointsOut) error {
	out.TracepointResults = s.debugger.GetBufferedTracepoints()
	return nil
}

// GetBreakpoint gets a breakpoint by Name (if Name is not an empty string) or by ID.
func (s *RPCServer) GetBreakpoint(arg GetBreakpointIn, out *GetBreakpointOut) error {
	var bp *api.Breakpoint
	if arg.Name != "" {
		bp = s.debugger.FindBreakpointByName(arg.Name)
		if bp == nil {
			return fmt.Errorf("no breakpoint with name %s", arg.Name)
		}
	} else {
		bp = s.debugger.FindBreakpoint(arg.Id)
		if bp == nil {
			return fmt.Errorf("no breakpoint with id %d", arg.Id)
		}
	}
	out.Breakpoint = *bp
	return nil
}

// Stacktrace returns stacktrace of goroutine Id up to the specified Depth.
//
// If Full is set it will also the variable of all local variables
// and function arguments of all stack frames.
func (s *RPCServer) Stacktrace(arg StacktraceIn, out *StacktraceOut) error {
	cfg := arg.Cfg
	if cfg == nil && arg.Full {
		cfg = &api.LoadConfig{FollowPointers: true, MaxVariableRecurse: 1, MaxStringLen: 64, MaxArrayValues: 64, MaxStructFields: -1}
	}
	if arg.Defers {
		arg.Opts |= api.StacktraceReadDefers
	}
	var err error
	rawlocs, err := s.debugger.Stacktrace(arg.Id, arg.Depth, arg.Opts)
	if err != nil {
		return err
	}
	out.Locations, err = s.debugger.ConvertStacktrace(rawlocs, api.LoadConfigToProc(cfg))
	return err
}

// Ancestors returns the stacktraces for the ancestors of a goroutine.
func (s *RPCServer) Ancestors(arg AncestorsIn, out *AncestorsOut) error {
	var err error
	out.Ancestors, err = s.debugger.Ancestors(arg.GoroutineID, arg.NumAncestors, arg.Depth)
	return err
}

// ListBreakpoints gets all breakpoints.
func (s *RPCServer) ListBreakpoints(arg ListBreakpointsIn, out *ListBreakpointsOut) error {
	out.Breakpoints = s.debugger.Breakpoints(arg.All)
	return nil
}

// CreateBreakpoint creates a new breakpoint. The client is expected to populate `CreateBreakpointIn`
// with an `api.Breakpoint` struct describing where to set the breakpoing. For more information on
// how to properly request a breakpoint via the `api.Breakpoint` struct see the documentation for
// `debugger.CreateBreakpoint` here: https://pkg.go.dev/github.com/hitzhangjie/dlv/service/debugger#Debugger.CreateBreakpoint.
func (s *RPCServer) CreateBreakpoint(arg CreateBreakpointIn, out *CreateBreakpointOut) error {
	if err := api.ValidBreakpointName(arg.Breakpoint.Name); err != nil {
		return err
	}
	createdbp, err := s.debugger.CreateBreakpoint(&arg.Breakpoint)
	if err != nil {
		return err
	}
	out.Breakpoint = *createdbp
	return nil
}

// CreateEBPFTracepoint create ebpf tracepoint
func (s *RPCServer) CreateEBPFTracepoint(arg CreateEBPFTracepointIn, out *CreateEBPFTracepointOut) error {
	return s.debugger.CreateEBPFTracepoint(arg.FunctionName)
}

// ClearBreakpoint deletes a breakpoint by Name (if Name is not an
// empty string) or by ID.
func (s *RPCServer) ClearBreakpoint(arg ClearBreakpointIn, out *ClearBreakpointOut) error {
	var bp *api.Breakpoint
	if arg.Name != "" {
		bp = s.debugger.FindBreakpointByName(arg.Name)
		if bp == nil {
			return fmt.Errorf("no breakpoint with name %s", arg.Name)
		}
	} else {
		bp = s.debugger.FindBreakpoint(arg.Id)
		if bp == nil {
			return fmt.Errorf("no breakpoint with id %d", arg.Id)
		}
	}
	deleted, err := s.debugger.ClearBreakpoint(bp)
	if err != nil {
		return err
	}
	out.Breakpoint = deleted
	return nil
}

// ToggleBreakpoint toggles on or off a breakpoint by Name (if Name is not an
// empty string) or by ID.
func (s *RPCServer) ToggleBreakpoint(arg ToggleBreakpointIn, out *ToggleBreakpointOut) error {
	var bp *api.Breakpoint
	if arg.Name != "" {
		bp = s.debugger.FindBreakpointByName(arg.Name)
		if bp == nil {
			return fmt.Errorf("no breakpoint with name %s", arg.Name)
		}
	} else {
		bp = s.debugger.FindBreakpoint(arg.Id)
		if bp == nil {
			return fmt.Errorf("no breakpoint with id %d", arg.Id)
		}
	}
	bp.Disabled = !bp.Disabled
	if err := api.ValidBreakpointName(bp.Name); err != nil {
		return err
	}
	if err := s.debugger.AmendBreakpoint(bp); err != nil {
		return err
	}
	out.Breakpoint = bp
	return nil
}

// AmendBreakpoint allows user to update an existing breakpoint
// for example to change the information retrieved when the
// breakpoint is hit or to change, add or remove the break condition.
//
// arg.Breakpoint.ID must be a valid breakpoint ID
func (s *RPCServer) AmendBreakpoint(arg AmendBreakpointIn, out *AmendBreakpointOut) error {
	if err := api.ValidBreakpointName(arg.Breakpoint.Name); err != nil {
		return err
	}
	return s.debugger.AmendBreakpoint(&arg.Breakpoint)
}

func (s *RPCServer) CancelNext(arg CancelNextIn, out *CancelNextOut) error {
	return s.debugger.CancelNext()
}

// ListThreads lists all threads.
func (s *RPCServer) ListThreads(arg ListThreadsIn, out *ListThreadsOut) (err error) {
	threads, err := s.debugger.Threads()
	if err != nil {
		return err
	}
	s.debugger.LockTarget()
	defer s.debugger.UnlockTarget()
	out.Threads = api.ConvertThreads(threads)
	return nil
}

// GetThread gets a thread by its ID.
func (s *RPCServer) GetThread(arg GetThreadIn, out *GetThreadOut) error {
	t, err := s.debugger.FindThread(arg.Id)
	if err != nil {
		return err
	}
	if t == nil {
		return fmt.Errorf("no thread with id %d", arg.Id)
	}
	s.debugger.LockTarget()
	defer s.debugger.UnlockTarget()
	out.Thread = api.ConvertThread(t)
	return nil
}

// ListPackageVars lists all package variables in the context of the current thread.
func (s *RPCServer) ListPackageVars(arg ListPackageVarsIn, out *ListPackageVarsOut) error {
	vars, err := s.debugger.PackageVariables(arg.Filter, *api.LoadConfigToProc(&arg.Cfg))
	if err != nil {
		return err
	}
	out.Variables = api.ConvertVars(vars)
	return nil
}

// ListRegisters lists registers and their values.
// If ListRegistersIn.Scope is not nil the registers of that eval scope will
// be returned, otherwise ListRegistersIn.ThreadID will be used.
func (s *RPCServer) ListRegisters(arg ListRegistersIn, out *ListRegistersOut) error {
	if arg.ThreadID == 0 && arg.Scope == nil {
		state, err := s.debugger.State(false)
		if err != nil {
			return err
		}
		arg.ThreadID = state.CurrentThread.ID
	}

	var regs *op.DwarfRegisters
	var err error

	if arg.Scope != nil {
		regs, err = s.debugger.ScopeRegisters(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, arg.IncludeFp)
	} else {
		regs, err = s.debugger.ThreadRegisters(arg.ThreadID, arg.IncludeFp)
	}
	if err != nil {
		return err
	}
	out.Regs = api.ConvertRegisters(regs, s.debugger.DwarfRegisterToString, arg.IncludeFp)
	out.Registers = out.Regs.String()

	return nil
}

// ListLocalVars lists all local variables in scope.
func (s *RPCServer) ListLocalVars(arg ListLocalVarsIn, out *ListLocalVarsOut) error {
	vars, err := s.debugger.LocalVariables(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, *api.LoadConfigToProc(&arg.Cfg))
	if err != nil {
		return err
	}
	out.Variables = api.ConvertVars(vars)
	return nil
}

// ListFunctionArgs lists all arguments to the current function
func (s *RPCServer) ListFunctionArgs(arg ListFunctionArgsIn, out *ListFunctionArgsOut) error {
	vars, err := s.debugger.FunctionArguments(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, *api.LoadConfigToProc(&arg.Cfg))
	if err != nil {
		return err
	}
	out.Args = api.ConvertVars(vars)
	return nil
}

// Eval returns a variable in the specified context.
//
// See https://github.com/hitzhangjie/dlv/blob/master/Documentation/cli/expr.md
// for a description of acceptable values of arg.Expr.
func (s *RPCServer) Eval(arg EvalIn, out *EvalOut) error {
	cfg := arg.Cfg
	if cfg == nil {
		cfg = &api.LoadConfig{FollowPointers: true, MaxVariableRecurse: 1, MaxStringLen: 64, MaxArrayValues: 64, MaxStructFields: -1}
	}
	v, err := s.debugger.EvalVariableInScope(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, arg.Expr, *api.LoadConfigToProc(cfg))
	if err != nil {
		return err
	}
	out.Variable = api.ConvertVar(v)
	return nil
}

// Set sets the value of a variable. Only numerical types and
// pointers are currently supported.
func (s *RPCServer) Set(arg SetIn, out *SetOut) error {
	return s.debugger.SetVariableInScope(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, arg.Symbol, arg.Value)
}

// ListSources lists all source files in the process matching filter.
func (s *RPCServer) ListSources(arg ListSourcesIn, out *ListSourcesOut) error {
	ss, err := s.debugger.Sources(arg.Filter)
	if err != nil {
		return err
	}
	out.Sources = ss
	return nil
}

// ListFunctions lists all functions in the process matching filter.
func (s *RPCServer) ListFunctions(arg ListFunctionsIn, out *ListFunctionsOut) error {
	fns, err := s.debugger.Functions(arg.Filter)
	if err != nil {
		return err
	}
	out.Funcs = fns
	return nil
}

// ListTypes lists all types in the process matching filter.
func (s *RPCServer) ListTypes(arg ListTypesIn, out *ListTypesOut) error {
	tps, err := s.debugger.Types(arg.Filter)
	if err != nil {
		return err
	}
	out.Types = tps
	return nil
}

// ListGoroutines lists all goroutines.
// If Count is specified ListGoroutines will return at the first Count
// goroutines and an index in Nextg, that can be passed as the Start
// parameter, to get more goroutines from ListGoroutines.
// Passing a value of Start that wasn't returned by ListGoroutines will skip
// an undefined number of goroutines.
//
// If arg.Filters are specified the list of returned goroutines is filtered
// applying the specified filters.
// For example:
//    ListGoroutinesFilter{ Kind: ListGoroutinesFilterUserLoc, Negated: false, Arg: "afile.go" }
// will only return goroutines whose UserLoc contains "afile.go" as a substring.
// More specifically a goroutine matches a location filter if the specified
// location, formatted like this:
//    filename:lineno in function
// contains Arg[0] as a substring.
//
// Filters can also be applied to goroutine labels:
//    ListGoroutineFilter{ Kind: ListGoroutinesFilterLabel, Negated: false, Arg: "key=value" }
// this filter will only return goroutines that have a key=value label.
//
// If arg.GroupBy is not GoroutineFieldNone then the goroutines will
// be grouped with the specified criterion.
// If the value of arg.GroupBy is GoroutineLabel goroutines will
// be grouped by the value of the label with key GroupByKey.
// For each group a maximum of MaxExamples example goroutines are
// returned, as well as the total number of goroutines in the group.
func (s *RPCServer) ListGoroutines(arg ListGoroutinesIn, out *ListGoroutinesOut) error {
	// TODO(aarzilli): if arg contains a running goroutines filter (not negated)
	// and start == 0 and count == 0 then we can optimize this by just looking
	// at threads directly.
	gs, nextg, err := s.debugger.Goroutines(arg.Start, arg.Count)
	if err != nil {
		return err
	}
	gs = s.debugger.FilterGoroutines(gs, arg.Filters)
	gs, out.Groups, out.TooManyGroups = s.debugger.GroupGoroutines(gs, &arg.GoroutineGroupingOptions)
	s.debugger.LockTarget()
	defer s.debugger.UnlockTarget()
	out.Goroutines = api.ConvertGoroutines(s.debugger.Target(), gs)
	out.Nextg = nextg
	return nil
}

// AttachedToExistingProcess returns whether we attached to a running process or not
func (c *RPCServer) AttachedToExistingProcess(arg AttachedToExistingProcessIn, out *AttachedToExistingProcessOut) error {
	if c.config.DebuggerConfig.AttachPid != 0 {
		out.Answer = true
	}
	return nil
}

// FindLocation returns concrete location information described by a location expression.
//
//  loc ::= <filename>:<line> | <function>[:<line>] | /<regex>/ | (+|-)<offset> | <line> | *<address>
//  * <filename> can be the full path of a file or just a suffix
//  * <function> ::= <package>.<receiver type>.<name> | <package>.(*<receiver type>).<name> | <receiver type>.<name> | <package>.<name> | (*<receiver type>).<name> | <name>
//  * <function> must be unambiguous
//  * /<regex>/ will return a location for each function matched by regex
//  * +<offset> returns a location for the line that is <offset> lines after the current line
//  * -<offset> returns a location for the line that is <offset> lines before the current line
//  * <line> returns a location for a line in the current file
//  * *<address> returns the location corresponding to the specified address
//
// NOTE: this function does not actually set breakpoints.
func (c *RPCServer) FindLocation(arg FindLocationIn, out *FindLocationOut) (err error) {
	goid := arg.Scope.GoroutineID
	frame := arg.Scope.Frame
	deferred := arg.Scope.DeferredCall
	out.Locations, err = c.debugger.FindLocation(goid, frame, deferred, arg.Loc, arg.IncludeNonExecutableLines, arg.SubstitutePathRules)
	return err
}

// Disassemble code.
//
// If both StartPC and EndPC are non-zero the specified range will be disassembled, otherwise the function containing StartPC will be disassembled.
//
// Scope is used to mark the instruction the specified goroutine is stopped at.
//
// Disassemble will also try to calculate the destination address of an absolute indirect CALL if it happens to be the instruction the selected goroutine is stopped at.
func (c *RPCServer) Disassemble(arg DisassembleIn, out *DisassembleOut) error {
	var err error
	insts, err := c.debugger.Disassemble(arg.Scope.GoroutineID, arg.StartPC, arg.EndPC)
	if err != nil {
		return err
	}
	out.Disassemble = make(api.AsmInstructions, len(insts))
	for i := range insts {
		out.Disassemble[i] = api.ConvertAsmInstruction(insts[i], c.debugger.AsmInstructionText(&insts[i], proc.AssemblyFlavour(arg.Flavour)))
	}
	return nil
}

func (s *RPCServer) Recorded(arg RecordedIn, out *RecordedOut) error {
	out.Recorded, out.TraceDirectory = s.debugger.Recorded()
	return nil
}

func (s *RPCServer) Checkpoint(arg CheckpointIn, out *CheckpointOut) error {
	var err error
	out.ID, err = s.debugger.Checkpoint(arg.Where)
	return err
}

func (s *RPCServer) ListCheckpoints(arg ListCheckpointsIn, out *ListCheckpointsOut) error {
	var err error
	cps, err := s.debugger.Checkpoints()
	if err != nil {
		return err
	}
	out.Checkpoints = make([]api.Checkpoint, len(cps))
	for i := range cps {
		out.Checkpoints[i] = api.Checkpoint(cps[i])
	}
	return nil
}

func (s *RPCServer) ClearCheckpoint(arg ClearCheckpointIn, out *ClearCheckpointOut) error {
	return s.debugger.ClearCheckpoint(arg.ID)
}

func (s *RPCServer) IsMulticlient(arg IsMulticlientIn, out *IsMulticlientOut) error {
	*out = IsMulticlientOut{
		IsMulticlient: s.config.AcceptMulti,
	}
	return nil
}

// FunctionReturnLocations is the implements the client call of the same name. Look at client documentation for more information.
func (s *RPCServer) FunctionReturnLocations(in FunctionReturnLocationsIn, out *FunctionReturnLocationsOut) error {
	addrs, err := s.debugger.FunctionReturnLocations(in.FnName)
	if err != nil {
		return err
	}
	*out = FunctionReturnLocationsOut{
		Addrs: addrs,
	}
	return nil
}

func (s *RPCServer) ListDynamicLibraries(in ListDynamicLibrariesIn, out *ListDynamicLibrariesOut) error {
	imgs := s.debugger.ListDynamicLibraries()
	out.List = make([]api.Image, 0, len(imgs))
	for i := range imgs {
		out.List = append(out.List, api.ConvertImage(imgs[i]))
	}
	return nil
}

// ListPackagesBuildInfo returns the list of packages used by the program along with
// the directory where each package was compiled and optionally the list of
// files constituting the package.
// Note that the directory path is a best guess and may be wrong is a tool
// other than cmd/go is used to perform the build.
func (s *RPCServer) ListPackagesBuildInfo(in ListPackagesBuildInfoIn, out *ListPackagesBuildInfoOut) error {
	pkgs := s.debugger.ListPackagesBuildInfo(in.IncludeFiles)
	out.List = make([]api.PackageBuildInfo, 0, len(pkgs))
	for _, pkg := range pkgs {
		var files []string

		if len(pkg.Files) > 0 {
			files = make([]string, 0, len(pkg.Files))
			for file := range pkg.Files {
				files = append(files, file)
			}
		}

		sort.Strings(files)

		out.List = append(out.List, api.PackageBuildInfo{
			ImportPath:    pkg.ImportPath,
			DirectoryPath: pkg.DirectoryPath,
			Files:         files,
		})
	}
	return nil
}

func (s *RPCServer) ExamineMemory(arg ExamineMemoryIn, out *ExaminedMemoryOut) error {
	if arg.Length > 1000 {
		return fmt.Errorf("len must be less than or equal to 1000")
	}
	Mem, err := s.debugger.ExamineMemory(arg.Address, arg.Length)
	if err != nil {
		return err
	}

	out.Mem = Mem
	out.IsLittleEndian = true // TODO: get byte order from debugger.target.BinInfo().Arch

	return nil
}

func (s *RPCServer) StopRecording(arg StopRecordingIn, cb service.RPCCallback) {
	close(cb.SetupDoneChan())
	var out StopRecordingOut
	err := s.debugger.StopRecording()
	if err != nil {
		cb.Return(nil, err)
		return
	}
	cb.Return(out, nil)
}

// DumpStart starts a core dump to arg.Destination.
func (s *RPCServer) DumpStart(arg DumpStartIn, out *DumpStartOut) error {
	err := s.debugger.DumpStart(arg.Destination)
	if err != nil {
		return err
	}
	out.State = *api.ConvertDumpState(s.debugger.DumpWait(0))
	return nil
}

// DumpWait waits for the core dump to finish or for arg.Wait milliseconds.
// Wait == 0 means return immediately.
// Returns the core dump status
func (s *RPCServer) DumpWait(arg DumpWaitIn, out *DumpWaitOut) error {
	out.State = *api.ConvertDumpState(s.debugger.DumpWait(time.Duration(arg.Wait) * time.Millisecond))
	return nil
}

// DumpCancel cancels the core dump.
func (s *RPCServer) DumpCancel(arg DumpCancelIn, out *DumpCancelOut) error {
	return s.debugger.DumpCancel()
}

func (s *RPCServer) CreateWatchpoint(arg CreateWatchpointIn, out *CreateWatchpointOut) error {
	var err error
	out.Breakpoint, err = s.debugger.CreateWatchpoint(arg.Scope.GoroutineID, arg.Scope.Frame, arg.Scope.DeferredCall, arg.Expr, arg.Type)
	return err
}
