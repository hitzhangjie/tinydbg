// Package terminal implements functions for responding to user
// input and dispatching to appropriate backend commands.
package terminal

//lint:file-ignore ST1005 errors here can be capitalized

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/locspec"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/pkg/terminal/colorize"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
)

const optimizedFunctionWarning = "Warning: debugging optimized function"

// TODO what does cmdPrefix do?
type cmdPrefix int

const (
	noPrefix = cmdPrefix(0)
	onPrefix = cmdPrefix(1 << iota)
	deferredPrefix
)

type callContext struct {
	Prefix     cmdPrefix
	Scope      api.EvalScope
	Breakpoint *api.Breakpoint
}

func (ctx *callContext) scoped() bool {
	return ctx.Scope.GoroutineID >= 0 || ctx.Scope.Frame > 0
}

type frameDirection int

const (
	frameSet frameDirection = iota
	frameUp
	frameDown
)

type cmdfunc func(t *Term, ctx callContext, args string) error

// command debugging command
type command struct {
	aliases         []string     // command aliases, like `h` for `help`
	builtinAliases  []string     // TODO what does this do?
	group           commandGroup // TODO cobra supports this! why do this mannually?
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

// Commands represents the commands for Delve terminal process.
type Commands struct {
	cmds   []command
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

// byFirstAlias will sort by the first
// alias of a command.
type byFirstAlias []command

func (a byFirstAlias) Len() int           { return len(a) }
func (a byFirstAlias) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFirstAlias) Less(i, j int) bool { return a[i].aliases[0] < a[j].aliases[0] }

// DebugCommands returns a Commands struct with default commands defined.
func DebugCommands(client service.Client) *Commands {
	c := &Commands{client: client}

	// clientside debugging commands
	//
	// readability: this code is really really dirty!
	c.cmds = []command{
		{aliases: []string{"help", "h"}, cmdFn: c.help, helpMsg: helpCmdHelpMsg},
		{aliases: []string{"break", "b"}, group: breakCmds, cmdFn: breakpoint, helpMsg: breakCmdHelpMsg},
		{aliases: []string{"trace", "t"}, group: breakCmds, cmdFn: tracepoint, allowedPrefixes: onPrefix, helpMsg: traceCmdHelpMsg},
		{aliases: []string{"restart", "r"}, group: runCmds, cmdFn: restart, helpMsg: restartCmdHelpMsg},
		{aliases: []string{"rebuild"}, group: runCmds, cmdFn: c.rebuild, helpMsg: rebuildCmdHelpMsg},
		{aliases: []string{"continue", "c"}, group: runCmds, cmdFn: c.cont, helpMsg: continueCmdHelpMsg},
		{aliases: []string{"step", "s"}, group: runCmds, cmdFn: c.step, helpMsg: stepCmdHelpMsg},
		{aliases: []string{"step-instruction", "si"}, group: runCmds, cmdFn: c.stepInstruction, helpMsg: stepInstCmdHelpMsg},
		{aliases: []string{"next", "n"}, group: runCmds, cmdFn: c.next, helpMsg: nextCmdHelpMsg},
		{aliases: []string{"stepout", "so"}, group: runCmds, cmdFn: c.stepout, helpMsg: stepOutCmdHelpMsg},
		{aliases: []string{"call"}, group: runCmds, cmdFn: c.call, helpMsg: callCmdHelpMsg},
		{aliases: []string{"threads"}, group: goroutineCmds, cmdFn: threads, helpMsg: threadsCmdHelpMsg},
		{aliases: []string{"thread", "tr"}, group: goroutineCmds, cmdFn: thread, helpMsg: threadCmdHelpMsg},
		{aliases: []string{"toggle"}, group: breakCmds, cmdFn: toggle, helpMsg: toggleCmdHelpMsg},
		{aliases: []string{"goroutines", "grs"}, group: goroutineCmds, cmdFn: goroutines, helpMsg: goroutinesCmdHelpMsg},
		{aliases: []string{"goroutine", "gr"}, group: goroutineCmds, allowedPrefixes: onPrefix, cmdFn: c.goroutine, helpMsg: goroutineCmdHelpMsg},
		{aliases: []string{"breakpoints", "bp"}, group: breakCmds, cmdFn: breakpoints, helpMsg: breakpointsCmdHelpMsg},
		{aliases: []string{"print", "p"}, group: dataCmds, allowedPrefixes: onPrefix | deferredPrefix, cmdFn: printVar, helpMsg: printCmdHelpMsg},
		{aliases: []string{"whatis"}, group: dataCmds, cmdFn: whatisCommand, helpMsg: whatisCmdHelpMsg},
		{aliases: []string{"set"}, group: dataCmds, cmdFn: setVar, helpMsg: setCmdHelpMsg},
		{aliases: []string{"sources"}, cmdFn: sources, helpMsg: sourcesCmdHelpMsg},
		{aliases: []string{"funcs"}, cmdFn: funcs, helpMsg: funcsCmdHelpMsg},
		{aliases: []string{"types"}, cmdFn: types, helpMsg: typesCmdHelpMsg},
		{aliases: []string{"args"}, allowedPrefixes: onPrefix | deferredPrefix, group: dataCmds, cmdFn: args, helpMsg: argsCmdHelpMsg},
		{aliases: []string{"locals"}, allowedPrefixes: onPrefix | deferredPrefix, group: dataCmds, cmdFn: locals, helpMsg: localsCmdHelpMsg},
		{aliases: []string{"vars"}, cmdFn: vars, group: dataCmds, helpMsg: varsCmdHelpMsg},
		{aliases: []string{"regs"}, cmdFn: regs, group: dataCmds, helpMsg: regsCmdHelpMsg},
		{aliases: []string{"exit", "quit", "q"}, cmdFn: exitCommand, helpMsg: exitCmdHelpMsg},
		{aliases: []string{"list", "ls", "l"}, cmdFn: listCommand, helpMsg: listCmdHelpMsg},
		{aliases: []string{"stack", "bt"}, allowedPrefixes: onPrefix, group: stackCmds, cmdFn: stackCommand, helpMsg: stackCmdHelpMsg},
		{aliases: []string{"frame"},
			group: stackCmds,
			cmdFn: func(t *Term, ctx callContext, arg string) error {
				return c.frameCommand(t, ctx, arg, frameSet)
			},
			helpMsg: frameCmdHelpMsg,
		},
		{aliases: []string{"up"},
			group: stackCmds,
			cmdFn: func(t *Term, ctx callContext, arg string) error {
				return c.frameCommand(t, ctx, arg, frameUp)
			},
			helpMsg: upCmdHelpMsg,
		},
		{aliases: []string{"down"},
			group: stackCmds,
			cmdFn: func(t *Term, ctx callContext, arg string) error {
				return c.frameCommand(t, ctx, arg, frameDown)
			},
			helpMsg: downCmdHelpMsg,
		},
		{aliases: []string{"deferred"}, group: stackCmds, cmdFn: c.deferredCommand, helpMsg: deferredCmdHelpMsg},
		{aliases: []string{"source"}, cmdFn: c.sourceCommand, helpMsg: sourceCmdHelpMsg},
		{aliases: []string{"disassemble", "disass"}, cmdFn: disassCommand, helpMsg: disassCmdHelpMsg},
		{aliases: []string{"on"}, group: breakCmds, cmdFn: c.onCmd, helpMsg: onCmdHelpMsg},
		{aliases: []string{"condition", "cond"}, group: breakCmds, cmdFn: conditionCmd, allowedPrefixes: onPrefix, helpMsg: conditionCmdHelpMsg},
		{aliases: []string{"config"}, cmdFn: configureCmd, helpMsg: configCmdHelpMsg},
		{aliases: []string{"edit", "ed"}, cmdFn: edit, helpMsg: editCmdHelpMsg},
		{aliases: []string{"libraries"}, cmdFn: libraries, helpMsg: librariesCmdHelpMsg},
		{aliases: []string{"examinemem", "x"}, group: dataCmds, cmdFn: examineMemoryCmd, helpMsg: examinememCmdHelpMsg},
		{aliases: []string{"display"}, group: dataCmds, cmdFn: display, helpMsg: disassCmdHelpMsg},
		{aliases: []string{"dump"}, cmdFn: dump, helpMsg: dumpCmdHelpMsg},
	}

	sort.Sort(byFirstAlias(c.cmds))
	return c
}

// Find will look up the command function for the given command input.
// If it cannot find the command it will default to noCmdAvailable().
// If the command is an empty string it will replay the last command.
func (c *Commands) Find(cmdstr string, prefix cmdPrefix) command {
	// If <enter> use last command, if there was one.
	if cmdstr == "" {
		return command{aliases: []string{"nullcmd"}, cmdFn: nullCommand}
	}

	for _, v := range c.cmds {
		if v.match(cmdstr) {
			if prefix != noPrefix && v.allowedPrefixes&prefix == 0 {
				continue
			}
			return v
		}
	}

	return command{aliases: []string{"nocmd"}, cmdFn: noCmdAvailable}
}

// CallWithContext takes a command and a context that command should be executed in.
func (c *Commands) CallWithContext(cmdstr string, t *Term, ctx callContext) error {
	vals := strings.SplitN(strings.TrimSpace(cmdstr), " ", 2)
	cmdname := vals[0]
	var args string
	if len(vals) > 1 {
		args = strings.TrimSpace(vals[1])
	}
	return c.Find(cmdname, ctx.Prefix).cmdFn(t, ctx, args)
}

// Call takes a command to execute.
func (c *Commands) Call(cmdstr string, t *Term) error {
	ctx := callContext{Prefix: noPrefix, Scope: api.EvalScope{GoroutineID: -1, Frame: c.frame, DeferredCall: 0}}
	return c.CallWithContext(cmdstr, t, ctx)
}

// Merge takes aliases defined in the config struct and merges them with the default aliases.
func (c *Commands) Merge(allAliases map[string][]string) {
	for i := range c.cmds {
		if c.cmds[i].builtinAliases != nil {
			c.cmds[i].aliases = append(c.cmds[i].aliases[:0], c.cmds[i].builtinAliases...)
		}
	}
	for i := range c.cmds {
		if aliases, ok := allAliases[c.cmds[i].aliases[0]]; ok {
			if c.cmds[i].builtinAliases == nil {
				c.cmds[i].builtinAliases = make([]string, len(c.cmds[i].aliases))
				copy(c.cmds[i].builtinAliases, c.cmds[i].aliases)
			}
			c.cmds[i].aliases = append(c.cmds[i].aliases, aliases...)
		}
	}
}

var errNoCmd = errors.New("command not available")

func noCmdAvailable(t *Term, ctx callContext, args string) error {
	return errNoCmd
}

func nullCommand(t *Term, ctx callContext, args string) error {
	return nil
}

// help print specific subcmd's help message, or prettyprint all subcommands'
// help message group by commandGroup.
func (c *Commands) help(t *Term, ctx callContext, args string) error {
	// print specific subcmd's help message
	if args != "" {
		for _, cmd := range c.cmds {
			for _, alias := range cmd.aliases {
				if alias == args {
					log.Info(cmd.helpMsg)
					return nil
				}
			}
		}
		return errNoCmd
	}

	// prettyprint all subcommands' help message group by commandGroup
	log.Info("The following commands are available:")

	for _, cgd := range commandGroupDescriptions {
		log.Info("\n%s:", cgd.description)
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 0, '-', 0)
		for _, cmd := range c.cmds {
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

	log.Info("")
	log.Info("Type help followed by a command for full documentation.")
	return nil
}

type byThreadID []*api.Thread

func (a byThreadID) Len() int { return len(a) }
func (a byThreadID) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a byThreadID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func threads(t *Term, ctx callContext, args string) error {
	threads, err := t.client.ListThreads()
	if err != nil {
		return err
	}
	state, err := t.client.GetState()
	if err != nil {
		return err
	}
	sort.Sort(byThreadID(threads))
	for _, th := range threads {
		prefix := "  "
		if state.CurrentThread != nil && state.CurrentThread.ID == th.ID {
			prefix = "* "
		}
		if th.Function != nil {
			log.Info("%sThread %d at %#v %s:%d %s",
				prefix, th.ID, th.PC, t.formatPath(th.File),
				th.Line, th.Function.Name())
		} else {
			log.Info("%sThread %s", prefix, t.formatThread(th))
		}
	}
	return nil
}

func thread(t *Term, ctx callContext, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("you must specify a thread")
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
	log.Info("Switched from %s to %s", oldThread, newThread)
	return nil
}

type byGoroutineID []*api.Goroutine

func (a byGoroutineID) Len() int { return len(a) }
func (a byGoroutineID) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a byGoroutineID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func printGoroutines(t *Term, indent string, gs []*api.Goroutine, fgl api.FormatGoroutineLoc, flags api.PrintGoroutinesFlags, depth int, state *api.DebuggerState) error {
	for _, g := range gs {
		prefix := indent + "  "
		if state.SelectedGoroutine != nil && g.ID == state.SelectedGoroutine.ID {
			prefix = indent + "* "
		}
		log.Info("%sGoroutine %s", prefix, t.formatGoroutine(g, fgl))
		if flags&api.PrintGoroutinesLabels != 0 {
			writeGoroutineLabels(os.Stdout, g, indent+"\t")
		}
		if flags&api.PrintGoroutinesStack != 0 {
			stack, err := t.client.Stacktrace(g.ID, depth, 0, nil)
			if err != nil {
				return err
			}
			printStack(t, os.Stdout, stack, indent+"\t", false)
		}
	}
	return nil
}

func goroutines(t *Term, ctx callContext, argstr string) error {
	filters, group, fgl, flags, depth, batchSize, err := api.ParseGoroutineArgs(argstr)
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
	t.longCommandStart()
	for start >= 0 {
		if t.longCommandCanceled() {
			log.Info("interrupted")
			return nil
		}
		gs, groups, start, tooManyGroups, err = t.client.ListGoroutinesWithFilter(start, batchSize, filters, &group)
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			for i := range groups {
				log.Info("%s", groups[i].Name)
				err = printGoroutines(t, "\t", gs[groups[i].Offset:][:groups[i].Count], fgl, flags, depth, state)
				if err != nil {
					return err
				}
				log.Info("\tTotal: %d", groups[i].Total)
				if i != len(groups)-1 {
					log.Info("")
				}
			}
			if tooManyGroups {
				log.Warn("Too many groups")
			}
		} else {
			sort.Sort(byGoroutineID(gs))
			err = printGoroutines(t, "", gs, fgl, flags, depth, state)
			if err != nil {
				return err
			}
			gslen += len(gs)
		}
	}
	if gslen > 0 {
		log.Info("[%d goroutines]", gslen)
	}
	return nil
}

func selectedGID(state *api.DebuggerState) int {
	if state.SelectedGoroutine == nil {
		return 0
	}
	return state.SelectedGoroutine.ID
}

func (c *Commands) goroutine(t *Term, ctx callContext, argstr string) error {
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
		gid, err := strconv.Atoi(argstr)
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
		c.frame = 0
		log.Info("Switched from %d to %d (thread %d)", selectedGID(oldState), gid, newState.CurrentThread.ID)
		return nil
	}

	var err error
	ctx.Scope.GoroutineID, err = strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	return c.CallWithContext(args[1], t, ctx)
}

// Handle "frame", "up", "down" commands.
func (c *Commands) frameCommand(t *Term, ctx callContext, argstr string, direction frameDirection) error {
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
		frame = c.frame + frame
	case frameDown:
		frame = c.frame - frame
	}
	if len(arg) > 0 {
		ctx.Scope.Frame = frame
		return c.CallWithContext(arg, t, ctx)
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
	c.frame = frame
	state, err := t.client.GetState()
	if err != nil {
		return err
	}
	printcontext(t, state)
	th := stack[frame]
	log.Info("Frame %d: %s:%d (PC: %x)", frame, t.formatPath(th.File), th.Line, th.PC)
	printfile(t, th.File, th.Line, true)
	return nil
}

func (c *Commands) deferredCommand(t *Term, ctx callContext, argstr string) error {
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
	return c.CallWithContext(argstr[space:], t, ctx)
}

func printscope(t *Term) error {
	state, err := t.client.GetState()
	if err != nil {
		return err
	}

	log.Info("Thread %s", t.formatThread(state.CurrentThread))
	if state.SelectedGoroutine != nil {
		writeGoroutineLong(t, os.Stdout, state.SelectedGoroutine, "")
	}
	return nil
}

func (t *Term) formatThread(th *api.Thread) string {
	if th == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d at %s:%d", th.ID, t.formatPath(th.File), th.Line)
}

func (t *Term) formatLocation(loc api.Location) string {
	return fmt.Sprintf("%s:%d %s (%#v)", t.formatPath(loc.File), loc.Line, loc.Function.Name(), loc.PC)
}

func (t *Term) formatGoroutine(g *api.Goroutine, fgl api.FormatGoroutineLoc) string {
	if g == nil {
		return "<nil>"
	}
	if g.Unreadable != "" {
		return fmt.Sprintf("(unreadable %s)", g.Unreadable)
	}
	var locname string
	var loc api.Location
	switch fgl {
	case api.FormatGLocRuntimeCurrent:
		locname = "Runtime"
		loc = g.CurrentLoc
	case api.FormatGLocUserCurrent:
		locname = "User"
		loc = g.UserCurrentLoc
	case api.FormatGLocGo:
		locname = "Go"
		loc = g.GoStatementLoc
	case api.FormatGLocStart:
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
			fmt.Fprintf(buf, " %s", time.Since(time.Unix(0, g.WaitSince)).String())
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
}

func writeGoroutineLong(t *Term, w io.Writer, g *api.Goroutine, prefix string) {
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

func restart(t *Term, ctx callContext, args string) error {
	discarded, err := t.client.Restart(false)
	if err != nil {
		return err
	}
	for i := range discarded {
		log.Info("Discarded %s at %s: %v", formatBreakpointName(discarded[i].Breakpoint, false), t.formatBreakpointLocation(discarded[i].Breakpoint), discarded[i].Reason)
	}

	log.Info("Process restarted with PID: %d", t.client.ProcessPid())
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

func printcontextNoState(t *Term) {
	state, _ := t.client.GetState()
	if state == nil || state.CurrentThread == nil {
		return
	}
	printcontext(t, state)
}

func (c *Commands) rebuild(t *Term, ctx callContext, args string) error {
	defer t.printDisplays()
	discarded, err := t.client.Restart(true)
	if len(discarded) > 0 {
		log.Warn("not all breakpoints could be restored.")
	}
	return err
}

// args != "", continue <locspec>
// args == "", continue
func (c *Commands) cont(t *Term, ctx callContext, args string) error {
	// args != "", add breakpoint at <locspec> first
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
					log.Error("failed to clear temporary breakpoint: %d", bp.ID)
				}
			}
		}()
	}

	// if continue, so run to next breakpoint
	defer t.printDisplays()

	c.frame = 0
	stateChan := t.client.Continue()
	var state *api.DebuggerState
	for state = range stateChan {
		if state.Err != nil {
			printcontextNoState(t)
			return state.Err
		}
		printcontext(t, state)
	}
	printfile(t, state.CurrentThread.File, state.CurrentThread.Line, true)
	return nil
}

func continueUntilCompleteNext(t *Term, state *api.DebuggerState, op string, shouldPrintFile bool) error {
	defer t.printDisplays()
	if !state.NextInProgress {
		if shouldPrintFile {
			printfile(t, state.CurrentThread.File, state.CurrentThread.Line, true)
		}
		return nil
	}
	skipBreakpoints := false
	for {
		log.Info("\tbreakpoint hit during %s", op)
		if !skipBreakpoints {
			log.Info("")
			answer, err := promptAutoContinue(t, op)
			switch answer {
			case "f": // finish next
				skipBreakpoints = true
				fallthrough
			case "c": // continue once
				log.Info("continuing...")
			case "s": // stop and cancel
				fallthrough
			default:
				t.client.CancelNext()
				printfile(t, state.CurrentThread.File, state.CurrentThread.Line, true)
				return err
			}
		} else {
			log.Info(", continuing...\n")
		}
		stateChan := t.client.DirectionCongruentContinue()
		var state *api.DebuggerState
		for state = range stateChan {
			if state.Err != nil {
				printcontextNoState(t)
				return state.Err
			}
			printcontext(t, state)
		}
		if !state.NextInProgress {
			printfile(t, state.CurrentThread.File, state.CurrentThread.Line, true)
			return nil
		}
	}
}

func promptAutoContinue(t *Term, op string) (string, error) {
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

func scopePrefixSwitch(t *Term, ctx callContext) error {
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

func (c *Commands) step(t *Term, ctx callContext, args string) error {
	// tell dbg server to switch to target goroutine
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	// tell dbg server to step target goroutine,
	// this step operation maybe interrupted by some breakpoints.
	c.frame = 0
	stepfn := t.client.Step
	state, err := exitedToError(stepfn())
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	// tell dbg server to continue to finish step operation
	return continueUntilCompleteNext(t, state, "step", true)
}

var errNotOnFrameZero = errors.New("not on topmost frame")

func (c *Commands) stepInstruction(t *Term, ctx callContext, args string) error {
	// tell dbg server to switch to target goroutine
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	if c.frame != 0 {
		return errNotOnFrameZero
	}

	defer t.printDisplays()

	// tell dbg server to step next instruction
	fn := t.client.StepInstruction
	state, err := exitedToError(fn())
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	printfile(t, state.CurrentThread.File, state.CurrentThread.Line, true)
	return nil
}

func (c *Commands) next(t *Term, ctx callContext, args string) error {
	// tell dbg server to switch to target goroutine
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	if c.frame != 0 {
		return errNotOnFrameZero
	}

	// tell dbg server to run to next source
	nextfn := t.client.Next

	// next [count]
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
		// this next operation maybe interrupted by breakpoints,
		// here, we try to finish the next operation by relaunch
		// this operation.
		if err := continueUntilCompleteNext(t, state, "next", finishedNext); err != nil {
			return err
		}
	}
	return nil
}

func (c *Commands) stepout(t *Term, ctx callContext, args string) error {
	if err := scopePrefixSwitch(t, ctx); err != nil {
		return err
	}
	if c.frame != 0 {
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

func (c *Commands) call(t *Term, ctx callContext, args string) error {
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
	c.frame = 0
	if err != nil {
		printcontextNoState(t)
		return err
	}
	printcontext(t, state)
	return continueUntilCompleteNext(t, state, "call", true)
}

func clear(t *Term, ctx callContext, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("not enough arguments")
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
	log.Info("%s cleared at %s", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	return nil
}

func clearAll(t *Term, ctx callContext, args string) error {
	breakPoints, err := t.client.ListBreakpoints(false)
	if err != nil {
		return err
	}

	var locPCs map[uint64]struct{}
	if args != "" {
		locs, err := t.client.FindLocation(api.EvalScope{GoroutineID: -1, Frame: 0}, args, true, t.substitutePathRules())
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
			log.Error("Couldn't delete %s at %s: %s", formatBreakpointName(bp, false), t.formatBreakpointLocation(bp), err)
		}
		log.Info("%s cleared at %s", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	}
	return nil
}

func toggle(t *Term, ctx callContext, args string) error {
	if args == "" {
		return fmt.Errorf("not enough arguments")
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
	log.Info("%s toggled at %s", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	return nil
}

// byID sorts breakpoints by ID.
type byID []*api.Breakpoint

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].ID < a[j].ID }
func breakpoints(t *Term, ctx callContext, args string) error {
	breakPoints, err := t.client.ListBreakpoints(args == "-a")
	if err != nil {
		return err
	}
	sort.Sort(byID(breakPoints))
	for _, bp := range breakPoints {
		enabled := "(enabled)"
		if bp.Disabled {
			enabled = "(disabled)"
		}
		log.Info("%s %s at %v (%d)", formatBreakpointName(bp, true), enabled, t.formatBreakpointLocation(bp), bp.TotalHitCount)

		attrs := formatBreakpointAttrs("\t", bp, false)

		if len(attrs) > 0 {
			log.Info("%s", strings.Join(attrs, "\n"))
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
		attrs = append(attrs, fmt.Sprintf("%scond -hitcount %s", prefix, bp.HitCond))
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

func setBreakpoint(t *Term, ctx callContext, tracepoint bool, argstr string) ([]*api.Breakpoint, error) {
	args := config.Split2PartsBySpace(argstr)

	requestedBp := &api.Breakpoint{}
	spec := ""
	switch len(args) {
	case 1: // break <locspec>
		spec = argstr
	case 2: // break [name] <locspec>
		if api.ValidBreakpointName(args[0]) == nil {
			requestedBp.Name = args[0]
			spec = args[1]
		} else {
			spec = argstr
		}
	default:
		return nil, fmt.Errorf("address required")
	}

	// launch rpc to find candidated locations for this locspec
	requestedBp.Tracepoint = tracepoint
	locs, err := t.client.FindLocation(ctx.Scope, spec, true, t.substitutePathRules())
	if err != nil {
		if requestedBp.Name == "" {
			return nil, err
		}
		requestedBp.Name = ""
		spec = argstr
		var err2 error
		locs, err2 = t.client.FindLocation(ctx.Scope, spec, true, t.substitutePathRules())
		if err2 != nil {
			return nil, err
		}
	}

	// launch rpc to create breakpoints for each candidated location?
	created := []*api.Breakpoint{}
	for _, loc := range locs {
		requestedBp.Addr = loc.PC
		requestedBp.Addrs = loc.PCs
		if tracepoint {
			requestedBp.LoadArgs = &ShortLoadConfig
		}

		bp, err := t.client.CreateBreakpoint(requestedBp)
		if err != nil {
			return nil, err
		}
		created = append(created, bp)

		log.Info("%s set at %s", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	}

	// test whether we should set another breakpoint at return position for function call
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

	if !(tracepoint && shouldSetReturnBreakpoints && locs[0].Function != nil) {
		return created, nil
	}

	// create the breakpoints at return position for function call
	for i := range locs {
		if locs[i].Function == nil {
			continue
		}
		// todo why we launch the same RPC in a for-loop? maybe the result is different each call?
		addrs, err := t.client.FunctionReturnLocations(locs[0].Function.Name())
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
	return created, nil
}

func breakpoint(t *Term, ctx callContext, args string) error {
	_, err := setBreakpoint(t, ctx, false, args)
	return err
}

func tracepoint(t *Term, ctx callContext, args string) error {
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

func runEditor(args ...string) error {
	var editor string
	if editor = os.Getenv("DELVE_EDITOR"); editor == "" {
		if editor = os.Getenv("EDITOR"); editor == "" {
			return fmt.Errorf("Neither DELVE_EDITOR or EDITOR is set")
		}
	}

	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func edit(t *Term, ctx callContext, args string) error {
	file, lineno, _, err := getLocation(t, ctx, args, false)
	if err != nil {
		return err
	}
	return runEditor(fmt.Sprintf("+%d", lineno), file)
}

func watchpoint(t *Term, ctx callContext, args string) error {
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
	log.Info("%s set at %s", formatBreakpointName(bp, true), t.formatBreakpointLocation(bp))
	return nil
}

func examineMemoryCmd(t *Term, ctx callContext, argstr string) error {
	var (
		address uint64
		err     error
		ok      bool
		args    = strings.Split(argstr, " ")
	)

	// Default value
	priFmt := byte('x')
	count := 1
	size := 1
	isExpr := false

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
				return fmt.Errorf("expected argument after -fmt")
			}
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
		case "-count", "-len":
			arg := nextArg()
			if arg == "" {
				return fmt.Errorf("expected argument after -count/-len")
			}
			var err error
			count, err = strconv.Atoi(arg)
			if err != nil || count <= 0 {
				return fmt.Errorf("count/len must be a positive integer")
			}
		case "-size":
			arg := nextArg()
			if arg == "" {
				return fmt.Errorf("expected argument after -size")
			}
			var err error
			size, err = strconv.Atoi(arg)
			if err != nil || size <= 0 || size > 8 {
				return fmt.Errorf("size must be a positive integer (<=8)")
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

	// TODO, maybe configured by user.
	if count*size > 1000 {
		return fmt.Errorf("read memory range (count*size) must be less than or equal to 1000 bytes")
	}

	if len(args) == 0 {
		return fmt.Errorf("no address specified")
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

	memArea, isLittleEndian, err := t.client.ExamineMemory(address, count*size)
	if err != nil {
		return err
	}
	log.Info(api.PrettyExamineMemory(uintptr(address), memArea, isLittleEndian, priFmt, size))
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

func printVar(t *Term, ctx callContext, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("not enough arguments")
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

	log.Info(val.MultilineString("", fmtstr))
	return nil
}

func whatisCommand(t *Term, ctx callContext, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("not enough arguments")
	}
	val, err := t.client.EvalVariable(ctx.Scope, args, ShortLoadConfig)
	if err != nil {
		return err
	}
	if val.Flags&api.VariableCPURegister != 0 {
		log.Info("CPU Register")
		return nil
	}
	if val.Type != "" {
		log.Info(val.Type)
	}
	if val.RealType != val.Type {
		log.Info("Real type: %s", val.RealType)
	}
	if val.Kind == reflect.Interface && len(val.Children) > 0 {
		log.Info("Concrete type: %s", val.Children[0].Type)
	}
	if t.conf.ShowLocationExpr && val.LocationExpr != "" {
		log.Info("location: %s", val.LocationExpr)
	}
	return nil
}

// analyze the lexpr and rexpr by AST analysis, then tell dbg server
// to set the value (`rexpr`) to the variable (`lexpr`).
func setVar(t *Term, ctx callContext, args string) error {
	// HACK: in go '=' is not an operator, we detect the error and try to recover from it by splitting the input string
	_, err := parser.ParseExpr(args)
	if err == nil {
		return fmt.Errorf("syntax error '=' not found")
	}

	el, ok := err.(scanner.ErrorList)
	if !ok || el[0].Msg != "expected '==', found '='" {
		return err
	}

	lexpr := args[:el[0].Pos.Offset]
	rexpr := args[el[0].Pos.Offset+1:]
	return t.client.SetVariable(ctx.Scope, lexpr, rexpr)
}

func printFilteredVariables(varType string, vars []api.Variable, filter string, cfg api.LoadConfig) error {
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
				log.Info("%s = %s", name, v.SinglelineString())
			} else {
				log.Info("%s = %s", name, v.MultilineString("", ""))
			}
		}
	}
	if !match {
		log.Warn("(no %s)", varType)
	}
	return nil
}

func printSortedStrings(v []string, err error) error {
	if err != nil {
		return err
	}
	sort.Strings(v)
	for _, d := range v {
		log.Info(d)
	}
	return nil
}

func sources(t *Term, ctx callContext, args string) error {
	return printSortedStrings(t.client.ListSources(args))
}

func funcs(t *Term, ctx callContext, args string) error {
	return printSortedStrings(t.client.ListFunctions(args))
}

func types(t *Term, ctx callContext, args string) error {
	return printSortedStrings(t.client.ListTypes(args))
}

func parseVarArguments(args string, t *Term) (filter string, cfg api.LoadConfig) {
	if v := config.Split2PartsBySpace(args); len(v) >= 1 && v[0] == "-v" {
		if len(v) == 2 {
			return v[1], t.loadConfig()
		} else {
			return "", t.loadConfig()
		}
	}
	return args, ShortLoadConfig
}

func args(t *Term, ctx callContext, args string) error {
	filter, cfg := parseVarArguments(args, t)
	if ctx.Prefix == onPrefix {
		if filter != "" {
			return fmt.Errorf("filter not supported on breakpoint")
		}
		ctx.Breakpoint.LoadArgs = &cfg
		return nil
	}
	vars, err := t.client.ListFunctionArgs(ctx.Scope, cfg)
	if err != nil {
		return err
	}
	return printFilteredVariables("args", vars, filter, cfg)
}

func locals(t *Term, ctx callContext, args string) error {
	filter, cfg := parseVarArguments(args, t)
	if ctx.Prefix == onPrefix {
		if filter != "" {
			return fmt.Errorf("filter not supported on breakpoint")
		}
		ctx.Breakpoint.LoadLocals = &cfg
		return nil
	}
	locals, err := t.client.ListLocalVariables(ctx.Scope, cfg)
	if err != nil {
		return err
	}
	return printFilteredVariables("locals", locals, filter, cfg)
}

func vars(t *Term, ctx callContext, args string) error {
	filter, cfg := parseVarArguments(args, t)
	vars, err := t.client.ListPackageVariables(filter, cfg)
	if err != nil {
		return err
	}
	return printFilteredVariables("vars", vars, filter, cfg)
}

func regs(t *Term, ctx callContext, args string) error {
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
	log.Info("%s", regs)
	return nil
}

func stackCommand(t *Term, ctx callContext, args string) error {
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
	printStack(t, os.Stdout, stack, "", sa.offsets)
	if sa.ancestors > 0 {
		ancestors, err := t.client.Ancestors(ctx.Scope.GoroutineID, sa.ancestors, sa.ancestorDepth)
		if err != nil {
			return err
		}
		for _, ancestor := range ancestors {
			log.Info("Created by Goroutine %d:", ancestor.ID)
			if ancestor.Unreadable != "" {
				log.Info("\t%s", ancestor.Unreadable)
				continue
			}
			printStack(t, os.Stdout, ancestor.Stack, "\t", false)
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
					return stackArgs{}, fmt.Errorf("expected normal, simple or fromg after -mode")
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
					return stackArgs{}, fmt.Errorf("expected normal, simple or fromg after -mode")
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
					return stackArgs{}, fmt.Errorf("depth must be a number")
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
func getLocation(t *Term, ctx callContext, args string, showContext bool) (file string, lineno int, showarrow bool, err error) {
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
			log.Info("Goroutine %d frame %d at %s:%d (PC: %#x)", gid, ctx.Scope.Frame, loc.File, loc.Line, loc.PC)
		}
		return loc.File, loc.Line, true, nil

	default:
		locs, err := t.client.FindLocation(ctx.Scope, args, false, t.substitutePathRules())
		if err != nil {
			return "", 0, false, err
		}
		if len(locs) > 1 {
			return "", 0, false, locspec.AmbiguousLocationError{Location: args, CandidatesLocation: locs}
		}
		loc := locs[0]
		if showContext {
			log.Info("Showing %s:%d (PC: %#x)", loc.File, loc.Line, loc.PC)
		}
		return loc.File, loc.Line, false, nil
	}
}

func listCommand(t *Term, ctx callContext, args string) error {
	file, lineno, showarrow, err := getLocation(t, ctx, args, true)
	if err != nil {
		return err
	}
	return printfile(t, file, lineno, showarrow)
}

func (c *Commands) sourceCommand(t *Term, ctx callContext, args string) error {
	if len(args) == 0 {
		return fmt.Errorf("wrong number of arguments: source <filename>")
	}

	if filepath.Ext(args) == ".star" {
		_, err := t.starlarkEnv.Execute(args, nil, "main", nil)
		return err
	}

	if args == "-" {
		return t.starlarkEnv.REPL()
	}

	return c.executeFile(t, args)
}

var errDisasmUsage = errors.New("wrong number of arguments: disassemble [-a <start> <end>] [-l <locspec>]")

func disassCommand(t *Term, ctx callContext, args string) error {
	var cmd, rest string

	if args != "" {
		argv := config.Split2PartsBySpace(args)
		if len(argv) != 2 {
			return errDisasmUsage
		}
		cmd = argv[0]
		rest = argv[1]
	}

	flavor := api.IntelFlavour
	if t.conf != nil && t.conf.DisassembleFlavor != nil {
		switch *t.conf.DisassembleFlavor {
		case "go":
			flavor = api.GoFlavour
		case "gnu":
			flavor = api.GNUFlavour
		default:
			flavor = api.IntelFlavour
		}
	}

	var disasm api.AsmInstructions
	var disasmErr error

	switch cmd {
	case "":
		locs, err := t.client.FindLocation(ctx.Scope, "+0", true, t.substitutePathRules())
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
		locs, err := t.client.FindLocation(ctx.Scope, rest, true, t.substitutePathRules())
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

	disasmPrint(disasm, os.Stdout)

	return nil
}

func libraries(t *Term, ctx callContext, args string) error {
	libs, err := t.client.ListDynamicLibraries()
	if err != nil {
		return err
	}
	d := digits(len(libs))
	for i := range libs {
		log.Info("%"+strconv.Itoa(d)+"d. %#x %s", i, libs[i].Address, libs[i].Path)
	}
	return nil
}

func digits(n int) int {
	if n <= 0 {
		return 1
	}
	return int(math.Floor(math.Log10(float64(n)))) + 1
}

func printStack(t *Term, out io.Writer, stack []api.Stackframe, indent string, offsets bool) {
	api.PrintStack(t.formatPath, out, stack, indent, offsets, func(api.Stackframe) bool { return true })
}

func printcontext(t *Term, state *api.DebuggerState) {
	for i := range state.Threads {
		if (state.CurrentThread != nil) && (state.Threads[i].ID == state.CurrentThread.ID) {
			continue
		}
		if state.Threads[i].Breakpoint != nil {
			printcontextThread(t, state.Threads[i])
		}
	}

	if state.CurrentThread == nil {
		log.Info("No current thread available")
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
		log.Info("Stopped at: 0x%x", state.CurrentThread.PC)
		_ = colorize.Print(t.stdout, "", bytes.NewReader([]byte("no source available")), 1, 10, 1, nil)
		return
	}

	printcontextThread(t, th)

	if state.When != "" {
		log.Info(state.When)
	}

	for _, watchpoint := range state.WatchOutOfScope {
		log.Info("%s went out of scope and was cleared", formatBreakpointName(watchpoint, true))
	}
}

func printcontextLocation(t *Term, loc api.Location) {
	log.Info("> %s() %s:%d (PC: %#v)", loc.Function.Name(), t.formatPath(loc.File), loc.Line, loc.PC)
	if loc.Function != nil && loc.Function.Optimized {
		log.Info(optimizedFunctionWarning)
	}
}

func printReturnValues(th *api.Thread) {
	if th.ReturnValues == nil {
		return
	}
	log.Info("Values returned:")
	for _, v := range th.ReturnValues {
		log.Info("\t%s: %s", v.Name, v.MultilineString("\t", ""))
	}
	log.Info("")
}

func printcontextThread(t *Term, th *api.Thread) {
	fn := th.Function

	if th.Breakpoint == nil {
		printcontextLocation(t, api.Location{PC: th.PC, File: th.File, Line: th.Line, Function: th.Function})
		printReturnValues(th)
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
	}

	if th.Breakpoint.Tracepoint || th.Breakpoint.TraceReturn {
		printTracepoint(t, th, bpname, fn, args, hasReturnValue)
		return
	}

	if hitCount, ok := th.Breakpoint.HitCount[strconv.Itoa(th.GoroutineID)]; ok {
		log.Info("> %s%s(%s) %s:%d (hits goroutine(%d):%d total:%d) (PC: %#v)",
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
		log.Warn("> %s%s(%s) %s:%d (hits total:%d) (PC: %#v)",
			bpname,
			fn.Name(),
			args,
			t.formatPath(th.File),
			th.Line,
			th.Breakpoint.TotalHitCount,
			th.PC)
	}
	if th.Function != nil && th.Function.Optimized {
		log.Info(optimizedFunctionWarning)
	}

	printReturnValues(th)
	printBreakpointInfo(t, th, false)
}

func printBreakpointInfo(t *Term, th *api.Thread, tracepointOnNewline bool) {
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
		log.Info("")
	}

	if bpi.Goroutine != nil {
		tracepointnl()
		writeGoroutineLong(t, os.Stdout, bpi.Goroutine, "\t")
	}

	for _, v := range bpi.Variables {
		tracepointnl()
		log.Info("\t%s: %s", v.Name, v.MultilineString("\t", ""))
	}

	for _, v := range bpi.Locals {
		tracepointnl()
		if *bp.LoadLocals == longLoadConfig {
			log.Info("\t%s: %s", v.Name, v.MultilineString("\t", ""))
		} else {
			log.Info("\t%s: %s", v.Name, v.SinglelineString())
		}
	}

	if bp.LoadArgs != nil && *bp.LoadArgs == longLoadConfig {
		for _, v := range bpi.Arguments {
			tracepointnl()
			log.Info("\t%s: %s", v.Name, v.MultilineString("\t", ""))
		}
	}

	if bpi.Stacktrace != nil {
		tracepointnl()
		log.Info("\tStack:")
		printStack(t, os.Stdout, bpi.Stacktrace, "\t\t", false)
	}
}

func printTracepoint(t *Term, th *api.Thread, bpname string, fn *api.Function, args string, hasReturnValue bool) {
	if th.Breakpoint.Tracepoint {
		log.Error("> goroutine(%d): %s%s(%s)", th.GoroutineID, bpname, fn.Name(), args)
		if !hasReturnValue {
			log.Info("")
		}
		printBreakpointInfo(t, th, !hasReturnValue)
	}
	if th.Breakpoint.TraceReturn {
		retVals := make([]string, 0, len(th.ReturnValues))
		for _, v := range th.ReturnValues {
			retVals = append(retVals, v.SinglelineString())
		}
		log.Error(" => (%s)", strings.Join(retVals, ","))
	}
	if th.Breakpoint.TraceReturn || !hasReturnValue {
		if th.BreakpointInfo != nil && th.BreakpointInfo.Stacktrace != nil {
			log.Error("\tStack:")
			printStack(t, os.Stderr, th.BreakpointInfo.Stacktrace, "\t\t", false)
		}
	}
}

func printfile(t *Term, filename string, line int, showArrow bool) error {
	if filename == "" {
		return nil
	}

	lineCount := t.conf.GetSourceListLineCount()
	arrowLine := 0
	if showArrow {
		arrowLine = line
	}

	file, err := os.Open(t.substitutePath(filename))
	if err != nil {
		return err
	}
	defer file.Close()

	fi, _ := file.Stat()
	lastModExe := t.client.LastModified()
	if fi.ModTime().After(lastModExe) {
		log.Info("Warning: listing may not match stale executable")
	}

	return colorize.Print(t.stdout, file.Name(), file, line-lineCount, line+lineCount+1, arrowLine, t.colorEscapes)
}

// ExitRequestError is returned when the user
// exits Delve.
type ExitRequestError struct{}

func (ere ExitRequestError) Error() string {
	return ""
}

func exitCommand(t *Term, ctx callContext, args string) error {
	if args == "-c" {
		if !t.client.IsMulticlient() {
			return errors.New("not connected to an --accept-multiclient server")
		}
		t.quitContinue = true
	}
	return ExitRequestError{}
}

func getBreakpointByIDOrName(t *Term, arg string) (*api.Breakpoint, error) {
	if id, err := strconv.Atoi(arg); err == nil {
		return t.client.GetBreakpoint(id)
	}
	return t.client.GetBreakpointByName(arg)
}

func (c *Commands) onCmd(t *Term, ctx callContext, argstr string) error {
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
		f, err := ioutil.TempFile("", "dlv-on-cmd-")
		if err != nil {
			return err
		}
		defer func() {
			_ = os.Remove(f.Name())
		}()
		attrs := formatBreakpointAttrs("", ctx.Breakpoint, true)
		_, err = f.Write([]byte(strings.Join(attrs, "\n")))
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

func (c *Commands) parseBreakpointAttrs(t *Term, ctx callContext, r io.Reader) error {
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
		err := c.CallWithContext(scan.Text(), t, ctx)
		if err != nil {
			log.Info("%d: %s", lineno, err.Error())
		}
	}
	return scan.Err()
}

func conditionCmd(t *Term, ctx callContext, argstr string) error {
	args := config.Split2PartsBySpace(argstr)

	if len(args) < 2 {
		return fmt.Errorf("not enough arguments")
	}

	if args[0] == "-hitcount" {
		// hitcount breakpoint

		if ctx.Prefix == onPrefix {
			ctx.Breakpoint.HitCond = args[1]
			return nil
		}

		args = config.Split2PartsBySpace(args[1])
		if len(args) < 2 {
			return fmt.Errorf("not enough arguments")
		}

		bp, err := getBreakpointByIDOrName(t, args[0])
		if err != nil {
			return err
		}

		bp.HitCond = args[1]

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

func (c *Commands) executeFile(t *Term, name string) error {
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

		if err := c.Call(line, t); err != nil {
			if _, isExitRequest := err.(ExitRequestError); isExitRequest {
				return err
			}
			log.Info("%s:%d: %v", name, lineno, err)
		}
	}

	return scanner.Err()
}

func display(t *Term, ctx callContext, args string) error {
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
			return fmt.Errorf("not enough arguments")
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
		return fmt.Errorf("wrong arguments")
	}
	return nil
}

func dump(t *Term, ctx callContext, args string) error {
	if args == "" {
		return fmt.Errorf("not enough arguments")
	}
	dumpState, err := t.client.CoreDumpStart(args)
	if err != nil {
		return err
	}
	for {
		if dumpState.ThreadsDone != dumpState.ThreadsTotal {
			log.Info("\rDumping threads %d / %d...", dumpState.ThreadsDone, dumpState.ThreadsTotal)
		} else {
			log.Info("\rDumping memory %d / %d...", dumpState.MemDone, dumpState.MemTotal)
		}
		if !dumpState.Dumping {
			break
		}
		dumpState = t.client.CoreDumpWait(1000)
	}
	log.Info("")
	if dumpState.Err != "" {
		log.Error("error dumping: %s", dumpState.Err)
	} else if !dumpState.AllDone {
		log.Error("canceled")
	} else if dumpState.MemDone != dumpState.MemTotal {
		log.Error("Core dump could be incomplete")
	}
	return nil
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
		thing = strings.Title(thing)
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

func (t *Term) formatBreakpointLocation(bp *api.Breakpoint) string {
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
