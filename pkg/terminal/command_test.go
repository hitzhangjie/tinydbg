package terminal

import (
	"flag"
	"fmt"
	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/goversion"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/pkg/proc/test"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
	"github.com/hitzhangjie/dlv/service/rpccommon"
	"github.com/hitzhangjie/dlv/service/rpcx"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var testBackend, buildMode string

func TestMain(m *testing.M) {
	flag.StringVar(&testBackend, "backend", "", "selects backend")
	flag.StringVar(&buildMode, "test-buildmode", "", "selects build mode")
	var logConf string
	flag.StringVar(&logConf, "log", "", "configures logging")
	flag.Parse()
	test.DefaultTestBackend(&testBackend)
	if buildMode != "" && buildMode != "pie" {
		log.Error("unknown build mode %q", buildMode)
		os.Exit(1)
	}
	os.Exit(test.RunTestsWithFixtures(m))
}

type FakeTerminal struct {
	*Term
	t testing.TB
}

const logCommandOutput = false

func (ft *FakeTerminal) Exec(cmdstr string) (outstr string, err error) {
	outfh, err := ioutil.TempFile("", "cmdtestout")
	if err != nil {
		ft.t.Fatalf("could not create temporary file: %v", err)
	}

	stdout, stderr, termstdout := os.Stdout, os.Stderr, ft.Term.stdout
	os.Stdout, os.Stderr, ft.Term.stdout = outfh, outfh, outfh
	defer func() {
		os.Stdout, os.Stderr, ft.Term.stdout = stdout, stderr, termstdout
		outfh.Close()
		outbs, err1 := ioutil.ReadFile(outfh.Name())
		if err1 != nil {
			ft.t.Fatalf("could not read temporary output file: %v", err)
		}
		outstr = string(outbs)
		if logCommandOutput {
			ft.t.Logf("command %q -> %q", cmdstr, outstr)
		}
		os.Remove(outfh.Name())
	}()
	err = ft.cmds.Call(cmdstr, ft.Term)
	return
}

func (ft *FakeTerminal) ExecStarlark(starlarkProgram string) (outstr string, err error) {
	outfh, err := ioutil.TempFile("", "cmdtestout")
	if err != nil {
		ft.t.Fatalf("could not create temporary file: %v", err)
	}

	stdout, stderr, termstdout := os.Stdout, os.Stderr, ft.Term.stdout
	os.Stdout, os.Stderr, ft.Term.stdout = outfh, outfh, outfh
	defer func() {
		os.Stdout, os.Stderr, ft.Term.stdout = stdout, stderr, termstdout
		outfh.Close()
		outbs, err1 := ioutil.ReadFile(outfh.Name())
		if err1 != nil {
			ft.t.Fatalf("could not read temporary output file: %v", err)
		}
		outstr = string(outbs)
		if logCommandOutput {
			ft.t.Logf("command %q -> %q", starlarkProgram, outstr)
		}
		os.Remove(outfh.Name())
	}()
	_, err = ft.Term.starlarkEnv.Execute("<stdin>", starlarkProgram, "main", nil)
	return
}

func (ft *FakeTerminal) MustExec(cmdstr string) string {
	outstr, err := ft.Exec(cmdstr)
	if err != nil {
		ft.t.Errorf("output of %q: %q", cmdstr, outstr)
		ft.t.Fatalf("Error executing <%s>: %v", cmdstr, err)
	}
	return outstr
}

func (ft *FakeTerminal) MustExecStarlark(starlarkProgram string) string {
	outstr, err := ft.ExecStarlark(starlarkProgram)
	if err != nil {
		ft.t.Errorf("output of %q: %q", starlarkProgram, outstr)
		ft.t.Fatalf("Error executing <%s>: %v", starlarkProgram, err)
	}
	return outstr
}

func (ft *FakeTerminal) AssertExec(cmdstr, tgt string) {
	out := ft.MustExec(cmdstr)
	if out != tgt {
		ft.t.Fatalf("Error executing %q, expected %q got %q", cmdstr, tgt, out)
	}
}

func (ft *FakeTerminal) AssertExecError(cmdstr, tgterr string) {
	_, err := ft.Exec(cmdstr)
	if err == nil {
		ft.t.Fatalf("Expected error executing %q", cmdstr)
	}
	if err.Error() != tgterr {
		ft.t.Fatalf("Expected error %q executing %q, got error %q", tgterr, cmdstr, err.Error())
	}
}

func withTestTerminal(name string, t testing.TB, fn func(*FakeTerminal)) {
	withTestTerminalBuildFlags(name, t, 0, fn)
}

func withTestTerminalBuildFlags(name string, t testing.TB, buildFlags test.BuildFlags, fn func(*FakeTerminal)) {
	os.Setenv("TERM", "dumb")
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("couldn't start listener: %s\n", err)
	}
	defer listener.Close()
	if buildMode == "pie" {
		buildFlags |= test.BuildModePIE
	}
	server := rpccommon.NewServer(&service.Config{
		Listener:       listener,
		ProcessArgs:    []string{test.BuildFixture(name, buildFlags).Path},
		DebuggerConfig: debugger.Config{},
	})
	if err := server.Run(); err != nil {
		t.Fatal(err)
	}
	client := rpcx.NewClient(listener.Addr().String())
	defer func() {
		client.Detach(true)
	}()

	ft := &FakeTerminal{
		t:    t,
		Term: New(client, &config.Config{}),
	}
	fn(ft)
}

func TestCommandDefault(t *testing.T) {
	var (
		cmds = Commands{}
		cmd  = cmds.Find("non-existant-command", noPrefix).cmdFn
	)

	err := cmd(nil, callContext{}, "")
	if err == nil {
		t.Fatal("cmd() did not default")
	}

	if err.Error() != "command not available" {
		t.Fatal("wrong command output")
	}
}

func TestCommandReplayWithoutPreviousCommand(t *testing.T) {
	var (
		cmds = DebugCommands(nil)
		cmd  = cmds.Find("", noPrefix).cmdFn
		err  = cmd(nil, callContext{}, "")
	)

	if err != nil {
		t.Error("Null command not returned", err)
	}
}

func TestCommandThread(t *testing.T) {
	var (
		cmds = DebugCommands(nil)
		cmd  = cmds.Find("thread", noPrefix).cmdFn
	)

	err := cmd(nil, callContext{}, "")
	if err == nil {
		t.Fatal("thread terminal command did not default")
	}

	if err.Error() != "you must specify a thread" {
		t.Fatal("wrong command output: ", err.Error())
	}
}

func TestExecuteFile(t *testing.T) {
	breakCount := 0
	traceCount := 0
	c := &Commands{
		client: nil,
		cmds: []command{
			{aliases: []string{"trace"}, cmdFn: func(t *Term, ctx callContext, args string) error {
				traceCount++
				return nil
			}},
			{aliases: []string{"break"}, cmdFn: func(t *Term, ctx callContext, args string) error {
				breakCount++
				return nil
			}},
		},
	}

	fixturesDir := test.FindFixturesDir()
	err := c.executeFile(nil, filepath.Join(fixturesDir, "bpfile"))
	if err != nil {
		t.Fatalf("executeFile: %v", err)
	}

	if breakCount != 1 || traceCount != 1 {
		t.Fatalf("Wrong counts break: %d trace: %d\n", breakCount, traceCount)
	}
}

func TestTrace(t *testing.T) {
	test.AllowRecording(t)
	withTestTerminal("issue573", t, func(term *FakeTerminal) {
		term.MustExec("trace foo")
		out, _ := term.Exec("continue")
		// The output here is a little strange, but we don't filter stdout vs stderr so it gets jumbled.
		// Therefore we assert about the call and return values separately.
		if !strings.Contains(out, "> goroutine(1): main.foo(99, 9801)") {
			t.Fatalf("Wrong output for tracepoint: %s", out)
		}
		if !strings.Contains(out, "=> (9900)") {
			t.Fatalf("Wrong output for tracepoint return value: %s", out)
		}
	})
}

func TestTraceWithName(t *testing.T) {
	test.AllowRecording(t)
	withTestTerminal("issue573", t, func(term *FakeTerminal) {
		term.MustExec("trace foobar foo")
		out, _ := term.Exec("continue")
		// The output here is a little strange, but we don't filter stdout vs stderr so it gets jumbled.
		// Therefore we assert about the call and return values separately.
		if !strings.Contains(out, "> goroutine(1): [foobar] main.foo(99, 9801)") {
			t.Fatalf("Wrong output for tracepoint: %s", out)
		}
		if !strings.Contains(out, "=> (9900)") {
			t.Fatalf("Wrong output for tracepoint return value: %s", out)
		}
	})
}

func TestTraceOnNonFunctionEntry(t *testing.T) {
	test.AllowRecording(t)
	withTestTerminal("issue573", t, func(term *FakeTerminal) {
		term.MustExec("trace foobar issue573.go:19")
		out, _ := term.Exec("continue")
		if !strings.Contains(out, "> goroutine(1): [foobar] main.foo(99, 9801)") {
			t.Fatalf("Wrong output for tracepoint: %s", out)
		}
		if strings.Contains(out, "=> (9900)") {
			t.Fatalf("Tracepoint on non-function locspec should not have return value:\n%s", out)
		}
	})
}

func TestExitStatus(t *testing.T) {
	withTestTerminal("continuetestprog", t, func(term *FakeTerminal) {
		term.Exec("continue")
		status, err := term.handleExit()
		if err != nil {
			t.Fatal(err)
		}
		if status != 0 {
			t.Fatalf("incorrect exit status, expected 0, got %d", status)
		}
	})
}

func TestScopePrefix(t *testing.T) {
	const goroutinesLinePrefix = "  Goroutine "
	const goroutinesCurLinePrefix = "* Goroutine "
	test.AllowRecording(t)

	lenient := 0

	withTestTerminal("goroutinestackprog", t, func(term *FakeTerminal) {
		term.MustExec("b stacktraceme")
		term.MustExec("continue")

		goroutinesOut := strings.Split(term.MustExec("goroutines"), "\n")
		agoroutines := []int{}
		nonagoroutines := []int{}
		curgid := -1

		for _, line := range goroutinesOut {
			iscur := strings.HasPrefix(line, goroutinesCurLinePrefix)
			if !iscur && !strings.HasPrefix(line, goroutinesLinePrefix) {
				continue
			}

			dash := strings.Index(line, " - ")
			if dash < 0 {
				continue
			}

			gid, err := strconv.Atoi(line[len(goroutinesLinePrefix):dash])
			if err != nil {
				continue
			}

			if iscur {
				curgid = gid
			}

			if idx := strings.Index(line, " main.agoroutine "); idx < 0 {
				nonagoroutines = append(nonagoroutines, gid)
				continue
			}

			agoroutines = append(agoroutines, gid)
		}

		if len(agoroutines) > 10 {
			t.Fatalf("Output of goroutines did not have 10 goroutines stopped on main.agoroutine (%d found): %q", len(agoroutines), goroutinesOut)
		}

		if len(agoroutines) < 10 {
			extraAgoroutines := 0
			for _, gid := range nonagoroutines {
				stackOut := strings.Split(term.MustExec(fmt.Sprintf("goroutine %d stack", gid)), "\n")
				for _, line := range stackOut {
					if strings.HasSuffix(line, " main.agoroutine") {
						extraAgoroutines++
						break
					}
				}
			}
			if len(agoroutines)+extraAgoroutines < 10-lenient {
				t.Fatalf("Output of goroutines did not have 10 goroutines stopped on main.agoroutine (%d+%d found): %q", len(agoroutines), extraAgoroutines, goroutinesOut)
			}
		}

		if curgid < 0 {
			t.Fatalf("Could not find current goroutine in output of goroutines: %q", goroutinesOut)
		}

		seen := make([]bool, 10)
		for _, gid := range agoroutines {
			stackOut := strings.Split(term.MustExec(fmt.Sprintf("goroutine %d stack", gid)), "\n")
			fid := -1
			for _, line := range stackOut {
				line = strings.TrimLeft(line, " ")
				space := strings.Index(line, " ")
				if space < 0 {
					continue
				}
				curfid, err := strconv.Atoi(line[:space])
				if err != nil {
					continue
				}

				if idx := strings.Index(line, " main.agoroutine"); idx >= 0 {
					fid = curfid
					break
				}
			}
			if fid < 0 {
				t.Fatalf("Could not find frame for goroutine %d: %q", gid, stackOut)
			}
			term.AssertExec(fmt.Sprintf("goroutine     %d    frame     %d     locals", gid, fid), "(no locals)\n")
			argsOut := strings.Split(term.MustExec(fmt.Sprintf("goroutine %d frame %d args", gid, fid)), "\n")
			if len(argsOut) != 4 || argsOut[3] != "" {
				t.Fatalf("Wrong number of arguments in goroutine %d frame %d: %v", gid, fid, argsOut)
			}
			out := term.MustExec(fmt.Sprintf("goroutine %d frame %d p i", gid, fid))
			ival, err := strconv.Atoi(out[:len(out)-1])
			if err != nil {
				t.Fatalf("could not parse value %q of i for goroutine %d frame %d: %v", out, gid, fid, err)
			}
			seen[ival] = true
		}

		for i := range seen {
			if !seen[i] {
				if lenient > 0 {
					lenient--
				} else {
					t.Fatalf("goroutine %d not found", i)
				}
			}
		}

		term.MustExec("c")

		term.AssertExecError("frame", "not enough arguments")
		term.AssertExecError(fmt.Sprintf("goroutine %d frame 10 locals", curgid), fmt.Sprintf("Frame 10 does not exist in goroutine %d", curgid))
		term.AssertExecError("goroutine 9000 locals", "unknown goroutine 9000")

		term.AssertExecError("print n", "could not find symbol value for n")
		term.AssertExec("frame 1 print n", "3\n")
		term.AssertExec("frame 2 print n", "2\n")
		term.AssertExec("frame 3 print n", "1\n")
		term.AssertExec("frame 4 print n", "0\n")
		term.AssertExecError("frame 5 print n", "could not find symbol value for n")

		term.MustExec("frame 2")
		term.AssertExec("print n", "2\n")
		term.MustExec("frame 4")
		term.AssertExec("print n", "0\n")
		term.MustExec("down")
		term.AssertExec("print n", "1\n")
		term.MustExec("down 2")
		term.AssertExec("print n", "3\n")
		term.AssertExecError("down 2", "Invalid frame -1")
		term.AssertExec("print n", "3\n")
		term.MustExec("up 2")
		term.AssertExec("print n", "1\n")
		term.AssertExecError("up 100", "Invalid frame 103")
		term.AssertExec("print n", "1\n")

		term.MustExec("step")
		term.AssertExecError("print n", "could not find symbol value for n")
		term.MustExec("frame 2")
		term.AssertExec("print n", "2\n")
	})
}

func TestOnPrefix(t *testing.T) {
	const prefix = "\ti: "
	test.AllowRecording(t)
	lenient := false

	withTestTerminal("goroutinestackprog", t, func(term *FakeTerminal) {
		term.MustExec("b agobp main.agoroutine")
		term.MustExec("on agobp print i")

		seen := make([]bool, 10)

		for {
			outstr, err := term.Exec("continue")
			if err != nil {
				if !strings.Contains(err.Error(), "exited") {
					t.Fatalf("Unexpected error executing 'continue': %v", err)
				}
				break
			}
			out := strings.Split(outstr, "\n")

			for i := range out {
				if !strings.HasPrefix(out[i], prefix) {
					continue
				}
				id, err := strconv.Atoi(out[i][len(prefix):])
				if err != nil {
					continue
				}
				if seen[id] {
					t.Fatalf("Goroutine %d seen twice\n", id)
				}
				seen[id] = true
			}
		}

		for i := range seen {
			if !seen[i] {
				if lenient {
					lenient = false
				} else {
					t.Fatalf("Goroutine %d not seen\n", i)
				}
			}
		}
	})
}

func TestNoVars(t *testing.T) {
	test.AllowRecording(t)
	withTestTerminal("locationsUpperCase", t, func(term *FakeTerminal) {
		term.MustExec("b main.main")
		term.MustExec("continue")
		term.AssertExec("args", "(no args)\n")
		term.AssertExec("locals", "(no locals)\n")
		term.AssertExec("vars filterThatMatchesNothing", "(no vars)\n")
	})
}

func TestOnPrefixLocals(t *testing.T) {
	const prefix = "\ti: "
	test.AllowRecording(t)
	withTestTerminal("goroutinestackprog", t, func(term *FakeTerminal) {
		term.MustExec("b agobp main.agoroutine")
		term.MustExec("on agobp args -v")

		seen := make([]bool, 10)

		for {
			outstr, err := term.Exec("continue")
			if err != nil {
				if !strings.Contains(err.Error(), "exited") {
					t.Fatalf("Unexpected error executing 'continue': %v", err)
				}
				break
			}
			out := strings.Split(outstr, "\n")

			for i := range out {
				if !strings.HasPrefix(out[i], prefix) {
					continue
				}
				id, err := strconv.Atoi(out[i][len(prefix):])
				if err != nil {
					continue
				}
				if seen[id] {
					t.Fatalf("Goroutine %d seen twice\n", id)
				}
				seen[id] = true
			}
		}

		for i := range seen {
			if !seen[i] {
				t.Fatalf("Goroutine %d not seen\n", i)
			}
		}
	})
}

func countOccurrences(s, needle string) int {
	count := 0
	for {
		idx := strings.Index(s, needle)
		if idx < 0 {
			break
		}
		count++
		s = s[idx+len(needle):]
	}
	return count
}

func listIsAt(t *testing.T, term *FakeTerminal, listcmd string, cur, start, end int) {
	t.Helper()
	outstr := term.MustExec(listcmd)
	lines := strings.Split(outstr, "\n")

	t.Logf("%q: %q", listcmd, outstr)

	if cur >= 0 && !strings.Contains(lines[0], fmt.Sprintf(":%d", cur)) {
		t.Fatalf("Could not find current line number in first output line: %q", lines[0])
	}

	re := regexp.MustCompile(`(=>)?\s+(\d+):`)

	outStart, outEnd := 0, 0

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		v := re.FindStringSubmatch(line)
		if len(v) != 3 {
			continue
		}
		curline, _ := strconv.Atoi(v[2])
		if v[1] == "=>" {
			if cur != curline {
				t.Fatalf("Wrong current line, got %d expected %d", curline, cur)
			}
		}
		if outStart == 0 {
			outStart = curline
		}
		outEnd = curline
	}

	if start != -1 || end != -1 {
		if outStart != start || outEnd != end {
			t.Fatalf("Wrong output range, got %d:%d expected %d:%d", outStart, outEnd, start, end)
		}
	}
}

func TestListCmd(t *testing.T) {
	withTestTerminal("testvariables", t, func(term *FakeTerminal) {
		term.MustExec("continue")
		term.MustExec("continue")
		listIsAt(t, term, "list", 27, 22, 32)
		listIsAt(t, term, "list 69", 69, 64, 74)
		listIsAt(t, term, "frame 1 list", 66, 61, 71)
		listIsAt(t, term, "frame 1 list 69", 69, 64, 74)
		_, err := term.Exec("frame 50 list")
		if err == nil {
			t.Fatalf("Expected error requesting 50th frame")
		}
		listIsAt(t, term, "list testvariables.go:1", -1, 1, 6)
		listIsAt(t, term, "list testvariables.go:10000", -1, 0, 0)
	})
}

func TestNextWithCount(t *testing.T) {
	test.AllowRecording(t)
	withTestTerminal("nextcond", t, func(term *FakeTerminal) {
		term.MustExec("break main.main")
		listIsAt(t, term, "continue", 8, -1, -1)
		listIsAt(t, term, "next 2", 10, -1, -1)
	})
}

func TestRestart(t *testing.T) {
	withTestTerminal("restartargs", t, func(term *FakeTerminal) {
		term.MustExec("break main.printArgs")
		term.MustExec("continue")
		if out := term.MustExec("print main.args"); !strings.Contains(out, ", []") {
			t.Fatalf("wrong args: %q", out)
		}
		// Reset the arg list
		term.MustExec("restart hello")
		term.MustExec("continue")
		if out := term.MustExec("print main.args"); !strings.Contains(out, ", [\"hello\"]") {
			t.Fatalf("wrong args: %q ", out)
		}
		// Restart w/o arg should retain the current args.
		term.MustExec("restart")
		term.MustExec("continue")
		if out := term.MustExec("print main.args"); !strings.Contains(out, ", [\"hello\"]") {
			t.Fatalf("wrong args: %q ", out)
		}
		// Empty arg list
		term.MustExec("restart -noargs")
		term.MustExec("continue")
		if out := term.MustExec("print main.args"); !strings.Contains(out, ", []") {
			t.Fatalf("wrong args: %q ", out)
		}
	})
}

func findCmdName(c *Commands, cmdstr string, prefix cmdPrefix) string {
	for _, v := range c.cmds {
		if v.match(cmdstr) {
			if prefix != noPrefix && v.allowedPrefixes&prefix == 0 {
				continue
			}
			return v.aliases[0]
		}
	}
	return ""
}

func TestConfig(t *testing.T) {
	var term Term
	term.conf = &config.Config{}
	term.cmds = DebugCommands(nil)

	err := configureCmd(&term, callContext{}, "nonexistent-parameter 10")
	if err == nil {
		t.Fatalf("expected error executing configureCmd(nonexistent-parameter)")
	}

	err = configureCmd(&term, callContext{}, "max-string-len 10")
	if err != nil {
		t.Fatalf("error executing configureCmd(max-string-len): %v", err)
	}
	if term.conf.MaxStringLen == nil {
		t.Fatalf("expected MaxStringLen 10, got nil")
	}
	if *term.conf.MaxStringLen != 10 {
		t.Fatalf("expected MaxStringLen 10, got: %d", *term.conf.MaxStringLen)
	}
	err = configureCmd(&term, callContext{}, "show-location-expr   true")
	if err != nil {
		t.Fatalf("error executing configureCmd(show-location-expr   true)")
	}
	if term.conf.ShowLocationExpr != true {
		t.Fatalf("expected ShowLocationExpr true, got false")
	}
	err = configureCmd(&term, callContext{}, "max-variable-recurse 4")
	if err != nil {
		t.Fatalf("error executing configureCmd(max-variable-recurse): %v", err)
	}
	if term.conf.MaxVariableRecurse == nil {
		t.Fatalf("expected MaxVariableRecurse 4, got nil")
	}
	if *term.conf.MaxVariableRecurse != 4 {
		t.Fatalf("expected MaxVariableRecurse 4, got: %d", *term.conf.MaxVariableRecurse)
	}

	err = configureCmd(&term, callContext{}, "substitute-path a b")
	if err != nil {
		t.Fatalf("error executing configureCmd(substitute-path a b): %v", err)
	}
	if len(term.conf.SubstitutePath) != 1 || (term.conf.SubstitutePath[0] != config.SubstitutePathRule{From: "a", To: "b"}) {
		t.Fatalf("unexpected SubstitutePathRules after insert %v", term.conf.SubstitutePath)
	}

	err = configureCmd(&term, callContext{}, "substitute-path a")
	if err != nil {
		t.Fatalf("error executing configureCmd(substitute-path a): %v", err)
	}
	if len(term.conf.SubstitutePath) != 0 {
		t.Fatalf("unexpected SubstitutePathRules after delete %v", term.conf.SubstitutePath)
	}

	err = configureCmd(&term, callContext{}, "alias print blah")
	if err != nil {
		t.Fatalf("error executing configureCmd(alias print blah): %v", err)
	}
	if len(term.conf.Aliases["print"]) != 1 {
		t.Fatalf("aliases not changed after configure command %v", term.conf.Aliases)
	}
	if findCmdName(term.cmds, "blah", noPrefix) != "print" {
		t.Fatalf("new alias not found")
	}

	err = configureCmd(&term, callContext{}, "alias blah")
	if err != nil {
		t.Fatalf("error executing configureCmd(alias blah): %v", err)
	}
	if len(term.conf.Aliases["print"]) != 0 {
		t.Fatalf("alias not removed after configure command %v", term.conf.Aliases)
	}
	if findCmdName(term.cmds, "blah", noPrefix) != "" {
		t.Fatalf("new alias found after delete")
	}
}

func TestPrintContextParkedGoroutine(t *testing.T) {
	withTestTerminal("goroutinestackprog", t, func(term *FakeTerminal) {
		term.MustExec("break stacktraceme")
		term.MustExec("continue")

		// pick a goroutine that isn't running on a thread
		gid := ""
		gout := strings.Split(term.MustExec("goroutines"), "\n")
		t.Logf("goroutines -> %q", gout)
		for _, gline := range gout {
			if !strings.Contains(gline, "thread ") && strings.Contains(gline, "agoroutine") {
				if dash := strings.Index(gline, " - "); dash > 0 {
					gid = gline[len("  Goroutine "):dash]
					break
				}
			}
		}

		t.Logf("picked %q", gid)
		term.MustExec(fmt.Sprintf("goroutine %s", gid))

		frameout := strings.Split(term.MustExec("frame 0"), "\n")
		t.Logf("frame 0 -> %q", frameout)
		if strings.Contains(frameout[0], "stacktraceme") {
			t.Fatal("bad output for `frame 0` command on a parked goorutine")
		}

		listout := strings.Split(term.MustExec("list"), "\n")
		t.Logf("list -> %q", listout)
		if strings.Contains(listout[0], "stacktraceme") {
			t.Fatal("bad output for list command on a parked goroutine")
		}
	})
}

func TestStepOutReturn(t *testing.T) {
	ver, _ := goversion.Parse(runtime.Version())
	if ver.Major >= 0 && !ver.AfterOrEqual(goversion.GoVersion{Major: 1, Minor: 10, Rev: -1}) {
		t.Skip("return variables aren't marked on 1.9 or earlier")
	}
	withTestTerminal("stepoutret", t, func(term *FakeTerminal) {
		term.MustExec("break main.stepout")
		term.MustExec("continue")
		out := term.MustExec("stepout")
		t.Logf("output: %q", out)
		if !strings.Contains(out, "num: ") || !strings.Contains(out, "str: ") {
			t.Fatal("could not find parameter")
		}
	})
}

func TestOptimizationCheck(t *testing.T) {
	withTestTerminal("continuetestprog", t, func(term *FakeTerminal) {
		term.MustExec("break main.main")
		out := term.MustExec("continue")
		t.Logf("output %q", out)
		if strings.Contains(out, optimizedFunctionWarning) {
			t.Fatal("optimized function warning")
		}
	})

	if goversion.VersionAfterOrEqual(runtime.Version(), 1, 10) {
		withTestTerminalBuildFlags("continuetestprog", t, test.EnableOptimization|test.EnableInlining, func(term *FakeTerminal) {
			term.MustExec("break main.main")
			out := term.MustExec("continue")
			t.Logf("output %q", out)
			if !strings.Contains(out, optimizedFunctionWarning) {
				t.Fatal("optimized function warning missing")
			}
		})
	}
}

func TestTruncateStacktrace(t *testing.T) {
	const stacktraceTruncatedMessage = "(truncated)"
	withTestTerminal("stacktraceprog", t, func(term *FakeTerminal) {
		term.MustExec("break main.stacktraceme")
		term.MustExec("continue")
		out1 := term.MustExec("stack")
		t.Logf("untruncated output %q", out1)
		if strings.Contains(out1, stacktraceTruncatedMessage) {
			t.Fatalf("stacktrace was truncated")
		}
		out2 := term.MustExec("stack 1")
		t.Logf("truncated output %q", out2)
		if !strings.Contains(out2, stacktraceTruncatedMessage) {
			t.Fatalf("stacktrace was not truncated")
		}
	})
}

func findStarFile(name string) string {
	return filepath.Join(test.FindFixturesDir(), name+".star")
}

func TestExamineMemoryCmd(t *testing.T) {
	withTestTerminal("examinememory", t, func(term *FakeTerminal) {
		term.MustExec("break examinememory.go:19")
		term.MustExec("break examinememory.go:24")
		term.MustExec("continue")

		addressStr := strings.TrimSpace(term.MustExec("p bspUintptr"))
		address, err := strconv.ParseInt(addressStr, 0, 64)
		if err != nil {
			t.Fatalf("could convert %s into int64, err %s", addressStr, err)
		}

		res := term.MustExec("examinemem  -count 52 -fmt hex " + addressStr)
		t.Logf("the result of examining memory \n%s", res)
		// check first line
		firstLine := fmt.Sprintf("%#x:   0x0a   0x0b   0x0c   0x0d   0x0e   0x0f   0x10   0x11", address)
		if !strings.Contains(res, firstLine) {
			t.Fatalf("expected first line: %s", firstLine)
		}

		// check last line
		lastLine := fmt.Sprintf("%#x:   0x3a   0x3b   0x3c   0x00", address+6*8)
		if !strings.Contains(res, lastLine) {
			t.Fatalf("expected last line: %s", lastLine)
		}

		// second examining memory
		term.MustExec("continue")
		res = term.MustExec("x -count 52 -fmt bin " + addressStr)
		t.Logf("the second result of examining memory result \n%s", res)

		// check first line
		firstLine = fmt.Sprintf("%#x:   11111111   00001011   00001100   00001101", address)
		if !strings.Contains(res, firstLine) {
			t.Fatalf("expected first line: %s", firstLine)
		}

		// third examining memory: -x addr
		res = term.MustExec("examinemem -x " + addressStr)
		t.Logf("the third result of examining memory result \n%s", res)
		firstLine = fmt.Sprintf("%#x:   0xff", address)
		if !strings.Contains(res, firstLine) {
			t.Fatalf("expected first line: %s", firstLine)
		}

		// fourth examining memory: -x addr + offset
		res = term.MustExec("examinemem -x " + addressStr + " + 8")
		t.Logf("the fourth result of examining memory result \n%s", res)
		firstLine = fmt.Sprintf("%#x:   0x12", address+8)
		if !strings.Contains(res, firstLine) {
			t.Fatalf("expected first line: %s", firstLine)
		}
		// fifth examining memory: -x &var
		res = term.MustExec("examinemem -x &bs[0]")
		t.Logf("the fifth result of examining memory result \n%s", res)
		firstLine = fmt.Sprintf("%#x:   0xff", address)
		if !strings.Contains(res, firstLine) {
			t.Fatalf("expected first line: %s", firstLine)
		}

		// sixth examining memory: -fmt and double spaces
		res = term.MustExec("examinemem -fmt  hex  -x &bs[0]")
		t.Logf("the sixth result of examining memory result \n%s", res)
		firstLine = fmt.Sprintf("%#x:   0xff", address)
		if !strings.Contains(res, firstLine) {
			t.Fatalf("expected first line: %s", firstLine)
		}
	})

	withTestTerminal("testvariables2", t, func(term *FakeTerminal) {
		tests := []struct {
			Expr string
			Want int
		}{
			{Expr: "&i1", Want: 1},
			{Expr: "&i2", Want: 2},
			{Expr: "p1", Want: 1},
			{Expr: "*pp1", Want: 1},
			{Expr: "&str1[1]", Want: '1'},
			{Expr: "c1.pb", Want: 1},
			{Expr: "&c1.pb.a", Want: 1},
			{Expr: "&c1.pb.a.A", Want: 1},
			{Expr: "&c1.pb.a.B", Want: 2},
		}
		term.MustExec("continue")
		for _, test := range tests {
			res := term.MustExec("examinemem -fmt dec -x " + test.Expr)
			// strip addr from output, e.g. "0xc0000160b8:   023" -> "023"
			res = strings.TrimSpace(strings.Split(res, ":")[1])
			got, err := strconv.Atoi(res)
			if err != nil {
				t.Fatalf("expr=%q err=%s", test.Expr, err)
			} else if got != test.Want {
				t.Errorf("expr=%q got=%d want=%d", test.Expr, got, test.Want)
			}
		}
	})
}

func TestPrintOnTracepoint(t *testing.T) {
	withTestTerminal("increment", t, func(term *FakeTerminal) {
		term.MustExec("trace main.Increment")
		term.MustExec("on 1 print y+1")
		out, _ := term.Exec("continue")
		if !strings.Contains(out, "y+1: 4") || !strings.Contains(out, "y+1: 2") || !strings.Contains(out, "y+1: 1") {
			t.Errorf("output did not contain breakpoint information: %q", out)
		}
	})
}

func TestPrintCastToInterface(t *testing.T) {
	withTestTerminal("testvariables2", t, func(term *FakeTerminal) {
		term.MustExec("continue")
		out := term.MustExec(`p (*"interface {}")(uintptr(&iface2))`)
		t.Logf("%q", out)
	})
}

func TestParseNewArgv(t *testing.T) {
	testCases := []struct {
		in       string
		tgtargs  string
		tgtredir string
		tgterr   string
	}{
		{"-noargs", "", " |  | ", ""},
		{"-noargs arg1", "", "", "too many arguments to restart"},
		{"arg1 arg2", "arg1 | arg2", " |  | ", ""},
		{"arg1 arg2 <input.txt", "arg1 | arg2", "input.txt |  | ", ""},
		{"arg1 arg2 < input.txt", "arg1 | arg2", "input.txt |  | ", ""},
		{"<input.txt", "", "input.txt |  | ", ""},
		{"< input.txt", "", "input.txt |  | ", ""},
		{"arg1 < input.txt > output.txt 2> error.txt", "arg1", "input.txt | output.txt | error.txt", ""},
		{"< input.txt > output.txt 2> error.txt", "", "input.txt | output.txt | error.txt", ""},
		{"arg1 <input.txt >output.txt 2>error.txt", "arg1", "input.txt | output.txt | error.txt", ""},
		{"<input.txt >output.txt 2>error.txt", "", "input.txt | output.txt | error.txt", ""},
		{"<input.txt <input2.txt", "", "", "redirect error: stdin redirected twice"},
	}

	for _, tc := range testCases {
		resetArgs, newArgv, newRedirects, err := parseNewArgv(tc.in)
		t.Logf("%q -> %q %q %v\n", tc.in, newArgv, newRedirects, err)
		if tc.tgterr != "" {
			if err == nil {
				t.Errorf("Expected error %q, got no error", tc.tgterr)
			} else if errstr := err.Error(); errstr != tc.tgterr {
				t.Errorf("Expected error %q, got error %q", tc.tgterr, errstr)
			}
		} else {
			if !resetArgs {
				t.Errorf("parse error, resetArgs is false")
				continue
			}
			argvstr := strings.Join(newArgv, " | ")
			if argvstr != tc.tgtargs {
				t.Errorf("Expected new arguments %q, got %q", tc.tgtargs, argvstr)
			}
			redirstr := strings.Join(newRedirects[:], " | ")
			if redirstr != tc.tgtredir {
				t.Errorf("Expected new redirects %q, got %q", tc.tgtredir, redirstr)
			}
		}
	}
}

func TestContinueUntil(t *testing.T) {
	withTestTerminal("continuetestprog", t, func(term *FakeTerminal) {
		if runtime.GOARCH != "386" {
			listIsAt(t, term, "continue main.main", 16, -1, -1)
		} else {
			listIsAt(t, term, "continue main.main", 17, -1, -1)
		}
		listIsAt(t, term, "continue main.sayhi", 12, -1, -1)
	})
}

func TestContinueUntilExistingBreakpoint(t *testing.T) {
	withTestTerminal("continuetestprog", t, func(term *FakeTerminal) {
		term.MustExec("break main.main")
		if runtime.GOARCH != "386" {
			listIsAt(t, term, "continue main.main", 16, -1, -1)
		} else {
			listIsAt(t, term, "continue main.main", 17, -1, -1)
		}
		listIsAt(t, term, "continue main.sayhi", 12, -1, -1)
	})
}

func TestPrintFormat(t *testing.T) {
	withTestTerminal("testvariables2", t, func(term *FakeTerminal) {
		term.MustExec("continue")
		out := term.MustExec("print %#x m2[1].B")
		if !strings.Contains(out, "0xb\n") {
			t.Fatalf("output did not contain '0xb': %q", out)
		}
	})
}

func TestHitCondBreakpoint(t *testing.T) {
	withTestTerminal("break", t, func(term *FakeTerminal) {
		term.MustExec("break bp1 main.main:4")
		term.MustExec("condition -hitcount bp1 > 2")
		listIsAt(t, term, "continue", 7, -1, -1)
		out := term.MustExec("print i")
		t.Logf("%q", out)
		if !strings.Contains(out, "3\n") {
			t.Fatalf("wrong value of i")
		}
	})
}

func TestBreakpointEditing(t *testing.T) {
	term := &FakeTerminal{
		t:    t,
		Term: New(nil, &config.Config{}),
	}
	_ = term

	var testCases = []struct {
		inBp    *api.Breakpoint
		inBpStr string
		edit    string
		outBp   *api.Breakpoint
	}{
		{ // tracepoint -> breakpoint
			&api.Breakpoint{Tracepoint: true},
			"trace",
			"",
			&api.Breakpoint{}},
		{ // breakpoint -> tracepoint
			&api.Breakpoint{Variables: []string{"a"}},
			"print a",
			"print a\ntrace",
			&api.Breakpoint{Tracepoint: true, Variables: []string{"a"}}},
		{ // add print var
			&api.Breakpoint{Variables: []string{"a"}},
			"print a",
			"print b\nprint a\n",
			&api.Breakpoint{Variables: []string{"b", "a"}}},
		{ // add goroutine flag
			&api.Breakpoint{},
			"",
			"goroutine",
			&api.Breakpoint{Goroutine: true}},
		{ // remove goroutine flag
			&api.Breakpoint{Goroutine: true},
			"goroutine",
			"",
			&api.Breakpoint{}},
		{ // add stack directive
			&api.Breakpoint{},
			"",
			"stack 10",
			&api.Breakpoint{Stacktrace: 10}},
		{ // remove stack directive
			&api.Breakpoint{Stacktrace: 20},
			"stack 20",
			"print a",
			&api.Breakpoint{Variables: []string{"a"}}},
		{ // add condition
			&api.Breakpoint{Variables: []string{"a"}},
			"print a",
			"print a\ncond a < b",
			&api.Breakpoint{Variables: []string{"a"}, Cond: "a < b"}},
		{ // remove condition
			&api.Breakpoint{Cond: "a < b"},
			"cond a < b",
			"",
			&api.Breakpoint{}},
		{ // change condition
			&api.Breakpoint{Cond: "a < b"},
			"cond a < b",
			"cond a < 5",
			&api.Breakpoint{Cond: "a < 5"}},
		{ // change hitcount condition
			&api.Breakpoint{HitCond: "% 2"},
			"cond -hitcount % 2",
			"cond -hitcount = 2",
			&api.Breakpoint{HitCond: "= 2"}},
	}

	for _, tc := range testCases {
		bp := *tc.inBp
		bpStr := strings.Join(formatBreakpointAttrs("", &bp, true), "\n")
		if bpStr != tc.inBpStr {
			t.Errorf("Expected %q got %q for:\n%#v", tc.inBpStr, bpStr, tc.inBp)
		}
		ctx := callContext{Prefix: onPrefix, Scope: api.EvalScope{GoroutineID: -1, Frame: 0, DeferredCall: 0}, Breakpoint: &bp}
		err := term.cmds.parseBreakpointAttrs(nil, ctx, strings.NewReader(tc.edit))
		if err != nil {
			t.Errorf("Unexpected error during edit %q", tc.edit)
		}
		if !reflect.DeepEqual(bp, *tc.outBp) {
			t.Errorf("mismatch after edit\nexpected: %#v\ngot: %#v", tc.outBp, bp)
		}
	}
}
