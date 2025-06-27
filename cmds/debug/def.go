// Package terminal implements functions for responding to user
// input and dispatching to appropriate backend commands.
package debug

//lint:file-ignore ST1005 errors here can be capitalized

import (
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cosiner/argv"
	"github.com/hitzhangjie/tinydbg/pkg/config"
	"github.com/hitzhangjie/tinydbg/pkg/locspec"
	"github.com/hitzhangjie/tinydbg/pkg/proc/debuginfod"
	"github.com/hitzhangjie/tinydbg/service"
	"github.com/hitzhangjie/tinydbg/service/api"
	"github.com/hitzhangjie/tinydbg/service/rpc2"
	"github.com/spf13/pflag"
)

// cmdPrefix represents the prefix of a command.
type cmdPrefix int

const (
	noPrefix = cmdPrefix(0)
	onPrefix = cmdPrefix(1 << iota)
	deferredPrefix
)

// callContext represents the context of a command.
type callContext struct {
	Prefix     cmdPrefix
	Scope      api.EvalScope
	Breakpoint *api.Breakpoint
}

func (ctx *callContext) scoped() bool {
	return ctx.Scope.GoroutineID >= 0 || ctx.Scope.Frame > 0
}

// frameDirection represents the direction of the frame.
type frameDirection int

const (
	frameSet frameDirection = iota
	frameUp
	frameDown
)

type cmdfunc func(t *Session, ctx callContext, args string) error

type command struct {
	aliases         []string
	builtinAliases  []string
	group           commandGroup
	allowedPrefixes cmdPrefix
	helpMsg         string
	cmdFn           cmdfunc
}

// Returns true if the command string matches one of the aliases for this command
func (c command) match(cmdstr string) bool {
	for _, v := range c.aliases {
		if v == cmdstr {
			return true
		}
	}
	return false
}

// reprensents the group of commands
type commandGroup uint8

const (
	otherCmds commandGroup = iota
	breakCmds
	runCmds
	dataCmds
	goroutineCmds
	stackCmds
	sourceCmds
)

type commandGroupDescription struct {
	description string
	group       commandGroup
}

var commandGroupDescriptions = []commandGroupDescription{
	{"Running the program", runCmds},
	{"Manipulating breakpoints", breakCmds},
	{"Inspect program variables and memory", dataCmds},
	{"Inspect the call stack and selecting frames", stackCmds},
	{"Viewing source and disassembly, Listing pkgs, funcs, types", sourceCmds},
	{"Listing and switching between threads and goroutines", goroutineCmds},
	{"Other commands", otherCmds},
}

// DebugCommands represents the commands for Delve terminal process.
type DebugCommands struct {
	cmds   []*command
	client service.Client
	frame  int // Current frame as set by frame/up/down commands.
}

var (
	// longLoadConfig loads more information:
	// * Follows pointers
	// * Loads more array values
	// * Does not limit struct fields
	longLoadConfig = api.LoadConfig{FollowPointers: true, MaxVariableRecurse: 1, MaxStringLen: 64, MaxArrayValues: 64, MaxStructFields: -1}
	// ShortLoadConfig loads less information, not following pointers
	// and limiting struct fields loaded to 3.
	ShortLoadConfig = api.LoadConfig{MaxStringLen: 64, MaxStructFields: 3}
)

type newDebugCmdFunc func(*DebugCommands) *command

var supportedDebugCmds []newDebugCmdFunc

func registerDebugCmd(c newDebugCmdFunc) {
	supportedDebugCmds = append(supportedDebugCmds, c)
}

// NewDebugCommands returns a Commands struct with default commands defined.
func NewDebugCommands(client service.Client) *DebugCommands {
	c := &DebugCommands{client: client}

	// Gather all commands from different groups
	for _, f := range supportedDebugCmds {
		c.cmds = append(c.cmds, f(c))
	}

	// Sort commands by first alias
	slices.SortFunc(c.cmds, func(a, b *command) int {
		return strings.Compare(a.aliases[0], b.aliases[0])
	})

	return c
}

// Register custom commands. Expects cf to be a func of type cmdfunc,
// returning only an error.
func (s *DebugCommands) Register(cmdstr string, cf cmdfunc, helpMsg string) {
	for _, v := range s.cmds {
		if v.match(cmdstr) {
			v.cmdFn = cf
			return
		}
	}

	s.cmds = append(s.cmds, &command{aliases: []string{cmdstr}, cmdFn: cf, helpMsg: helpMsg})
}

// Find will look up the command function for the given command input.
// If it cannot find the command it will default to noCmdAvailable().
// If the command is an empty string it will replay the last command.
func (s *DebugCommands) Find(cmdstr string, prefix cmdPrefix) *command {
	// If <enter> use last command, if there was one.
	if cmdstr == "" {
		return &command{aliases: []string{"nullcmd"}, cmdFn: nullCommand}
	}

	for _, v := range s.cmds {
		if v.match(cmdstr) {
			if prefix != noPrefix && v.allowedPrefixes&prefix == 0 {
				continue
			}
			return v
		}
	}

	return &command{aliases: []string{"nocmd"}, cmdFn: noCmdAvailable}
}

// CallWithContext takes a command and a context that command should be executed in.
func (s *DebugCommands) CallWithContext(cmdstr string, t *Session, ctx callContext) error {
	vals := strings.SplitN(strings.TrimSpace(cmdstr), " ", 2)
	cmdname := vals[0]
	var args string
	if len(vals) > 1 {
		args = strings.TrimSpace(vals[1])
	}
	return s.Find(cmdname, ctx.Prefix).cmdFn(t, ctx, args)
}

// Call takes a command to execute.
func (s *DebugCommands) Call(cmdstr string, t *Session) error {
	ctx := callContext{Prefix: noPrefix, Scope: api.EvalScope{GoroutineID: -1, Frame: s.frame, DeferredCall: 0}}
	return s.CallWithContext(cmdstr, t, ctx)
}

// Merge takes aliases defined in the config struct and merges them with the default aliases.
func (s *DebugCommands) Merge(allAliases map[string][]string) {
	for i := range s.cmds {
		if s.cmds[i].builtinAliases != nil {
			s.cmds[i].aliases = append(s.cmds[i].aliases[:0], s.cmds[i].builtinAliases...)
		}
	}
	for i := range s.cmds {
		if aliases, ok := allAliases[s.cmds[i].aliases[0]]; ok {
			if s.cmds[i].builtinAliases == nil {
				s.cmds[i].builtinAliases = make([]string, len(s.cmds[i].aliases))
				copy(s.cmds[i].builtinAliases, s.cmds[i].aliases)
			}
			s.cmds[i].aliases = append(s.cmds[i].aliases, aliases...)
		}
	}
}

var errNoCmd = errors.New("command not available")

func noCmdAvailable(t *Session, ctx callContext, args string) error {
	return errNoCmd
}

func nullCommand(t *Session, ctx callContext, args string) error {
	return nil
}

func (s *DebugCommands) help(t *Session, ctx callContext, args string) error {
	if args != "" {
		for _, cmd := range s.cmds {
			for _, alias := range cmd.aliases {
				if alias == args {
					fmt.Fprintln(t.stdout, cmd.helpMsg)
					return nil
				}
			}
		}
		return errNoCmd
	}

	fmt.Fprintln(t.stdout, "The following commands are available:")

	for _, cgd := range commandGroupDescriptions {
		fmt.Fprintf(t.stdout, "\n%s:\n", cgd.description)
		w := new(tabwriter.Writer)
		w.Init(t.stdout, 0, 8, 0, '-', 0)
		for _, cmd := range s.cmds {
			if cmd.group != cgd.group {
				continue
			}
			h := cmd.helpMsg
			if idx := strings.Index(h, "\n"); idx >= 0 {
				h = h[:idx]
			}
			if len(cmd.aliases) > 1 {
				fmt.Fprintf(w, "    %s (alias: %s) \t %s\n", cmd.aliases[0], strings.Join(cmd.aliases[1:], " | "), h)
			} else {
				fmt.Fprintf(w, "    %s \t %s\n", cmd.aliases[0], h)
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	fmt.Fprintln(t.stdout)
	fmt.Fprintln(t.stdout, "Type help followed by a command for full documentation.")
	return nil
}

func threads(t *Session, ctx callContext, args string) error {
	threads, err := t.client.ListThreads()
	if err != nil {
		return err
	}
	state, err := t.client.GetState()
	if err != nil {
		return err
	}
	slices.SortFunc(threads, func(a, b *api.Thread) int { return cmp.Compare(a.ID, b.ID) })
	done := false
	for _, th := range threads {
		if done {
			break
		}
		prefix := "  "
		if state.CurrentThread != nil && state.CurrentThread.ID == th.ID {
			prefix = "* "
		}
		if th.Function != nil {
			fmt.Fprintf(t.stdout, "%sThread %d at %#v %s:%d %s\n",
				prefix, th.ID, th.PC, t.formatPath(th.File),
				th.Line, th.Function.Name())
		} else {
			fmt.Fprintf(t.stdout, "%sThread %s\n", prefix, t.formatThread(th))
		}
	}
	return nil
}

func thread(t *Session, ctx callContext, args string) error {
	if len(args) == 0 {
		return errors.New("you must specify a thread")
	}
	tid, err := strconv.Atoi(args)
	if err != nil {
		return err
	}
	oldState, err := t.client.GetState()
	if err != nil {
		return err
	}
	newState, err := t.client.SwitchThread(tid)
	if err != nil {
		return err
	}

	oldThread := "<none>"
	newThread := "<none>"
	if oldState.CurrentThread != nil {
		oldThread = strconv.Itoa(oldState.CurrentThread.ID)
	}
	if newState.CurrentThread != nil {
		newThread = strconv.Itoa(newState.CurrentThread.ID)
	}
	fmt.Fprintf(t.stdout, "Switched from %s to %s\n", oldThread, newThread)
	return nil
}

func (s *DebugCommands) printGoroutines(t *Session, ctx callContext, indent string, gs []*api.Goroutine, fgl api.FormatGoroutineLoc, flags api.PrintGoroutinesFlags, depth int, cmd string, pdone *bool, state *api.DebuggerState) error {
	for _, g := range gs {
		if t.longCommandCanceled() || (pdone != nil && *pdone) {
			break
		}
		prefix := indent + "  "
		if state.SelectedGoroutine != nil && g.ID == state.SelectedGoroutine.ID {
			prefix = indent + "* "
		}
		fmt.Fprintf(t.stdout, "%sGoroutine %s\n", prefix, t.formatGoroutine(g, fgl))
		if flags&api.PrintGoroutinesLabels != 0 {
			writeGoroutineLabels(t.stdout, g, indent+"\t")
		}
		if flags&api.PrintGoroutinesStack != 0 {
			stack, err := t.client.Stacktrace(g.ID, depth, 0, nil)
			if err != nil {
				return err
			}
			printStack(t, t.stdout, stack, indent+"\t", false)
		}
		if cmd != "" {
			ctx.Scope.GoroutineID = g.ID
			if err := s.CallWithContext(cmd, t, ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *DebugCommands) goroutines(t *Session, ctx callContext, argstr string) error {
	filters, group, fgl, flags, depth, batchSize, cmd, err := api.ParseGoroutineArgs(argstr)
	if err != nil {
		return err
	}

	state, err := t.client.GetState()
	if err != nil {
		return err
	}
	var (
		start         = 0
		gslen         = 0
		gs            []*api.Goroutine
		groups        []api.GoroutineGroup
		tooManyGroups bool
	)
	done := false
	t.longCommandStart()
	for start >= 0 {
		if t.longCommandCanceled() || done {
			fmt.Fprintf(t.stdout, "interrupted\n")
			return nil
		}
		gs, groups, start, tooManyGroups, err = t.client.ListGoroutinesWithFilter(start, batchSize, filters, &group, &api.EvalScope{GoroutineID: -1, Frame: s.frame})
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			for i := range groups {
				fmt.Fprintf(t.stdout, "%s\n", groups[i].Name)
				err = s.printGoroutines(t, ctx, "\t", gs[groups[i].Offset:][:groups[i].Count], fgl, flags, depth, cmd, &done, state)
				if err != nil {
					return err
				}
				fmt.Fprintf(t.stdout, "\tTotal: %d\n", groups[i].Total)
				if i != len(groups)-1 {
					fmt.Fprintf(t.stdout, "\n")
				}
			}
			if tooManyGroups {
				fmt.Fprintf(t.stdout, "Too many groups\n")
			}
		} else {
			slices.SortFunc(gs, func(a, b *api.Goroutine) int { return cmp.Compare(a.ID, b.ID) })
			err = s.printGoroutines(t, ctx, "", gs, fgl, flags, depth, cmd, &done, state)
			if err != nil {
				return err
			}
			gslen += len(gs)
		}
	}
	if gslen > 0 {
		fmt.Fprintf(t.stdout, "[%d goroutines]\n", gslen)
	}
	return nil
}

func selectedGID(state *api.DebuggerState) int64 {
	if state.SelectedGoroutine == nil {
		return 0
	}
	return state.SelectedGoroutine.ID
}

func (s *DebugCommands) goroutine(t *Session, ctx callContext, argstr string) error {
	args := config.Split2PartsBySpace(argstr)

	if ctx.Prefix == onPrefix {
		if len(args) != 1 || args[0] != "" {
			return errors.New("too many arguments to goroutine")
		}
		ctx.Breakpoint.Goroutine = true
		return nil
	}

	if len(args) == 1 {
		if args[0] == "" {
			return printscope(t)
		}
		gid, err := strconv.ParseInt(argstr, 10, 64)
		if err != nil {
			return err
		}

		oldState, err := t.client.GetState()
		if err != nil {
			return err
		}
		newState, err := t.client.SwitchGoroutine(gid)
		if err != nil {
			return err
		}
		s.frame = 0
		fmt.Fprintf(t.stdout, "Switched from %d to %d (thread %d)\n", selectedGID(oldState), gid, newState.CurrentThread.ID)
		return nil
	}

	var err error
	ctx.Scope.GoroutineID, err = strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}
	return s.CallWithContext(args[1], t, ctx)
}

// Handle "frame", "up", "down" commands.
func (s *DebugCommands) frameCommand(t *Session, ctx callContext, argstr string, direction frameDirection) error {
	frame := 1
	arg := ""
	if len(argstr) == 0 {
		if direction == frameSet {
			return errors.New("not enough arguments")
		}
	} else {
		args := config.Split2PartsBySpace(argstr)
		var err error
		if frame, err = strconv.Atoi(args[0]); err != nil {
			return err
		}
		if len(args) > 1 {
			arg = args[1]
		}
	}
	switch direction {
	case frameUp:
		frame = s.frame + frame
	case frameDown:
		frame = s.frame - frame
	}
	if len(arg) > 0 {
		ctx.Scope.Frame = frame
		return s.CallWithContext(arg, t, ctx)
	}
	if frame < 0 {
		return fmt.Errorf("Invalid frame %d", frame)
	}
	stack, err := t.client.Stacktrace(ctx.Scope.GoroutineID, frame, 0, nil)
	if err != nil {
		return err
	}
	if frame >= len(stack) {
		return fmt.Errorf("Invalid frame %d", frame)
	}
	s.frame = frame
	state, err := t.client.GetState()
	if err != nil {
		return err
	}
	printcontext(t, state)
	th := stack[frame]
	fmt.Fprintf(t.stdout, "Frame %d: %s:%d (PC: %x)\n", frame, t.formatPath(th.File), th.Line, th.PC)
	printfile(t, th.File, th.Line, true)
	return nil
}

func (s *DebugCommands) deferredCommand(t *Session, ctx callContext, argstr string) error {
	ctx.Prefix = deferredPrefix

	space := strings.IndexRune(argstr, ' ')
	if space < 0 {
		return errors.New("not enough arguments")
	}

	var err error
	ctx.Scope.DeferredCall, err = strconv.Atoi(argstr[:space])
	if err != nil {
		return err
	}
	if ctx.Scope.DeferredCall <= 0 {
		return errors.New("argument of deferred must be a number greater than 0 (use 'stack -defer' to see the list of deferred calls)")
	}
	return s.CallWithContext(argstr[space:], t, ctx)
}

func printscope(t *Session) error {
	state, err := t.client.GetState()
	if err != nil {
		return err
	}

	fmt.Fprintf(t.stdout, "Thread %s\n", t.formatThread(state.CurrentThread))
	if state.SelectedGoroutine != nil {
		writeGoroutineLong(t, t.stdout, state.SelectedGoroutine, "")
	}
	return nil
}

func (t *Session) formatThread(th *api.Thread) string {
	if th == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d at %s:%d", th.ID, t.formatPath(th.File), th.Line)
}

func (t *Session) formatLocation(loc api.Location) string {
	return fmt.Sprintf("%s:%d %s (%#v)", t.formatPath(loc.File), loc.Line, loc.Function.Name(), loc.PC)
}

func (t *Session) formatGoroutine(g *api.Goroutine, fgl api.FormatGoroutineLoc) string {
	if g == nil {
		return "<nil>"
	}
	if g.Unreadable != "" {
		return fmt.Sprintf("(unreadable %s)", g.Unreadable)
	}
	var locname string
	var loc api.Location
	switch fgl {
	case api.FglRuntimeCurrent:
		locname = "Runtime"
		loc = g.CurrentLoc
	case api.FglUserCurrent:
		locname = "User"
		loc = g.UserCurrentLoc
	case api.FglGo:
		locname = "Go"
		loc = g.GoStatementLoc
	case api.FglStart:
		locname = "Start"
		loc = g.StartLoc
	}

	buf := new(strings.Builder)
	fmt.Fprintf(buf, "%d - %s: %s", g.ID, locname, t.formatLocation(loc))
	if g.ThreadID != 0 {
		fmt.Fprintf(buf, " (thread %d)", g.ThreadID)
	}

	if (g.Status == api.GoroutineWaiting || g.Status == api.GoroutineSyscall) && g.WaitReason != 0 {
		var wr string
		if g.WaitReason > 0 && g.WaitReason < int64(len(waitReasonStrings)) {
			wr = waitReasonStrings[g.WaitReason]
		} else {
			wr = fmt.Sprintf("unknown wait reason %d", g.WaitReason)
		}
		fmt.Fprintf(buf, " [%s", wr)
		if g.WaitSince > 0 {
			fmt.Fprintf(buf, " %d", g.WaitSince)
		}
		fmt.Fprintf(buf, "]")
	}

	return buf.String()
}

var waitReasonStrings = [...]string{
	"",
	"GC assist marking",
	"IO wait",
	"chan receive (nil chan)",
	"chan send (nil chan)",
	"dumping heap",
	"garbage collection",
	"garbage collection scan",
	"panicwait",
	"select",
	"select (no cases)",
	"GC assist wait",
	"GC sweep wait",
	"GC scavenge wait",
	"chan receive",
	"chan send",
	"finalizer wait",
	"force gc (idle)",
	"semacquire",
	"sleep",
	"sync.Cond.Wait",
	"timer goroutine (idle)",
	"trace reader (blocked)",
	"wait for GC cycle",
	"GC worker (idle)",
	"preempted",
	"debug call",
	"GC mark termination",
	"stopping the world",
	"flushing proc caches",
	"trace goroutine status",
	"trace proc status",
	"page trace flush",
	"coroutine",
}

func writeGoroutineLong(t *Session, w io.Writer, g *api.Goroutine, prefix string) {
	fmt.Fprintf(w, "%sGoroutine %d:\n%s\tRuntime: %s\n%s\tUser: %s\n%s\tGo: %s\n%s\tStart: %s\n",
		prefix, g.ID,
		prefix, t.formatLocation(g.CurrentLoc),
		prefix, t.formatLocation(g.UserCurrentLoc),
		prefix, t.formatLocation(g.GoStatementLoc),
		prefix, t.formatLocation(g.StartLoc))
	writeGoroutineLabels(w, g, prefix+"\t")
}

func writeGoroutineLabels(w io.Writer, g *api.Goroutine, prefix string) {
	const maxNumberOfGoroutineLabels = 5

	if len(g.Labels) <= 0 {
		return
	}

	keys := make([]string, 0, len(g.Labels))
	for k := range g.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	more := false
	if len(keys) > maxNumberOfGoroutineLabels {
		more = true
		keys = keys[:maxNumberOfGoroutineLabels]
	}
	fmt.Fprintf(w, "%sLabels: ", prefix)
	for i, k := range keys {
		fmt.Fprintf(w, "%q:%q", k, g.Labels[k])
		if i != len(keys)-1 {
			fmt.Fprintf(w, ", ")
		} else if more {
			fmt.Fprintf(w, "... (%d more)", len(g.Labels)-maxNumberOfGoroutineLabels)
		}
	}
	fmt.Fprintf(w, "\n")
}

func restart(t *Session, ctx callContext, args string) error {
	t.oldPid = 0
	resetArgs, newArgv, newRedirects, err := parseNewArgv(args)
	if err != nil {
		return err
	}

	if err := restartIntl(t, false, "", resetArgs, newArgv, newRedirects); err != nil {
		return err
	}

	fmt.Fprintln(t.stdout, "Process restarted with PID", t.client.ProcessPid())
	return nil
}

// parseOptionalCount parses an optional count argument.
// If there are not arguments, a value of 1 is returned as the default.
func parseOptionalCount(arg string) (int64, error) {
	if len(arg) == 0 {
		return 1, nil
	}
	return strconv.ParseInt(arg, 0, 64)
}

func restartIntl(t *Session, rerecord bool, restartPos string, resetArgs bool, newArgv []string, newRedirects [3]string) error {
	discarded, err := t.client.RestartFrom(rerecord, restartPos, resetArgs, newArgv, newRedirects, false)
	if err != nil {
		return err
	}
	for i := range discarded {
		fmt.Fprintf(t.stdout, "Discarded %s at %s: %v\n", formatBreakpointName(discarded[i].Breakpoint, false), t.formatBreakpointLocation(discarded[i].Breakpoint), discarded[i].Reason)
	}
	return nil
}

func parseNewArgv(args string) (resetArgs bool, newArgv []string, newRedirects [3]string, err error) {
	if args == "" {
		return false, nil, [3]string{}, nil
	}
	v, err := argv.Argv(args,
		func(s string) (string, error) {
			return "", fmt.Errorf("Backtick not supported in '%s'", s)
		},
		nil)
	if err != nil {
		return false, nil, [3]string{}, err
	}
	if len(v) != 1 {
		return false, nil, [3]string{}, fmt.Errorf("illegal commandline '%s'", args)
	}
	w := v[0]
	if len(w) == 0 {
		return false, nil, [3]string{}, nil
	}
	if w[0] == "-noargs" {
		if len(w) > 1 {
			return false, nil, [3]string{}, errors.New("too many arguments to restart")
		}
		return true, nil, [3]string{}, nil
	}
	redirs := [3]string{}
	for len(w) > 0 {
		var found bool
		var err error
		w, found, err = parseOneRedirect(w, &redirs)
		if err != nil {
			return false, nil, [3]string{}, err
		}
		if !found {
			break
		}
	}
	return true, w, redirs, nil
}

func parseOneRedirect(w []string, redirs *[3]string) ([]string, bool, error) {
	prefixes := []string{"<", ">", "2>"}
	names := []string{"stdin", "stdout", "stderr"}
	if len(w) >= 2 {
		for _, prefix := range prefixes {
			if w[len(w)-2] == prefix {
				w[len(w)-2] += w[len(w)-1]
				w = w[:len(w)-1]
				break
			}
		}
	}
	for i, prefix := range prefixes {
		if strings.HasPrefix(w[len(w)-1], prefix) {
			if redirs[i] != "" {
				return nil, false, fmt.Errorf("redirect error: %s redirected twice", names[i])
			}
			redirs[i] = w[len(w)-1][len(prefix):]
			return w[:len(w)-1], true, nil
		}
	}
	return w, false, nil
}

func printcontextNoState(t *Session) {
	state, _ := t.client.GetState()
	if state == nil || state.CurrentThread == nil {
		return
	}
	printcontext(t, state)
}

func (s *DebugCommands) rebuild(t *Session, ctx callContext, args string) error {
	defer t.onStop()
	discarded, err := t.client.Restart(true)
	if len(discarded) > 0 {
		fmt.Fprintf(t.stdout, "not all breakpoints could be restored.")
	}
	return err
}

func (s *DebugCommands) cont(t *Session, ctx callContext, args string) error {
	if args != "" {
		tmp, err := setBreakpoint(t, ctx, false, args)
		if err != nil {
			if !strings.Contains(err.Error(), "Breakpoint exists") {
				return err
			}
		}
		defer func() {
			for _, bp := range tmp {
				if _, err := t.client.ClearBreakpoint(bp.ID); err != nil {
					fmt.Fprintf(t.stdout, "failed to clear temporary breakpoint: %d", bp.ID)
				}
			}
		}()
	}
	defer t.onStop()
	s.frame = 0
	stateChan := t.client.Continue()
	var state *api.DebuggerState
	for state = range stateChan {
		if state.Err != nil {
			printcontextNoState(t)
			return state.Err
		}
		printcontext(t, state)
	}
	printPos(t, state.CurrentThread, printPosShowArrow)
	return nil
}

func continueUntilCompleteNext(t *Session, state *api.DebuggerState, op string, shouldPrintFile bool) error {
	defer t.onStop()
	if !state.NextInProgress {
		if shouldPrintFile {
			printPos(t, state.CurrentThread, printPosShowArrow)
		}
		return nil
	}
	skipBreakpoints := false
	for {
		fmt.Fprintf(t.stdout, "\tbreakpoint hit during %s", op)
		if !skipBreakpoints {
			fmt.Fprintf(t.stdout, "\n")
			answer, err := promptAutoContinue(t, op)
			switch answer {
			case "f": // finish next
				skipBreakpoints = true
				fallthrough
			case "c": // continue once
				fmt.Fprintf(t.stdout, "continuing...\n")
			case "s": // stop and cancel
				fallthrough
			default:
				t.client.CancelNext()
				printPos(t, state.CurrentThread, printPosShowArrow)
				return err
			}
		} else {
			fmt.Fprintf(t.stdout, ", continuing...\n")
		}
		stateChan := t.client.Continue()
		var state *api.DebuggerState
		for state = range stateChan {
			if state.Err != nil {
				printcontextNoState(t)
				return state.Err
			}
			printcontext(t, state)
		}
		if !state.NextInProgress {
			printPos(t, state.CurrentThread, printPosShowArrow)
			return nil
		}
	}
}

func promptAutoContinue(t *Session, op string) (string, error) {
	for {
		answer, err := t.line.Prompt(fmt.Sprintf("[c] continue [s] stop here and cancel %s, [f] finish %s skipping all breakpoints? ", op, op))
		if err != nil {
			return "", err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		switch answer {
		case "f", "c", "s":
			return answer, nil
		}
	}
}

func scopePrefixSwitch(t *Session, ctx callContext) error {
	if ctx.Scope.GoroutineID > 0 {
		_, err := t.client.SwitchGoroutine(ctx.Scope.GoroutineID)
		if err != nil {
			return err
		}
	}
	return nil
}

func exitedToError(state *api.DebuggerState, err error) (*api.DebuggerState, error) {
	if err == nil && state.Exited {
		return nil, fmt.Errorf("Process %d has exited with status %d", state.Pid, state.ExitStatus)
	}
	return state, err
}

func (s *DebugCommands) step(t *Session, ctx callContext, args string) error {
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	s.frame = 0
	stepfn := t.client.Step
	state, err := exitedToError(stepfn())
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	return continueUntilCompleteNext(t, state, "step", true)
}

var errNotOnFrameZero = errors.New("not on topmost frame")

// stepInstruction implements the step-instruction (stepi) command.
func (s *DebugCommands) stepInstruction(t *Session, ctx callContext, args string) error {
	return stepInstruction(t, ctx, s.frame, false)
}

// nextInstruction implements the next-instruction (nexti) command.
func (s *DebugCommands) nextInstruction(t *Session, ctx callContext, args string) error {
	return stepInstruction(t, ctx, s.frame, true)
}

func stepInstruction(t *Session, ctx callContext, frame int, skipCalls bool) error {
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	if frame != 0 {
		return errNotOnFrameZero
	}

	defer t.onStop()

	fn := t.client.StepInstruction

	state, err := exitedToError(fn(skipCalls))
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	printPos(t, state.CurrentThread, printPosShowArrow|printPosStepInstruction)
	return nil
}

func (s *DebugCommands) next(t *Session, ctx callContext, args string) error {
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	if s.frame != 0 {
		return errNotOnFrameZero
	}

	nextfn := t.client.Next

	var count int64
	var err error
	if count, err = parseOptionalCount(args); err != nil {
		return err
	} else if count <= 0 {
		return errors.New("Invalid next count")
	}
	for ; count > 0; count-- {
		state, err := exitedToError(nextfn())
		if err != nil {
			printcontextNoState(t)
			return err
		}
		// If we're about the exit the loop, print the context.
		finishedNext := count == 1
		if finishedNext {
			printcontext(t, state)
		}
		if err := continueUntilCompleteNext(t, state, "next", finishedNext); err != nil {
			return err
		}
	}
	return nil
}

func (s *DebugCommands) stepout(t *Session, ctx callContext, args string) error {
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	if s.frame != 0 {
		return errNotOnFrameZero
	}

	stepoutfn := t.client.StepOut

	state, err := exitedToError(stepoutfn())
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	return continueUntilCompleteNext(t, state, "stepout", true)
}

func (s *DebugCommands) call(t *Session, ctx callContext, args string) error {
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	const unsafePrefix = "-unsafe "
	unsafe := false
	if strings.HasPrefix(args, unsafePrefix) {
		unsafe = true
		args = args[len(unsafePrefix):]
	}
	state, err := exitedToError(t.client.Call(ctx.Scope.GoroutineID, args, unsafe))
	s.frame = 0
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	return continueUntilCompleteNext(t, state, "call", true)
}

func clear(t *Session, ctx callContext, args string) error {
	if len(args) == 0 {
		return errors.New("not enough arguments")
	}
	id, err := strconv.Atoi(args)
	var bp *api.Breakpoint
	if err == nil {
		bp, err = t.client.ClearBreakpoint(id)
	} else {
		bp, err = t.client.ClearBreakpointByName(args)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(t.stdout, "%s cleared at %s\n", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	return nil
}

func clearAll(t *Session, ctx callContext, args string) error {
	breakPoints, err := t.client.ListBreakpoints(false)
	if err != nil {
		return err
	}

	var locPCs map[uint64]struct{}
	if args != "" {
		locs, _, err := t.client.FindLocation(api.EvalScope{GoroutineID: -1, Frame: 0}, args, true, t.substitutePathRules())
		if err != nil {
			return err
		}
		locPCs = make(map[uint64]struct{})
		for _, loc := range locs {
			for _, pc := range loc.PCs {
				locPCs[pc] = struct{}{}
			}
			locPCs[loc.PC] = struct{}{}
		}
	}

	for _, bp := range breakPoints {
		if locPCs != nil {
			if _, ok := locPCs[bp.Addr]; !ok {
				continue
			}
		}

		if bp.ID < 0 {
			continue
		}

		_, err := t.client.ClearBreakpoint(bp.ID)
		if err != nil {
			fmt.Fprintf(t.stdout, "Couldn't delete %s at %s: %s\n", formatBreakpointName(bp, false), t.formatBreakpointLocation(bp), err)
		}
		fmt.Fprintf(t.stdout, "%s cleared at %s\n", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	}
	return nil
}

func toggle(t *Session, ctx callContext, args string) error {
	if args == "" {
		return errors.New("not enough arguments")
	}
	id, err := strconv.Atoi(args)
	var bp *api.Breakpoint
	if err == nil {
		bp, err = t.client.ToggleBreakpoint(id)
	} else {
		bp, err = t.client.ToggleBreakpointByName(args)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(t.stdout, "%s toggled at %s\n", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	return nil
}

func breakpoints(t *Session, ctx callContext, args string) error {
	breakPoints, err := t.client.ListBreakpoints(args == "-a")
	if err != nil {
		return err
	}
	slices.SortFunc(breakPoints, func(a, b *api.Breakpoint) int { return cmp.Compare(a.ID, b.ID) })
	for _, bp := range breakPoints {
		enabled := "(enabled)"
		if bp.Disabled {
			enabled = "(disabled)"
		} else if bp.ExprString != "" {
			enabled = "(suspended)"
		}
		fmt.Fprintf(t.stdout, "%s %s", formatBreakpointName(bp, true), enabled)
		if bp.ExprString != "" {
			fmt.Fprintf(t.stdout, " at %s\n", bp.ExprString)
		} else {
			fmt.Fprintf(t.stdout, " at %v (%d)\n", t.formatBreakpointLocation(bp), bp.TotalHitCount)
		}

		attrs := formatBreakpointAttrs("\t", bp, false)

		if len(attrs) > 0 {
			fmt.Fprintf(t.stdout, "%s\n", strings.Join(attrs, "\n"))
		}
	}
	return nil
}

func formatBreakpointAttrs(prefix string, bp *api.Breakpoint, includeTrace bool) []string {
	var attrs []string
	if bp.Cond != "" {
		attrs = append(attrs, fmt.Sprintf("%scond %s", prefix, bp.Cond))
	}
	if bp.HitCond != "" {
		if bp.HitCondPerG {
			attrs = append(attrs, fmt.Sprintf("%scond -per-g-hitcount %s", prefix, bp.HitCond))
		} else {
			attrs = append(attrs, fmt.Sprintf("%scond -hitcount %s", prefix, bp.HitCond))
		}
	}
	if bp.Stacktrace > 0 {
		attrs = append(attrs, fmt.Sprintf("%sstack %d", prefix, bp.Stacktrace))
	}
	if bp.Goroutine {
		attrs = append(attrs, fmt.Sprintf("%sgoroutine", prefix))
	}
	if bp.LoadArgs != nil {
		if *(bp.LoadArgs) == longLoadConfig {
			attrs = append(attrs, fmt.Sprintf("%sargs -v", prefix))
		} else {
			attrs = append(attrs, fmt.Sprintf("%sargs", prefix))
		}
	}
	if bp.LoadLocals != nil {
		if *(bp.LoadLocals) == longLoadConfig {
			attrs = append(attrs, fmt.Sprintf("%slocals -v", prefix))
		} else {
			attrs = append(attrs, fmt.Sprintf("%slocals", prefix))
		}
	}
	for i := range bp.Variables {
		attrs = append(attrs, fmt.Sprintf("%sprint %s", prefix, bp.Variables[i]))
	}
	if includeTrace && bp.Tracepoint {
		attrs = append(attrs, fmt.Sprintf("%strace", prefix))
	}
	for i := range bp.VerboseDescr {
		attrs = append(attrs, fmt.Sprintf("%s%s", prefix, bp.VerboseDescr[i]))
	}
	return attrs
}

// `argstr` is the raw input string of `break [argstr]`. dlv setBreakpoint parsing
// argstr logic is really complex, here we change `dlv> break [name] [locspec] [if condition]`
// to `tinydbg> break [--name=name] [locspec] [if condition]`.
// After this, we remove some confusing logic in parsing. We only need to consider either of
// `[locspec] [if condition]` exists or not. So it's easier to understand.
//
// OK, there're following cases to support:
//
// with --name:
// - break --name=? locspec if <condition>
// - break --name=? locspec
// - break --name=? if <condition>
// - break --name=?
//
// without --name:
// - break locspec if <condition>
// - break locspec
// - break if <condition>
// - break
func setBreakpoint(t *Session, ctx callContext, tracepoint bool, argstr string) ([]*api.Breakpoint, error) {
	// parsed --name, args and error
	name, spec, cond, err := parseBreakpointArgs(argstr)
	if err != nil {
		return nil, err
	}

	requestedBp := &api.Breakpoint{
		Name:       name,
		Tracepoint: tracepoint,
		Cond:       cond,
	}
	locs, substSpec, findLocErr := t.client.FindLocation(ctx.Scope, spec, true, t.substitutePathRules())

	if findLocErr != nil && shouldAskToSuspendBreakpoint(t) {
		fmt.Fprintf(os.Stderr, "Command failed: %s\n", findLocErr.Error())
		question := "Set a suspended breakpoint (Delve will try to set this breakpoint when a plugin is loaded) [Y/n]?"
		if isErrProcessExited(findLocErr) {
			question = "Set a suspended breakpoint (Delve will try to set this breakpoint when the process is restarted) [Y/n]?"
		}
		answer, err := yesno(t.line, question, "yes")
		if err != nil {
			return nil, err
		}
		if !answer {
			return nil, nil
		}
		findLocErr = nil
		bp, err := t.client.CreateBreakpointWithExpr(requestedBp, spec, t.substitutePathRules(), true)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(t.stdout, "%s set at %s\n", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
		return nil, nil
	}
	if findLocErr != nil {
		return nil, findLocErr
	}
	if substSpec != "" {
		spec = substSpec
	}

	created := []*api.Breakpoint{}
	for _, loc := range locs {
		requestedBp.Addr = loc.PC
		requestedBp.Addrs = loc.PCs
		requestedBp.AddrPid = loc.PCPids
		if tracepoint {
			requestedBp.LoadArgs = &ShortLoadConfig
		}

		requestedBp.Cond = cond
		bp, err := t.client.CreateBreakpointWithExpr(requestedBp, spec, t.substitutePathRules(), false)
		if err != nil {
			return nil, err
		}
		created = append(created, bp)

		fmt.Fprintf(t.stdout, "%s set at %s\n", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	}

	var shouldSetReturnBreakpoints bool
	loc, err := locspec.Parse(spec)
	if err != nil {
		return nil, err
	}
	switch t := loc.(type) {
	case *locspec.NormalLocationSpec:
		shouldSetReturnBreakpoints = t.LineOffset == -1 && t.FuncBase != nil
	case *locspec.RegexLocationSpec:
		shouldSetReturnBreakpoints = true
	}
	if tracepoint && shouldSetReturnBreakpoints && locs[0].Function != nil {
		for i := range locs {
			if locs[i].Function == nil {
				continue
			}
			addrs, err := t.client.(*rpc2.RPCClient).FunctionReturnLocations(locs[0].Function.Name())
			if err != nil {
				return nil, err
			}
			for j := range addrs {
				_, err = t.client.CreateBreakpoint(&api.Breakpoint{
					Addr:        addrs[j],
					TraceReturn: true,
					Line:        -1,
					LoadArgs:    &ShortLoadConfig,
				})
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return created, nil
}

func breakpoint(t *Session, ctx callContext, args string) error {
	_, err := setBreakpoint(t, ctx, false, args)
	return err
}

func tracepoint(t *Session, ctx callContext, args string) error {
	if ctx.Prefix == onPrefix {
		if args != "" {
			return errors.New("too many arguments to trace")
		}
		ctx.Breakpoint.Tracepoint = true
		return nil
	}
	_, err := setBreakpoint(t, ctx, true, args)
	return err
}

func getEditorName() (string, []string, error) {
	var editor string
	if editor = os.Getenv("DELVE_EDITOR"); editor == "" {
		if editor = os.Getenv("EDITOR"); editor == "" {
			return "", nil, errors.New("Neither DELVE_EDITOR or EDITOR is set")
		}
	}

	editorParts := strings.Fields(editor)
	editor = editorParts[0]

	var userArgs []string
	if len(editorParts) > 1 {
		userArgs = editorParts[1:]
	}

	return editor, userArgs, nil
}

func runEditor(args ...string) error {
	editor, userArgs, err := getEditorName()
	if err != nil {
		return err
	}
	allArgs := append(userArgs, args...)
	cmd := exec.Command(editor, allArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func edit(t *Session, ctx callContext, args string) error {
	file, lineno, _, err := getLocation(t, ctx, args, false)
	if err != nil {
		return err
	}
	editor, _, err := getEditorName()
	if err != nil {
		return err
	}
	switch editor {
	case "code":
		return runEditor("--goto", fmt.Sprintf("%s:%d", file, lineno))
	case "zed":
		return runEditor(fmt.Sprintf("%s:%d:0", file, lineno))
	case "hx":
		return runEditor(fmt.Sprintf("%s:%d", file, lineno))
	case "vi", "vim", "nvim":
		return runEditor(fmt.Sprintf("+%d", lineno), file)
	default:
		return runEditor(fmt.Sprintf("+%d", lineno), file)
	}
}

func watchpoint(t *Session, ctx callContext, args string) error {
	v := strings.SplitN(args, " ", 2)
	if len(v) != 2 {
		return errors.New("wrong number of arguments: watch [-r|-w|-rw] <expr>")
	}
	var wtype api.WatchType
	switch v[0] {
	case "-r":
		wtype = api.WatchRead
	case "-w":
		wtype = api.WatchWrite
	case "-rw":
		wtype = api.WatchRead | api.WatchWrite
	default:
		return fmt.Errorf("wrong argument %q to watch", v[0])
	}
	bp, err := t.client.CreateWatchpoint(ctx.Scope, v[1], wtype)
	if err != nil {
		return err
	}
	fmt.Fprintf(t.stdout, "%s set at %s\n", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	return nil
}

func examineMemoryCmd(t *Session, ctx callContext, argstr string) error {
	var (
		address uint64
		err     error
		ok      bool
		args    = strings.Split(argstr, " ")
	)

	// Default value
	priFmt := byte('x')
	count := int64(1)
	size := int64(1)
	isExpr := false
	rawout := false

	// nextArg returns the next argument that is not an empty string, if any, and
	// advances the args slice to the position after that.
	nextArg := func() string {
		for len(args) > 0 {
			arg := args[0]
			args = args[1:]
			if arg != "" {
				return arg
			}
		}
		return ""
	}

loop:
	for {
		switch cmd := nextArg(); cmd {
		case "":
			// no more arguments
			break loop
		case "-fmt":
			arg := nextArg()
			if arg == "" {
				return errors.New("expected argument after -fmt")
			}
			if arg == "raw" {
				rawout = true
			} else {
				fmtMapToPriFmt := map[string]byte{
					"oct":         'o',
					"octal":       'o',
					"hex":         'x',
					"hexadecimal": 'x',
					"dec":         'd',
					"decimal":     'd',
					"bin":         'b',
					"binary":      'b',
				}
				priFmt, ok = fmtMapToPriFmt[arg]
				if !ok {
					return fmt.Errorf("%q is not a valid format", arg)
				}
			}
		case "-count", "-len":
			arg := nextArg()
			if arg == "" {
				return errors.New("expected argument after -count/-len")
			}
			var err error
			count, err = strconv.ParseInt(arg, 0, 64)
			if err != nil || count <= 0 {
				return errors.New("count/len must be a positive integer")
			}
		case "-size":
			arg := nextArg()
			if arg == "" {
				return errors.New("expected argument after -size")
			}
			var err error
			size, err = strconv.ParseInt(arg, 0, 64)
			if err != nil || size <= 0 || size > 8 {
				return errors.New("size must be a positive integer (<=8)")
			}
		case "-x":
			isExpr = true
			break loop // remaining args are going to be interpreted as expression
		default:
			if len(args) > 0 {
				return fmt.Errorf("unknown option %q", args[0])
			}
			args = []string{cmd}
			break loop // only one arg left to be evaluated as a uint
		}
	}

	if len(args) == 0 {
		return errors.New("no address specified")
	}

	if isExpr {
		expr := strings.Join(args, " ")
		val, err := t.client.EvalVariable(ctx.Scope, expr, t.loadConfig())
		if err != nil {
			return err
		}

		// "-x &myVar" or "-x myPtrVar"
		if val.Kind == reflect.Ptr {
			if len(val.Children) < 1 {
				return fmt.Errorf("bug? invalid pointer: %#v", val)
			}
			address = val.Children[0].Addr
			// "-x 0xc000079f20 + 8" or -x 824634220320 + 8
		} else if val.Kind == reflect.Int && val.Value != "" {
			address, err = strconv.ParseUint(val.Value, 0, 64)
			if err != nil {
				return fmt.Errorf("bad expression result: %q: %s", val.Value, err)
			}
		} else {
			return fmt.Errorf("unsupported expression type: %s", val.Kind)
		}
	} else {
		address, err = strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			return fmt.Errorf("convert address into uintptr type failed, %s", err)
		}
	}

	start := address
	remsz := int(count * size)

	for remsz > 0 {
		reqsz := rpc2.ExamineMemoryLengthLimit
		if reqsz > remsz {
			reqsz = remsz
		}
		memArea, isLittleEndian, err := t.client.ExamineMemory(start, reqsz)
		if err != nil {
			return err
		}
		if rawout {
			t.stdout.Write(memArea)
		} else {
			fmt.Fprint(t.stdout, api.PrettyExamineMemory(uintptr(start), memArea, isLittleEndian, priFmt, int(size)))
		}
		start += uint64(reqsz)
		remsz -= reqsz
	}
	return nil
}

func parseFormatArg(args string) (fmtstr, argsOut string) {
	if len(args) < 1 || args[0] != '%' {
		return "", args
	}
	v := strings.SplitN(args, " ", 2)
	if len(v) == 1 {
		return v[0], ""
	}
	return v[0], v[1]
}

const maxPrintVarChanGoroutines = 100

func (s *DebugCommands) printVar(t *Session, ctx callContext, args string) error {
	if len(args) == 0 {
		return errors.New("not enough arguments")
	}
	if ctx.Prefix == onPrefix {
		ctx.Breakpoint.Variables = append(ctx.Breakpoint.Variables, args)
		return nil
	}
	fmtstr, args := parseFormatArg(args)
	val, err := t.client.EvalVariable(ctx.Scope, args, t.loadConfig())
	if err != nil {
		return err
	}

	fmt.Fprintln(t.stdout, val.MultilineString("", fmtstr))

	if val.Kind == reflect.Chan {
		fmt.Fprintln(t.stdout)
		gs, _, _, _, err := t.client.ListGoroutinesWithFilter(0, maxPrintVarChanGoroutines, []api.ListGoroutinesFilter{{Kind: api.GoroutineWaitingOnChannel, Arg: fmt.Sprintf("*(*%q)(%#x)", val.Type, val.Addr)}}, nil, &ctx.Scope)
		if err != nil {
			fmt.Fprintf(t.stdout, "Error reading channel wait queue: %v", err)
		} else {
			fmt.Fprintln(t.stdout, "Goroutines waiting on this channel:")
			state, err := t.client.GetState()
			if err != nil {
				fmt.Fprintf(t.stdout, "Error printing channel wait queue: %v", err)
			}
			var done bool
			s.printGoroutines(t, ctx, "", gs, api.FglUserCurrent, 0, 0, "", &done, state)
		}
	}
	return nil
}

func whatisCommand(t *Session, ctx callContext, args string) error {
	if len(args) == 0 {
		return errors.New("not enough arguments")
	}
	val, err := t.client.EvalVariable(ctx.Scope, args, ShortLoadConfig)
	if err != nil {
		return err
	}
	if val.Flags&api.VariableCPURegister != 0 {
		fmt.Fprintln(t.stdout, "CPU Register")
		return nil
	}
	if val.Type != "" {
		fmt.Fprintln(t.stdout, val.Type)
	}
	if val.RealType != val.Type {
		fmt.Fprintf(t.stdout, "Real type: %s\n", val.RealType)
	}
	if val.Kind == reflect.Interface && len(val.Children) > 0 {
		fmt.Fprintf(t.stdout, "Concrete type: %s\n", val.Children[0].Type)
	}
	if t.conf.ShowLocationExpr && val.LocationExpr != "" {
		fmt.Fprintf(t.stdout, "location: %s\n", val.LocationExpr)
	}
	return nil
}

func setVar(t *Session, ctx callContext, args string) error {
	// HACK: in go '=' is not an operator, we detect the error and try to recover from it by splitting the input string
	_, err := parser.ParseExpr(args)
	if err == nil {
		return errors.New("syntax error '=' not found")
	}

	el, ok := err.(scanner.ErrorList)
	if !ok || el[0].Msg != "expected '==', found '='" {
		return err
	}

	lexpr := args[:el[0].Pos.Offset]
	rexpr := args[el[0].Pos.Offset+1:]
	return t.client.SetVariable(ctx.Scope, lexpr, rexpr)
}

func (t *Session) printFilteredVariables(varType string, vars []api.Variable, filter string, cfg api.LoadConfig) error {
	reg, err := regexp.Compile(filter)
	if err != nil {
		return err
	}
	match := false
	for _, v := range vars {
		if reg == nil || reg.Match([]byte(v.Name)) {
			match = true
			name := v.Name
			if v.Flags&api.VariableShadowed != 0 {
				name = "(" + name + ")"
			}
			if cfg == ShortLoadConfig {
				fmt.Fprintf(t.stdout, "%s = %s\n", name, v.SinglelineString())
			} else {
				fmt.Fprintf(t.stdout, "%s = %s\n", name, v.MultilineString("", ""))
			}
		}
	}
	if !match {
		fmt.Fprintf(t.stdout, "(no %s)\n", varType)
	}
	return nil
}

func (t *Session) printSortedStrings(v []string, err error) error {
	if err != nil {
		return err
	}
	sort.Strings(v)
	done := false
	for _, d := range v {
		if done {
			break
		}
		fmt.Fprintln(t.stdout, d)
	}
	return nil
}

func sources(t *Session, ctx callContext, args string) error {
	return t.printSortedStrings(t.client.ListSources(args))
}

func packages(t *Session, ctx callContext, args string) error {
	info, err := t.client.ListPackagesBuildInfo(args, false)
	if err != nil {
		return err
	}
	pkgs := make([]string, 0, len(info))
	for _, i := range info {
		pkgs = append(pkgs, i.ImportPath)
	}
	return t.printSortedStrings(pkgs, nil)
}

func funcs(t *Session, ctx callContext, args string) error {
	return t.printSortedStrings(t.client.ListFunctions(args, 0))
}

func types(t *Session, ctx callContext, args string) error {
	return t.printSortedStrings(t.client.ListTypes(args))
}

func parseVarArguments(args string, t *Session) (filter string, cfg api.LoadConfig) {
	if v := config.Split2PartsBySpace(args); len(v) >= 1 && v[0] == "-v" {
		if len(v) == 2 {
			return v[1], t.loadConfig()
		} else {
			return "", t.loadConfig()
		}
	}
	return args, ShortLoadConfig
}

func args(t *Session, ctx callContext, args string) error {
	filter, cfg := parseVarArguments(args, t)
	if ctx.Prefix == onPrefix {
		if filter != "" {
			return errors.New("filter not supported on breakpoint")
		}
		ctx.Breakpoint.LoadArgs = &cfg
		return nil
	}
	vars, err := t.client.ListFunctionArgs(ctx.Scope, cfg)
	if err != nil {
		return err
	}
	return t.printFilteredVariables("args", vars, filter, cfg)
}

func locals(t *Session, ctx callContext, args string) error {
	filter, cfg := parseVarArguments(args, t)
	if ctx.Prefix == onPrefix {
		if filter != "" {
			return errors.New("filter not supported on breakpoint")
		}
		ctx.Breakpoint.LoadLocals = &cfg
		return nil
	}
	locals, err := t.client.ListLocalVariables(ctx.Scope, cfg)
	if err != nil {
		return err
	}
	return t.printFilteredVariables("locals", locals, filter, cfg)
}

func vars(t *Session, ctx callContext, args string) error {
	filter, cfg := parseVarArguments(args, t)
	vars, err := t.client.ListPackageVariables(filter, cfg)
	if err != nil {
		return err
	}
	return t.printFilteredVariables("vars", vars, filter, cfg)
}

func regs(t *Session, ctx callContext, args string) error {
	includeFp := false
	if args == "-a" {
		includeFp = true
	}
	var regs api.Registers
	var err error
	if ctx.Scope.GoroutineID < 0 && ctx.Scope.Frame == 0 {
		regs, err = t.client.ListThreadRegisters(0, includeFp)
	} else {
		regs, err = t.client.ListScopeRegisters(ctx.Scope, includeFp)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(t.stdout, regs)
	return nil
}

func stackCommand(t *Session, ctx callContext, args string) error {
	sa, err := parseStackArgs(args)
	if err != nil {
		return err
	}
	if ctx.Prefix == onPrefix {
		ctx.Breakpoint.Stacktrace = sa.depth
		return nil
	}
	var cfg *api.LoadConfig
	if sa.full {
		cfg = &ShortLoadConfig
	}
	stack, err := t.client.Stacktrace(ctx.Scope.GoroutineID, sa.depth, sa.opts, cfg)
	if err != nil {
		return err
	}
	printStack(t, t.stdout, stack, "", sa.offsets)
	if sa.ancestors > 0 {
		ancestors, err := t.client.Ancestors(ctx.Scope.GoroutineID, sa.ancestors, sa.ancestorDepth)
		if err != nil {
			return err
		}
		for _, ancestor := range ancestors {
			fmt.Fprintf(t.stdout, "Created by Goroutine %d:\n", ancestor.ID)
			if ancestor.Unreadable != "" {
				fmt.Fprintf(t.stdout, "\t%s\n", ancestor.Unreadable)
				continue
			}
			printStack(t, t.stdout, ancestor.Stack, "\t", false)
		}
	}
	return nil
}

type stackArgs struct {
	depth   int
	full    bool
	offsets bool
	opts    api.StacktraceOptions

	ancestors     int
	ancestorDepth int
}

func parseStackArgs(argstr string) (stackArgs, error) {
	r := stackArgs{
		depth: 50,
		full:  false,
	}
	if argstr != "" {
		args := strings.Split(argstr, " ")
		for i := 0; i < len(args); i++ {
			numarg := func(name string) (int, error) {
				if i >= len(args) {
					return 0, fmt.Errorf("expected number after %s", name)
				}
				n, err := strconv.Atoi(args[i])
				if err != nil {
					return 0, fmt.Errorf("expected number after %s: %v", name, err)
				}
				return n, nil
			}
			switch args[i] {
			case "-full":
				r.full = true
			case "-offsets":
				r.offsets = true
			case "-defer":
				r.opts |= api.StacktraceReadDefers
			case "-mode":
				i++
				if i >= len(args) {
					return stackArgs{}, errors.New("expected normal, simple or fromg after -mode")
				}
				switch args[i] {
				case "normal":
					r.opts &^= api.StacktraceSimple
					r.opts &^= api.StacktraceG
				case "simple":
					r.opts |= api.StacktraceSimple
				case "fromg":
					r.opts |= api.StacktraceG | api.StacktraceSimple
				default:
					return stackArgs{}, errors.New("expected normal, simple or fromg after -mode")
				}
			case "-a":
				i++
				n, err := numarg("-a")
				if err != nil {
					return stackArgs{}, err
				}
				r.ancestors = n
			case "-adepth":
				i++
				n, err := numarg("-adepth")
				if err != nil {
					return stackArgs{}, err
				}
				r.ancestorDepth = n
			default:
				n, err := strconv.Atoi(args[i])
				if err != nil {
					return stackArgs{}, errors.New("depth must be a number")
				}
				r.depth = n
			}
		}
	}
	if r.ancestors > 0 && r.ancestorDepth == 0 {
		r.ancestorDepth = r.depth
	}
	return r, nil
}

// getLocation returns the current location or the locations specified by the argument.
// getLocation is used to process the argument of list and edit commands.
func getLocation(t *Session, ctx callContext, args string, showContext bool) (file string, lineno int, showarrow bool, err error) {
	switch {
	case len(args) == 0 && !ctx.scoped():
		state, err := t.client.GetState()
		if err != nil {
			return "", 0, false, err
		}
		if showContext {
			printcontext(t, state)
		}
		if state.SelectedGoroutine != nil {
			return state.SelectedGoroutine.CurrentLoc.File, state.SelectedGoroutine.CurrentLoc.Line, true, nil
		}
		return state.CurrentThread.File, state.CurrentThread.Line, true, nil

	case len(args) == 0 && ctx.scoped():
		locs, err := t.client.Stacktrace(ctx.Scope.GoroutineID, ctx.Scope.Frame, 0, nil)
		if err != nil {
			return "", 0, false, err
		}
		if ctx.Scope.Frame >= len(locs) {
			return "", 0, false, fmt.Errorf("Frame %d does not exist in goroutine %d", ctx.Scope.Frame, ctx.Scope.GoroutineID)
		}
		loc := locs[ctx.Scope.Frame]
		gid := ctx.Scope.GoroutineID
		if gid < 0 {
			state, err := t.client.GetState()
			if err != nil {
				return "", 0, false, err
			}
			if state.SelectedGoroutine != nil {
				gid = state.SelectedGoroutine.ID
			}
		}
		if showContext {
			fmt.Fprintf(t.stdout, "Goroutine %d frame %d at %s:%d (PC: %#x)\n", gid, ctx.Scope.Frame, loc.File, loc.Line, loc.PC)
		}
		return loc.File, loc.Line, true, nil

	default:
		locs, _, err := t.client.FindLocation(ctx.Scope, args, false, t.substitutePathRules())
		if err != nil {
			return "", 0, false, err
		}
		if len(locs) > 1 {
			return "", 0, false, locspec.AmbiguousLocationError{Location: args, CandidatesLocation: locs}
		}
		loc := locs[0]
		if showContext {
			fmt.Fprintf(t.stdout, "Showing %s:%d (PC: %#x)\n", loc.File, loc.Line, loc.PC)
		}
		return loc.File, loc.Line, false, nil
	}
}

func listCommand(t *Session, ctx callContext, args string) error {
	file, lineno, showarrow, err := getLocation(t, ctx, args, true)
	if err != nil {
		return err
	}
	return printfile(t, file, lineno, showarrow)
}

// we remove the support for starlark scripts `*.star` and starlark repl,
// we only leave the ordinary dlv commands saved here.
func (s *DebugCommands) sourceCommand(t *Session, ctx callContext, args string) error {
	if len(args) == 0 {
		return errors.New("wrong number of arguments: source <filename>")
	}

	if runtime.GOOS != "windows" && strings.HasPrefix(args, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if args == "~" {
				args = home
			} else if strings.HasPrefix(args, "~/") {
				args = filepath.Join(home, args[2:])
			}
		}
	}

	return s.executeFile(t, args)
}

var errDisasmUsage = errors.New("wrong number of arguments: disassemble [-a <start> <end>] [-l <locspec>]")

func disassCommand(t *Session, ctx callContext, args string) error {
	var cmd, rest string

	if args != "" {
		argv := config.Split2PartsBySpace(args)
		if len(argv) != 2 {
			return errDisasmUsage
		}
		cmd = argv[0]
		rest = argv[1]
	}

	flavor := t.conf.GetDisassembleFlavour()

	var disasm api.AsmInstructions
	var disasmErr error

	switch cmd {
	case "":
		locs, _, err := t.client.FindLocation(ctx.Scope, "+0", true, t.substitutePathRules())
		if err != nil {
			return err
		}
		disasm, disasmErr = t.client.DisassemblePC(ctx.Scope, locs[0].PC, flavor)
	case "-a":
		v := config.Split2PartsBySpace(rest)
		if len(v) != 2 {
			return errDisasmUsage
		}
		startpc, err := strconv.ParseInt(v[0], 0, 64)
		if err != nil {
			return fmt.Errorf("wrong argument: %q is not a number", v[0])
		}
		endpc, err := strconv.ParseInt(v[1], 0, 64)
		if err != nil {
			return fmt.Errorf("wrong argument: %q is not a number", v[1])
		}
		disasm, disasmErr = t.client.DisassembleRange(ctx.Scope, uint64(startpc), uint64(endpc), flavor)
	case "-l":
		locs, _, err := t.client.FindLocation(ctx.Scope, rest, true, t.substitutePathRules())
		if err != nil {
			return err
		}
		if len(locs) != 1 {
			return errors.New("expression specifies multiple locations")
		}
		disasm, disasmErr = t.client.DisassemblePC(ctx.Scope, locs[0].PC, flavor)
	default:
		return errDisasmUsage
	}

	if disasmErr != nil {
		return disasmErr
	}

	disasmPrint(disasm, t.stdout, true)

	return nil
}

func libraries(t *Session, ctx callContext, args string) error {
	libs, err := t.client.ListDynamicLibraries()
	if err != nil {
		return err
	}
	d := digits(len(libs))
	for i := range libs {
		fmt.Fprintf(t.stdout, "%"+strconv.Itoa(d)+"d. %#x %s\n", i, libs[i].Address, libs[i].Path)
		if libs[i].LoadError != "" {
			fmt.Fprintf(t.stdout, "    Load error: %s\n", libs[i].LoadError)
		}
	}
	return nil
}

func digits(n int) int {
	if n <= 0 {
		return 1
	}
	return int(math.Floor(math.Log10(float64(n)))) + 1
}

func printStack(t *Session, out io.Writer, stack []api.Stackframe, ind string, offsets bool) {
	api.PrintStack(t.formatPath, out, stack, ind, offsets, func(api.Stackframe) bool { return true })
}

func printcontext(t *Session, state *api.DebuggerState) {
	if t.IsTraceNonInteractive() {
		// If we're just running the `trace` subcommand there isn't any need
		// to print out the rest of the state below.
		for i := range state.Threads {
			if state.Threads[i].Breakpoint != nil {
				printcontextThread(t, state.Threads[i])
			}
		}
		return
	}

	if state.Pid != t.oldPid {
		if t.oldPid != 0 {
			fmt.Fprintf(t.stdout, "Switch target process from %d to %d (%s)\n", t.oldPid, state.Pid, state.TargetCommandLine)
		}
		t.oldPid = state.Pid
	}
	for i := range state.Threads {
		if (state.CurrentThread != nil) && (state.Threads[i].ID == state.CurrentThread.ID) {
			continue
		}
		if state.Threads[i].Breakpoint != nil {
			printcontextThread(t, state.Threads[i])
		}
	}

	if state.CurrentThread == nil {
		fmt.Fprintln(t.stdout, "No current thread available")
		return
	}

	var th *api.Thread
	if state.SelectedGoroutine == nil {
		th = state.CurrentThread
	} else {
		for i := range state.Threads {
			if state.Threads[i].ID == state.SelectedGoroutine.ThreadID {
				th = state.Threads[i]
				break
			}
		}
		if th == nil {
			printcontextLocation(t, state.SelectedGoroutine.CurrentLoc)
			return
		}
	}

	if th.File == "" {
		fmt.Fprintf(t.stdout, "Stopped at: 0x%x\n", state.CurrentThread.PC)
		Print(t.stdout, bytes.NewReader([]byte("no source available")), 1, 10, 1)
		return
	}

	printcontextThread(t, th)

	if state.When != "" {
		fmt.Fprintln(t.stdout, state.When)
	}

	for _, watchpoint := range state.WatchOutOfScope {
		fmt.Fprintf(t.stdout, "%s went out of scope and was cleared\n", formatBreakpointName(watchpoint, true))
	}
}

const optimizedFunctionWarning = "Warning: debugging optimized function"

func printcontextLocation(t *Session, loc api.Location) {
	fmt.Fprintf(t.stdout, "> %s() %s:%d (PC: %#v)\n", loc.Function.Name(), t.formatPath(loc.File), loc.Line, loc.PC)
	if loc.Function != nil && loc.Function.Optimized {
		fmt.Fprintln(t.stdout, optimizedFunctionWarning)
	}
}

func printReturnValues(t *Session, th *api.Thread) {
	if th.ReturnValues == nil {
		return
	}
	fmt.Fprintln(t.stdout, "Values returned:")
	for _, v := range th.ReturnValues {
		fmt.Fprintf(t.stdout, "\t%s: %s\n", v.Name, v.MultilineString("\t", ""))
	}
	fmt.Fprintln(t.stdout)
}

func printcontextThread(t *Session, th *api.Thread) {
	fn := th.Function

	if th.Breakpoint == nil {
		printcontextLocation(t, api.Location{PC: th.PC, File: th.File, Line: th.Line, Function: th.Function})
		printReturnValues(t, th)
		return
	}

	args := ""
	var hasReturnValue bool
	if th.BreakpointInfo != nil && th.Breakpoint.LoadArgs != nil && *th.Breakpoint.LoadArgs == ShortLoadConfig {
		var arg []string
		for _, ar := range th.BreakpointInfo.Arguments {
			// For AI compatibility return values are included in the
			// argument list. This is a relic of the dark ages when the
			// Go debug information did not distinguish between the two.
			// Filter them out here instead, so during trace operations
			// they are not printed as an argument.
			if (ar.Flags & api.VariableArgument) != 0 {
				arg = append(arg, ar.SinglelineString())
			}
			if (ar.Flags & api.VariableReturnArgument) != 0 {
				hasReturnValue = true
			}
		}
		args = strings.Join(arg, ", ")
	}

	bpname := ""
	if th.Breakpoint.WatchExpr != "" {
		bpname = fmt.Sprintf("watchpoint on [%s] ", th.Breakpoint.WatchExpr)
	} else if th.Breakpoint.Name != "" {
		bpname = fmt.Sprintf("[%s] ", th.Breakpoint.Name)
	} else if !th.Breakpoint.Tracepoint {
		bpname = fmt.Sprintf("[Breakpoint %d] ", th.Breakpoint.ID)
	}

	if th.Breakpoint.Tracepoint || th.Breakpoint.TraceReturn {
		printTracepoint(t, th, bpname, fn, args, hasReturnValue)
		return
	}

	if hitCount, ok := th.Breakpoint.HitCount[strconv.FormatInt(th.GoroutineID, 10)]; ok {
		fmt.Fprintf(t.stdout, "> %s%s(%s) %s:%d (hits goroutine(%d):%d total:%d) (PC: %#v)\n",
			bpname,
			fn.Name(),
			args,
			t.formatPath(th.File),
			th.Line,
			th.GoroutineID,
			hitCount,
			th.Breakpoint.TotalHitCount,
			th.PC)
	} else {
		fmt.Fprintf(t.stdout, "> %s%s(%s) %s:%d (hits total:%d) (PC: %#v)\n",
			bpname,
			fn.Name(),
			args,
			t.formatPath(th.File),
			th.Line,
			th.Breakpoint.TotalHitCount,
			th.PC)
	}
	if th.Function != nil && th.Function.Optimized {
		fmt.Fprintln(t.stdout, optimizedFunctionWarning)
	}

	printReturnValues(t, th)
	printBreakpointInfo(t, th, false)
}

func printBreakpointInfo(t *Session, th *api.Thread, tracepointOnNewline bool) {
	if th.BreakpointInfo == nil {
		return
	}
	bp := th.Breakpoint
	bpi := th.BreakpointInfo

	if bp.TraceReturn {
		return
	}

	didprintnl := tracepointOnNewline
	tracepointnl := func() {
		if !bp.Tracepoint || didprintnl {
			return
		}
		didprintnl = true
		fmt.Fprintln(t.stdout)
	}

	if bpi.Goroutine != nil {
		tracepointnl()
		writeGoroutineLong(t, t.stdout, bpi.Goroutine, "\t")
	}

	for _, v := range bpi.Variables {
		tracepointnl()
		fmt.Fprintf(t.stdout, "\t%s: %s\n", v.Name, v.MultilineString("\t", ""))
	}

	for _, v := range bpi.Locals {
		tracepointnl()
		if *bp.LoadLocals == longLoadConfig {
			fmt.Fprintf(t.stdout, "\t%s: %s\n", v.Name, v.MultilineString("\t", ""))
		} else {
			fmt.Fprintf(t.stdout, "\t%s: %s\n", v.Name, v.SinglelineString())
		}
	}

	if bp.LoadArgs != nil && *bp.LoadArgs == longLoadConfig {
		for _, v := range bpi.Arguments {
			tracepointnl()
			fmt.Fprintf(t.stdout, "\t%s: %s\n", v.Name, v.MultilineString("\t", ""))
		}
	}
	if bpi.Stacktrace != nil {
		// TraceFollowCalls and Stacktrace are mutually exclusive as they pollute each others outputs
		if th.Breakpoint.TraceFollowCalls <= 0 {
			tracepointnl()
			fmt.Fprintf(t.stdout, "\tStack:\n")
			printStack(t, t.stdout, bpi.Stacktrace, "\t\t", false)
		}
	}
}

func printTracepoint(t *Session, th *api.Thread, bpname string, fn *api.Function, args string, hasReturnValue bool) {
	if t.conf.TraceShowTimestamp {
		fmt.Fprintf(t.stdout, "%s ", time.Now().Format(time.RFC3339Nano))
	}

	var sdepth, rootindex int
	depthPrefix := ""
	tracePrefix := ""
	if th.Breakpoint.TraceFollowCalls > 0 {
		// Trace Follow Calls; stack is required to calculate depth of functions
		rootindex = -1
		if th.BreakpointInfo == nil || th.BreakpointInfo.Stacktrace == nil {
			return
		}

		stack := th.BreakpointInfo.Stacktrace
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].Function.Name() == th.Breakpoint.RootFuncName {
				if rootindex == -1 {
					rootindex = i
					break
				}
			}
		}
		sdepth = rootindex + 1
		tracePrefix = fmt.Sprintf("goroutine(%d):frame(%d)", th.GoroutineID, sdepth)
		if sdepth > 0 {
			depthPrefix = strings.Repeat(" ", sdepth-1)
		}
	} else {
		tracePrefix = fmt.Sprintf("goroutine(%d):", th.GoroutineID)
	}

	if th.Breakpoint.Tracepoint {
		// Print trace only if there was a match on the function while TraceFollowCalls is on or if it's a regular trace
		if rootindex != -1 || th.Breakpoint.TraceFollowCalls <= 0 {
			fmt.Fprintf(t.stdout, "%s> %s %s%s(%s)\n", depthPrefix, tracePrefix, bpname, fn.Name(), args)
		}
		printBreakpointInfo(t, th, !hasReturnValue)
	}
	if th.Breakpoint.TraceReturn {
		retVals := make([]string, 0, len(th.ReturnValues))
		for _, v := range th.ReturnValues {
			retVals = append(retVals, v.SinglelineString())
		}
		// Print trace only if there was a match on the function while TraceFollowCalls is on or if it's a regular trace
		if rootindex != -1 || th.Breakpoint.TraceFollowCalls <= 0 {
			fmt.Fprintf(t.stdout, "%s>> %s %s => (%s)\n", depthPrefix, tracePrefix, fn.Name(), strings.Join(retVals, ","))
		}
	}
	if th.Breakpoint.TraceFollowCalls > 0 {
		// As of now traceFollowCalls and Stacktrace are mutually exclusive options
		return
	}
	if th.Breakpoint.TraceReturn || !hasReturnValue {
		if th.BreakpointInfo != nil && th.BreakpointInfo.Stacktrace != nil {
			fmt.Fprintf(t.stdout, "\tStack:\n")
			printStack(t, t.stdout, th.BreakpointInfo.Stacktrace, "\t\t", false)
		}
	}
}

type printPosFlags uint8

const (
	printPosShowArrow printPosFlags = 1 << iota
	printPosStepInstruction
)

func printPos(t *Session, th *api.Thread, flags printPosFlags) error {
	if flags&printPosStepInstruction != 0 {
		if t.conf.Position == config.PositionSource {
			return printfile(t, th.File, th.Line, flags&printPosShowArrow != 0)
		}
		return printdisass(t, th.PC)
	}
	if t.conf.Position == config.PositionDisassembly {
		return printdisass(t, th.PC)
	}
	return printfile(t, th.File, th.Line, flags&printPosShowArrow != 0)
}

func printfile(t *Session, filename string, line int, showArrow bool) error {
	if filename == "" {
		return nil
	}

	lineCount := t.conf.GetSourceListLineCount()
	arrowLine := 0
	if showArrow {
		arrowLine = line
	}

	var file *os.File
	path := t.substitutePath(filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		foundPath, err := debuginfod.GetSource(t.client.BuildID(), filename)
		if err == nil {
			path = foundPath
		}
	}
	file, err := os.OpenFile(path, 0, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	return Print(t.stdout, file, line-lineCount, line+lineCount+1, arrowLine)
}

func printdisass(t *Session, pc uint64) error {
	disasm, err := t.client.DisassemblePC(api.EvalScope{GoroutineID: -1, Frame: 0, DeferredCall: 0}, pc, t.conf.GetDisassembleFlavour())
	if err != nil {
		return err
	}

	lineCount := t.conf.GetSourceListLineCount()

	showHeader := true
	for i := range disasm {
		if disasm[i].AtPC {
			s := i - lineCount
			if s < 0 {
				s = 0
			}
			e := i + lineCount + 1
			if e > len(disasm) {
				e = len(disasm)
			}
			showHeader = s == 0
			disasm = disasm[s:e]
			break
		}
	}

	disasmPrint(disasm, t.stdout, showHeader)
	return nil
}

// ExitRequestError is returned when the user
// exits Delve.
type ExitRequestError struct{}

func (ere ExitRequestError) Error() string {
	return ""
}

func exitCommand(t *Session, ctx callContext, args string) error {
	if args == "-c" {
		if !t.client.IsMulticlient() {
			return errors.New("not connected to an --accept-multiclient server")
		}
		bps, _ := t.client.ListBreakpoints(false)
		hasUserBreakpoints := false
		for _, bp := range bps {
			if bp.ID >= 0 {
				hasUserBreakpoints = true
				break
			}
		}
		if hasUserBreakpoints {
			yes, _ := yesno(t.line, "There are breakpoints set, do you wish to quit and continue without clearing breakpoints? [Y/n] ", "yes")
			if !yes {
				return nil
			}
		}
		t.quitContinue = true
	}
	return ExitRequestError{}
}

func getBreakpointByIDOrName(t *Session, arg string) (*api.Breakpoint, error) {
	if id, err := strconv.Atoi(arg); err == nil {
		return t.client.GetBreakpoint(id)
	}
	return t.client.GetBreakpointByName(arg)
}

func (c *DebugCommands) onCmd(t *Session, ctx callContext, argstr string) error {
	args := config.Split2PartsBySpace(argstr)

	if len(args) < 2 {
		return errors.New("not enough arguments")
	}

	bp, err := getBreakpointByIDOrName(t, args[0])
	if err != nil {
		return err
	}

	ctx.Prefix = onPrefix
	ctx.Breakpoint = bp

	if args[1] == "-edit" {
		f, err := os.CreateTemp("", "dlv-on-cmd-")
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}()
		attrs := formatBreakpointAttrs("", ctx.Breakpoint, true)
		_, err = f.WriteString(strings.Join(attrs, "\n"))
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}

		err = runEditor(f.Name())
		if err != nil {
			return err
		}

		fin, err := os.Open(f.Name())
		if err != nil {
			return err
		}
		defer fin.Close()

		err = c.parseBreakpointAttrs(t, ctx, fin)
		if err != nil {
			return err
		}
	} else {
		err = c.CallWithContext(args[1], t, ctx)
		if err != nil {
			return err
		}
	}
	return t.client.AmendBreakpoint(ctx.Breakpoint)
}

func (s *DebugCommands) parseBreakpointAttrs(t *Session, ctx callContext, r io.Reader) error {
	ctx.Breakpoint.Tracepoint = false
	ctx.Breakpoint.Goroutine = false
	ctx.Breakpoint.Stacktrace = 0
	ctx.Breakpoint.Variables = ctx.Breakpoint.Variables[:0]
	ctx.Breakpoint.Cond = ""
	ctx.Breakpoint.HitCond = ""

	scan := bufio.NewScanner(r)
	lineno := 0
	for scan.Scan() {
		lineno++
		err := s.CallWithContext(scan.Text(), t, ctx)
		if err != nil {
			fmt.Fprintf(t.stdout, "%d: %s\n", lineno, err.Error())
		}
	}
	return scan.Err()
}

func condition(t *Session, ctx callContext, argstr string) error {
	args := config.Split2PartsBySpace(argstr)

	if len(args) < 2 {
		return errors.New("not enough arguments")
	}

	hitCondPerG := args[0] == "-per-g-hitcount"
	if args[0] == "-hitcount" || hitCondPerG {
		// hitcount breakpoint

		if ctx.Prefix == onPrefix {
			ctx.Breakpoint.HitCond = args[1]
			ctx.Breakpoint.HitCondPerG = hitCondPerG
			return nil
		}

		args = config.Split2PartsBySpace(args[1])
		if len(args) < 2 {
			return errors.New("not enough arguments")
		}

		bp, err := getBreakpointByIDOrName(t, args[0])
		if err != nil {
			return err
		}

		bp.HitCond = args[1]
		bp.HitCondPerG = hitCondPerG

		return t.client.AmendBreakpoint(bp)
	}

	if args[0] == "-clear" {
		bp, err := getBreakpointByIDOrName(t, args[1])
		if err != nil {
			return err
		}
		bp.Cond = ""
		return t.client.AmendBreakpoint(bp)
	}

	if ctx.Prefix == onPrefix {
		ctx.Breakpoint.Cond = argstr
		return nil
	}

	bp, err := getBreakpointByIDOrName(t, args[0])
	if err != nil {
		return err
	}
	bp.Cond = args[1]

	return t.client.AmendBreakpoint(bp)
}

func (s *DebugCommands) executeFile(t *Session, name string) error {
	fh, err := os.Open(name)
	if err != nil {
		return err
	}
	defer fh.Close()

	scanner := bufio.NewScanner(fh)
	lineno := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineno++

		if line == "" || line[0] == '#' {
			continue
		}

		if err := s.Call(line, t); err != nil {
			if _, isExitRequest := err.(ExitRequestError); isExitRequest {
				return err
			}
			fmt.Fprintf(t.stdout, "%s:%d: %v\n", name, lineno, err)
		}
	}

	return scanner.Err()
}

func display(t *Session, ctx callContext, args string) error {
	const (
		addOption = "-a "
		delOption = "-d "
	)
	switch {
	case args == "":
		t.printDisplays()

	case strings.HasPrefix(args, addOption):
		args = strings.TrimSpace(args[len(addOption):])
		fmtstr, args := parseFormatArg(args)
		if args == "" {
			return errors.New("not enough arguments")
		}
		t.addDisplay(args, fmtstr)
		t.printDisplay(len(t.displays) - 1)

	case strings.HasPrefix(args, delOption):
		args = strings.TrimSpace(args[len(delOption):])
		n, err := strconv.Atoi(args)
		if err != nil {
			return fmt.Errorf("%q is not a number", args)
		}
		return t.removeDisplay(n)

	default:
		return errors.New("wrong arguments")
	}
	return nil
}

func dump(t *Session, ctx callContext, args string) error {
	if args == "" {
		return errors.New("not enough arguments")
	}
	dumpState, err := t.client.CoreDumpStart(args)
	if err != nil {
		return err
	}
	for {
		if dumpState.ThreadsDone != dumpState.ThreadsTotal {
			fmt.Fprintf(t.stdout, "\rDumping threads %d / %d...", dumpState.ThreadsDone, dumpState.ThreadsTotal)
		} else {
			fmt.Fprintf(t.stdout, "\rDumping memory %d / %d...", dumpState.MemDone, dumpState.MemTotal)
		}
		if !dumpState.Dumping {
			break
		}
		dumpState = t.client.CoreDumpWait(1000)
	}
	fmt.Fprintf(t.stdout, "\n")
	if dumpState.Err != "" {
		fmt.Fprintf(t.stdout, "error dumping: %s\n", dumpState.Err)
	} else if !dumpState.AllDone {
		fmt.Fprintf(t.stdout, "canceled\n")
	} else if dumpState.MemDone != dumpState.MemTotal {
		fmt.Fprintf(t.stdout, "Core dump could be incomplete\n")
	}
	return nil
}

func target(t *Session, ctx callContext, args string) error {
	argv := config.Split2PartsBySpace(args)
	switch argv[0] {
	case "list":
		tgts, err := t.client.ListTargets()
		if err != nil {
			return err
		}
		w := new(tabwriter.Writer)
		w.Init(t.stdout, 4, 4, 2, ' ', 0)
		for _, tgt := range tgts {
			selected := ""
			if tgt.Pid == t.oldPid {
				selected = "*"
			}
			fmt.Fprintf(w, "%s\t%d\t%s\n", selected, tgt.Pid, tgt.CmdLine)
		}
		w.Flush()
		return nil
	case "follow-exec":
		if len(argv) == 1 {
			if t.client.FollowExecEnabled() {
				fmt.Fprintf(t.stdout, "Follow exec is enabled.\n")
			} else {
				fmt.Fprintf(t.stdout, "Follow exec is disabled.\n")
			}
			return nil
		}
		argv = config.Split2PartsBySpace(argv[1])
		switch argv[0] {
		case "-on":
			var regex string
			if len(argv) == 2 {
				regex = argv[1]
			}
			t.client.FollowExec(true, regex)
		case "-off":
			if len(argv) > 1 {
				return errors.New("too many arguments")
			}
			t.client.FollowExec(false, "")
		default:
			return fmt.Errorf("unknown argument %q to 'target follow-exec'", argv[0])
		}
		return nil
	case "switch":
		tgts, err := t.client.ListTargets()
		if err != nil {
			return err
		}
		pid, err := strconv.Atoi(argv[1])
		if err != nil {
			return err
		}
		found := false
		for _, tgt := range tgts {
			if tgt.Pid == pid {
				found = true
				t.client.SwitchThread(tgt.CurrentThread.ID)
			}
		}
		if !found {
			return fmt.Errorf("could not find target %d", pid)
		}
		return nil
	case "":
		return errors.New("not enough arguments for 'target'")
	default:
		return fmt.Errorf("unknown command 'target %s'", argv[0])
	}
}

func formatBreakpointName(bp *api.Breakpoint, upcase bool) string {
	thing := "breakpoint"
	if bp.Tracepoint {
		thing = "tracepoint"
	}
	if bp.WatchExpr != "" {
		thing = "watchpoint"
	}
	if upcase {
		thing = strings.ToUpper(string(thing[0])) + thing[1:]
	}
	id := bp.Name
	if id == "" {
		id = strconv.Itoa(bp.ID)
	}
	if bp.WatchExpr != "" && bp.WatchExpr != bp.Name {
		return fmt.Sprintf("%s %s on [%s]", thing, id, bp.WatchExpr)
	}
	return fmt.Sprintf("%s %s", thing, id)
}

func (t *Session) formatBreakpointLocation(bp *api.Breakpoint) string {
	var out bytes.Buffer
	if len(bp.Addrs) > 0 {
		for i, addr := range bp.Addrs {
			if i == 0 {
				fmt.Fprintf(&out, "%#x", addr)
			} else {
				fmt.Fprintf(&out, ",%#x", addr)
			}
		}
	} else {
		// In case we are connecting to an older version of delve that does not return the Addrs field.
		fmt.Fprintf(&out, "%#x", bp.Addr)
	}
	if bp.WatchExpr == "" {
		fmt.Fprintf(&out, " for ")
		p := t.formatPath(bp.File)
		if bp.FunctionName != "" {
			fmt.Fprintf(&out, "%s() ", bp.FunctionName)
		}
		fmt.Fprintf(&out, "%s:%d", p, bp.Line)
	}
	return out.String()
}

func shouldAskToSuspendBreakpoint(t *Session) bool {
	fns, _ := t.client.ListFunctions(`^plugin\.Open$`, 0)
	_, err := t.client.GetState()
	return len(fns) > 0 || isErrProcessExited(err) || t.client.FollowExecEnabled()
}

func disasmPrint(dv api.AsmInstructions, out io.Writer, showHeader bool) {
	bw := bufio.NewWriter(out)
	defer bw.Flush()
	if len(dv) > 0 && dv[0].Loc.Function != nil && showHeader {
		fmt.Fprintf(bw, "TEXT %s(SB) %s\n", dv[0].Loc.Function.Name(), dv[0].Loc.File)
	}
	tw := tabwriter.NewWriter(bw, 1, 8, 1, '\t', 0)
	defer tw.Flush()
	for _, inst := range dv {
		atbp := ""
		if inst.Breakpoint {
			atbp = "*"
		}
		atpc := ""
		if inst.AtPC {
			atpc = "=>"
		}
		fmt.Fprintf(tw, "%s\t%s:%d\t%#x%s\t%x\t%s\n", atpc, filepath.Base(inst.Loc.File), inst.Loc.Line, inst.Loc.PC, atbp, inst.Bytes, inst.Text)
	}
}

// Print prints to out the text read from reader, between lines startLine and endLine.
func Print(out io.Writer, reader io.Reader, startLine, endLine, arrowLine int) error {
	scanner := bufio.NewScanner(reader)
	lineno := 0

	for scanner.Scan() {
		lineno++
		if lineno < startLine {
			continue
		}
		if lineno >= endLine {
			break
		}

		// Print line number and arrow
		if lineno == arrowLine {
			fmt.Fprintf(out, "=>")
		} else {
			fmt.Fprintf(out, "  ")
		}
		fmt.Fprintf(out, "%4d:\t%s\n", lineno, scanner.Text())
	}

	return scanner.Err()
}

// matching groups:
// - group0 is the whole match
// - group1 is wanted `spec“
// - group2 is if
// - group3 is wanted `cond“
var breakpointRE = regexp.MustCompile(`(?m)^(\S*)?\s*(if\s(.*)\s*)?$`)

func parseBreakpointArgs(argstr string) (name, spec, cond string, err error) {
	fs := pflag.NewFlagSet("name", pflag.ContinueOnError)
	fs.StringP("name", "n", "", "breakpoint name")

	if err := fs.Parse(strings.Fields(argstr)); err != nil {
		return "", "", "", err
	}

	// flag `--name|-n=?`
	name, err = fs.GetString("name")
	if err != nil {
		return "", "", "", fmt.Errorf("parse `name` error: %v", err)
	}

	// parse spec, cond
	left := strings.Join(fs.Args(), " ")
	submatches := breakpointRE.FindStringSubmatch(left)
	if len(submatches) != 4 {
		return "", "", "", errors.New("invalid [locspec] or [if condition]")
	}
	spec = submatches[1]
	cond = submatches[3]

	return name, spec, cond, nil
}
