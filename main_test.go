package main_test

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hitzhangjie/tinydbg/pkg/goversion"
	protest "github.com/hitzhangjie/tinydbg/pkg/proc/test"
	"github.com/hitzhangjie/tinydbg/service/rpc2"
	"golang.org/x/tools/go/packages"
)

func TestMain(m *testing.M) {
	protest.RunTestsWithFixtures(m)
}

func assertNoError(err error, t testing.TB, s string) {
	t.Helper()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fname := filepath.Base(file)
		t.Fatalf("failed assertion at %s:%d: %s - %s\n", fname, line, s, err)
	}
}

func TestBuild(t *testing.T) {
	const listenAddr = "127.0.0.1:40573"

	tinydbgbin := protest.GetTinyDbgBinary(t)

	fixtures := protest.FindFixturesDir()

	buildtestdir := filepath.Join(fixtures, "buildtest")

	cmd := exec.Command(tinydbgbin, "debug", "--headless=true", "--listen="+listenAddr, "--log", "--log-output=debugger,rpc")
	cmd.Dir = buildtestdir
	stderr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := stderr.Close(); err != nil {
			t.Logf("Failed to close stderr: %v", err)
		}
	}()

	assertNoError(cmd.Start(), t, "tinydbg debug")

	scan := bufio.NewScanner(stderr)
	// wait for the debugger to start
	for scan.Scan() {
		text := scan.Text()
		t.Log(text)
		if strings.Contains(text, "API server pid = ") {
			break
		}
	}
	go func() {
		for scan.Scan() {
			t.Log(scan.Text())
			// keep pipe empty
		}
	}()

	client := rpc2.NewClient(listenAddr)
	state := <-client.Continue()

	if !state.Exited {
		t.Fatal("Program did not exit")
	}

	if err := client.Detach(true); err != nil {
		t.Logf("Failed to detach client: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

func testOutput(t *testing.T, tinydbgbin, output string, debugCmds []string) (stdout, stderr []byte) {
	var stdoutBuf, stderrBuf bytes.Buffer
	buildtestdir := filepath.Join(protest.FindFixturesDir(), "buildtest")

	c := []string{tinydbgbin, "debug", "--allow-non-terminal-interactive=true"}
	debugbin := filepath.Join(buildtestdir, "__debug_bin")
	if output != "" {
		c = append(c, "--output", output)
		if filepath.IsAbs(output) {
			debugbin = output
		} else {
			debugbin = filepath.Join(buildtestdir, output)
		}
	}
	cmd := exec.Command(c[0], c[1:]...)
	cmd.Dir = buildtestdir
	stdin, err := cmd.StdinPipe()
	assertNoError(err, t, "stdin pipe")
	defer func() {
		if err := stdin.Close(); err != nil {
			t.Logf("Failed to close stdin: %v", err)
		}
	}()

	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	assertNoError(cmd.Start(), t, "tinydbg debug with output")

	// Give delve some time to compile and write the binary.
	foundIt := false
	for wait := 0; wait < 30; wait++ {
		_, err = os.Stat(debugbin)
		if err == nil {
			foundIt = true
			break
		}

		time.Sleep(1 * time.Second)
	}
	if !foundIt {
		t.Errorf("running %q: file not created: %v", debugCmds, err)
	}

	for _, c := range debugCmds {
		if _, err := fmt.Fprintf(stdin, "%s\n", c); err != nil {
			t.Logf("Failed to write to stdin: %v", err)
		}
	}

	// ignore "tinydbg debug" command error, it returns
	// errors even after successful debug session.
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
	stdout, stderr = stdoutBuf.Bytes(), stderrBuf.Bytes()

	_, err = os.Stat(debugbin)
	if err == nil {
		// debugbin should be deleted automatically after tracee exited.
		//
		// Sometimes delve on Windows can't remove the built binary before
		// exiting and gets an "Access is denied" error when trying.
		// See: https://travis-ci.com/go-delve/delve/jobs/296325131.
		// We have added a delay to gobuild.Remove, but to avoid any test
		// flakiness, we guard against this failure here as well.
		if runtime.GOOS != "windows" {
			t.Errorf("running %q: file %v was not deleted\nstdout is %q, stderr is %q", debugCmds, debugbin, stdout, stderr)
		}
		return
	}
	if !os.IsNotExist(err) {
		t.Errorf("running %q: %v\nstdout is %q, stderr is %q", debugCmds, err, stdout, stderr)
		return
	}
	return
}

// TestOutput verifies that the debug executable is created in the correct path
// and removed after exit.
func TestOutput(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	for _, output := range []string{"__debug_bin", "myownname", filepath.Join(t.TempDir(), "absolute.path")} {
		testOutput(t, tinydbgbin, output, []string{"exit"})

		const hello = "hello world!"
		stdout, _ := testOutput(t, tinydbgbin, output, []string{"continue", "exit"})
		if !strings.Contains(string(stdout), hello) {
			t.Errorf("stdout %q should contain %q", stdout, hello)
		}
	}
}

// TestUnattendedBreakpoint tests whether tinydbg will print a message to stderr when the client that sends continue is disconnected
// or not.
func TestUnattendedBreakpoint(t *testing.T) {
	const listenAddr = "127.0.0.1:40573"

	fixturePath := filepath.Join(protest.FindFixturesDir(), "panic.go")
	cmd := exec.Command(protest.GetTinyDbgBinary(t), "debug", "--continue", "--headless", "--accept-multiclient", "--listen", listenAddr, fixturePath)
	stderr, err := cmd.StderrPipe()
	assertNoError(err, t, "stdout pipe")
	defer func() {
		if err := stderr.Close(); err != nil {
			t.Logf("Failed to close stderr: %v", err)
		}
	}()

	assertNoError(cmd.Start(), t, "start headless instance")

	scan := bufio.NewScanner(stderr)
	for scan.Scan() {
		t.Log(scan.Text())
		if strings.Contains(scan.Text(), "execution is paused because your program is panicking") {
			break
		}
	}

	// and detach from and kill the headless instance
	client := rpc2.NewClient(listenAddr)
	if err := client.Detach(true); err != nil {
		t.Fatalf("error detaching from headless instance: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

// TestContinue verifies that the debugged executable starts immediately with --continue
func TestContinue(t *testing.T) {
	const listenAddr = "127.0.0.1:40573"

	tinydbgbin := protest.GetTinyDbgBinary(t)

	buildtestdir := filepath.Join(protest.FindFixturesDir(), "buildtest")
	cmd := exec.Command(tinydbgbin, "debug", "--headless", "--continue", "--accept-multiclient", "--listen", listenAddr)
	cmd.Dir = buildtestdir
	stdout, err := cmd.StdoutPipe()
	assertNoError(err, t, "stdout pipe")
	defer func() {
		if err := stdout.Close(); err != nil {
			t.Logf("Failed to close stdout: %v", err)
		}
	}()

	assertNoError(cmd.Start(), t, "start headless instance")

	scan := bufio.NewScanner(stdout)
	// wait for the debugger to start
	for scan.Scan() {
		t.Log(scan.Text())
		if scan.Text() == "hello world!" {
			break
		}
	}

	// and detach from and kill the headless instance
	client := rpc2.NewClient(listenAddr)
	if err := client.Detach(true); err != nil {
		t.Fatalf("error detaching from headless instance: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

// TestRedirect verifies that redirecting stdin works
func TestRedirect(t *testing.T) {
	const listenAddr = "127.0.0.1:40573"

	tinydbgbin := protest.GetTinyDbgBinary(t)

	catfixture := filepath.Join(protest.FindFixturesDir(), "cat.go")
	cmd := exec.Command(tinydbgbin, "debug", "--headless", "--continue", "--accept-multiclient", "--listen", listenAddr, "-r", catfixture, catfixture)
	stdout, err := cmd.StdoutPipe()
	assertNoError(err, t, "stdout pipe")
	defer func() {
		if err := stdout.Close(); err != nil {
			t.Logf("Failed to close stdout: %v", err)
		}
	}()

	assertNoError(cmd.Start(), t, "start headless instance")

	scan := bufio.NewScanner(stdout)
	// wait for the debugger to start
	for scan.Scan() {
		t.Log(scan.Text())
		if scan.Text() == "read \"}\"" {
			break
		}
	}

	// and detach from and kill the headless instance
	client := rpc2.NewClient(listenAddr)
	if err := client.Detach(true); err != nil {
		t.Logf("Failed to detach client: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

func TestExitWhenDebugSessionInit(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	buildtestdir := filepath.Join(protest.FindFixturesDir(), "buildtest")
	exitInit := filepath.Join(protest.FindFixturesDir(), "exit.init")
	cmd := exec.Command(tinydbgbin, "--init", exitInit, "debug")
	cmd.Dir = buildtestdir
	out, err := cmd.CombinedOutput()
	t.Logf("%q %v\n", string(out), err)
	// tinydbg will exit anyway because stdin is not a tty, but it will print the
	// prompt once if the init file didn't call exit successfully.
	if strings.Contains(string(out), "(tinydbg)") {
		t.Fatal("init did not cause tinydbg to exit")
	}
}

func getMethods(pkg *types.Package, typename string) map[string]*types.Func {
	funcs := make(map[string]*types.Func)
	mset := types.NewMethodSet(types.NewPointer(pkg.Scope().Lookup(typename).Type()))
	for i := 0; i < mset.Len(); i++ {
		fn := mset.At(i).Obj().(*types.Func)
		funcs[fn.Name()] = fn
	}
	return funcs
}

func publicMethodOf(decl ast.Decl, receiver string) *ast.FuncDecl {
	fndecl, isfunc := decl.(*ast.FuncDecl)
	if !isfunc {
		return nil
	}
	if fndecl.Name.Name[0] >= 'a' && fndecl.Name.Name[0] <= 'z' {
		return nil
	}
	if fndecl.Recv == nil || len(fndecl.Recv.List) != 1 {
		return nil
	}
	starexpr, isstar := fndecl.Recv.List[0].Type.(*ast.StarExpr)
	if !isstar {
		return nil
	}
	identexpr, isident := starexpr.X.(*ast.Ident)
	if !isident || identexpr.Name != receiver {
		return nil
	}
	if fndecl.Body == nil {
		return nil
	}
	return fndecl
}

func findCallCall(fndecl *ast.FuncDecl) *ast.CallExpr {
	for _, stmt := range fndecl.Body.List {
		var x ast.Expr = nil

		switch s := stmt.(type) {
		case *ast.AssignStmt:
			if len(s.Rhs) == 1 {
				x = s.Rhs[0]
			}
		case *ast.ReturnStmt:
			if len(s.Results) == 1 {
				x = s.Results[0]
			}
		case *ast.ExprStmt:
			x = s.X
		}

		callx, iscall := x.(*ast.CallExpr)
		if !iscall {
			continue
		}
		fun, issel := callx.Fun.(*ast.SelectorExpr)
		if !issel || fun.Sel.Name != "call" {
			continue
		}
		return callx
	}
	return nil
}

func qf(*types.Package) string {
	return ""
}

func TestTypecheckRPC(t *testing.T) {
	// go1.24 AST changed, so no need to run the following check, it only works for older version of go.
	// Actually, we should provide another version for go1.24 typechecking agains RPCServer, RPCClient, and client.call.
	if goversion.VersionAfterOrEqual(runtime.Version(), 1, 24) {
		t.Skip("disabled due to export format changes")
	}

	fset := &token.FileSet{}
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedName | packages.NeedCompiledGoFiles | packages.NeedTypes,
		Fset: fset,
	}
	pkgs, err := packages.Load(cfg, "github.com/hitzhangjie/tinydbg/service/rpc2")
	if err != nil {
		t.Fatal(err)
	}
	var clientAst *ast.File
	var serverMethods map[string]*types.Func
	var info *types.Info
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.PkgPath != "github.com/hitzhangjie/tinydbg/service/rpc2" {
			return true
		}
		t.Logf("package found: %v", pkg.PkgPath)
		serverMethods = getMethods(pkg.Types, "RPCServer")
		info = pkg.TypesInfo
		for i := range pkg.Syntax {
			t.Logf("file %q", pkg.CompiledGoFiles[i])
			if strings.HasSuffix(pkg.CompiledGoFiles[i], string(os.PathSeparator)+"client.go") {
				clientAst = pkg.Syntax[i]
				break
			}
		}
		return true
	}, nil)

	errcount := 0

	for _, decl := range clientAst.Decls {
		fndecl := publicMethodOf(decl, "RPCClient")
		if fndecl == nil {
			continue
		}

		switch fndecl.Name.Name {
		case "Continue":
			// support functions
			continue
		case "SetReturnValuesLoadConfig", "Disconnect":
			// support functions
			continue
		}

		callx := findCallCall(fndecl)
		if callx == nil {
			t.Errorf("%s: could not find RPC call", fset.Position(fndecl.Pos()))
			errcount++
			continue
		}

		if len(callx.Args) != 3 {
			t.Errorf("%s: wrong number of arguments for RPC call", fset.Position(callx.Pos()))
			errcount++
			continue
		}

		arg0, arg0islit := callx.Args[0].(*ast.BasicLit)
		arg1 := callx.Args[1]
		arg2 := callx.Args[2]
		if !arg0islit || arg0.Kind != token.STRING {
			continue
		}
		name, _ := strconv.Unquote(arg0.Value)
		serverMethod := serverMethods[name]
		if serverMethod == nil {
			t.Errorf("%s: could not find RPC method %q", fset.Position(callx.Pos()), name)
			errcount++
			continue
		}

		params := serverMethod.Type().(*types.Signature).Params()

		if a, e := info.TypeOf(arg1), params.At(0).Type(); !types.AssignableTo(a, e) {
			t.Errorf("%s: wrong type of first argument %s, expected %s", fset.Position(callx.Pos()), types.TypeString(a, qf), types.TypeString(e, qf))
			errcount++
			continue
		}

		if !strings.HasSuffix(params.At(1).Type().String(), "/service.RPCCallback") {
			if a, e := info.TypeOf(arg2), params.At(1).Type(); !types.AssignableTo(a, e) {
				t.Errorf("%s: wrong type of second argument %s, expected %s", fset.Position(callx.Pos()), types.TypeString(a, qf), types.TypeString(e, qf))
				errcount++
				continue
			}
		}

		if clit, ok := arg1.(*ast.CompositeLit); ok {
			typ := params.At(0).Type()
			st := typ.Underlying().(*types.Struct)
			if len(clit.Elts) != st.NumFields() && types.TypeString(typ, qf) != "DebuggerCommand" {
				t.Errorf("%s: wrong number of fields in first argument's literal %d, expected %d", fset.Position(callx.Pos()), len(clit.Elts), st.NumFields())
				errcount++
				continue
			}
		}
	}

	if errcount > 0 {
		t.Errorf("%d errors", errcount)
	}
}

func TestTrace(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	expected := []byte("> goroutine(1): main.foo(99, 9801)\nx=99, y=9801, z=0\n>> goroutine(1): main.foo => (9900)\nz=9900")

	fixtures, err := filepath.Abs(protest.FindFixturesDir())
	assertNoError(err, t, "determine _fixture path")

	cmd := exec.Command(tinydbgbin, "trace", "--output", filepath.Join(t.TempDir(), "__debug"), filepath.Join(fixtures, "issue573.go"), "foo")
	cmd.Dir = filepath.Join(fixtures, "buildtest")

	output, err := cmd.CombinedOutput()
	assertNoError(err, t, fmt.Sprintf("get output: %s", string(output)))

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

func TestTrace2(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	expected := []byte("> goroutine(1): main.callme(2)\n>> goroutine(1): main.callme => (4)\n")

	fixtures, err := filepath.Abs(protest.FindFixturesDir())
	assertNoError(err, t, "determine _fixture path")

	cmd := exec.Command(tinydbgbin, "trace", "--output", filepath.Join(t.TempDir(), "__debug"), filepath.Join(fixtures, "traceprog.go"), "callme")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := rdr.Close(); err != nil {
			t.Logf("Failed to close rdr: %v", err)
		}
	}()

	cmd.Dir = filepath.Join(fixtures, "buildtest")

	assertNoError(cmd.Start(), t, "running trace")

	output, err := io.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}
	assertNoError(cmd.Wait(), t, "cmd.Wait()")
}

func TestTraceDirRecursion(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	expected := []byte("> goroutine(1):frame(1) main.A(5, 5)\n > goroutine(1):frame(2) main.A(4, 4)\n  > goroutine(1):frame(3) main.A(3, 3)\n   > goroutine(1):frame(4) main.A(2, 2)\n    > goroutine(1):frame(5) main.A(1, 1)\n    >> goroutine(1):frame(5) main.A => (1)\n   >> goroutine(1):frame(4) main.A => (2)\n  >> goroutine(1):frame(3) main.A => (6)\n >> goroutine(1):frame(2) main.A => (24)\n>> goroutine(1):frame(1) main.A => (120)\n")

	fixtures, err := filepath.Abs(protest.FindFixturesDir())
	assertNoError(err, t, "determine _fixture path")

	cmd := exec.Command(tinydbgbin, "trace", "--output", filepath.Join(t.TempDir(), "__debug"), filepath.Join(fixtures, "leafrec.go"), "main.A", "--follow-calls", "4")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := rdr.Close(); err != nil {
			t.Logf("Failed to close rdr: %v", err)
		}
	}()

	cmd.Dir = filepath.Join(fixtures, "buildtest")

	assertNoError(cmd.Start(), t, "running trace")
	// Parse output to ignore calls to morestack_noctxt for comparison
	scan := bufio.NewScanner(rdr)
	text := ""
	outputtext := ""
	for scan.Scan() {
		text = scan.Text()
		if !strings.Contains(text, "morestack_noctxt") {
			outputtext += text
			outputtext += "\n"
		}
	}
	output := []byte(outputtext)

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}
	assertNoError(cmd.Wait(), t, "cmd.Wait()")
}

func TestTraceMultipleGoroutines(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	// TODO(derekparker) this test has to be a bit vague to avoid flakiness.
	// I think a future improvement could be to use regexp captures to match the
	// goroutine IDs at function entry and exit.
	expected := []byte("main.callme(0, \"five\")\n")
	expected2 := []byte("main.callme => (0)\n")

	fixtures, err := filepath.Abs(protest.FindFixturesDir())
	assertNoError(err, t, "determine _fixtures path")

	cmd := exec.Command(tinydbgbin, "trace", "--output", filepath.Join(t.TempDir(), "__debug"), filepath.Join(fixtures, "goroutines-trace.go"), "callme")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := rdr.Close(); err != nil {
			t.Logf("Failed to close rdr: %v", err)
		}
	}()

	cmd.Dir = filepath.Join(fixtures, "buildtest")

	assertNoError(cmd.Start(), t, "running trace")

	output, err := io.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}
	if !bytes.Contains(output, expected2) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

func TestTracePid(t *testing.T) {
	if runtime.GOOS == "linux" {
		bs, _ := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
		if bs == nil || strings.TrimSpace(string(bs)) != "0" {
			t.Logf("can not run TestAttachDetach: %v\n", bs)
			return
		}
	}

	tinydbgbin := protest.GetTinyDbgBinary(t)

	expected := []byte("goroutine(1): main.A()\n>> goroutine(1): main.A => ()\n")

	// make process run
	fix := protest.BuildFixture("issue2023", 0)
	targetCmd := exec.Command(fix.Path)
	assertNoError(targetCmd.Start(), t, "execute issue2023")

	if targetCmd.Process == nil || targetCmd.Process.Pid == 0 {
		t.Fatal("expected target process running")
	}
	defer func() {
		if err := targetCmd.Process.Kill(); err != nil {
			t.Logf("Failed to kill process: %v", err)
		}
	}()

	// tinydbg attach the process by pid
	cmd := exec.Command(tinydbgbin, "trace", "-p", strconv.Itoa(targetCmd.Process.Pid), "main.A")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := rdr.Close(); err != nil {
			t.Logf("Failed to close rdr: %v", err)
		}
	}()

	assertNoError(cmd.Start(), t, "running trace")

	output, err := io.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}

	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}

func TestTraceBreakpointExists(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	fixtures := protest.FindFixturesDir()
	// We always set breakpoints on some runtime functions at startup, so this would return with
	// a breakpoints exists error.
	// TODO: Perhaps we shouldn't be setting these default breakpoints in trace mode, however.
	cmd := exec.Command(tinydbgbin, "trace", "--output", filepath.Join(t.TempDir(), "__debug"), filepath.Join(fixtures, "issue573.go"), "runtime.*panic")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := rdr.Close(); err != nil {
			t.Logf("Failed to close rdr: %v", err)
		}
	}()

	cmd.Dir = filepath.Join(fixtures, "buildtest")

	assertNoError(cmd.Start(), t, "running trace")

	defer func() {
		if err := cmd.Wait(); err != nil {
			t.Logf("Failed to wait for command: %v", err)
		}
	}()

	output, err := io.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if bytes.Contains(output, []byte("Breakpoint exists")) {
		t.Fatal("Breakpoint exists errors should be ignored")
	}
}

func TestTracePrintStack(t *testing.T) {
	tinydbgbin := protest.GetTinyDbgBinary(t)

	fixtures := protest.FindFixturesDir()
	cmd := exec.Command(tinydbgbin, "trace", "--output", filepath.Join(t.TempDir(), "__debug"), "--stack", "2", filepath.Join(fixtures, "issue573.go"), "foo")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := rdr.Close(); err != nil {
			t.Logf("Failed to close rdr: %v", err)
		}
	}()

	cmd.Dir = filepath.Join(fixtures, "buildtest")
	assertNoError(cmd.Start(), t, "running trace")

	defer func() {
		if err := cmd.Wait(); err != nil {
			t.Logf("Failed to wait for command: %v", err)
		}
	}()

	output, err := io.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if !bytes.Contains(output, []byte("Stack:")) && !bytes.Contains(output, []byte("main.main")) {
		t.Fatal("stacktrace not printed")
	}
}

func TestUnixDomainSocket(t *testing.T) {
	tmpdir := os.TempDir()
	if tmpdir == "" {
		return
	}

	listenPath := filepath.Join(tmpdir, "delve_test")

	tinydbgbin := protest.GetTinyDbgBinary(t)

	fixtures := protest.FindFixturesDir()

	buildtestdir := filepath.Join(fixtures, "buildtest")

	cmd := exec.Command(tinydbgbin, "debug", "--headless=true", "--listen=unix:"+listenPath, "--log", "--log-output=debugger,rpc")
	cmd.Dir = buildtestdir
	stderr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer func() {
		if err := stderr.Close(); err != nil {
			t.Logf("Failed to close stderr: %v", err)
		}
	}()

	assertNoError(cmd.Start(), t, "tinydbg debug")

	scan := bufio.NewScanner(stderr)
	// wait for the debugger to start
	for scan.Scan() {
		text := scan.Text()
		t.Log(text)
		if strings.Contains(text, "API server pid = ") {
			break
		}
	}
	go func() {
		for scan.Scan() {
			t.Log(scan.Text())
			// keep pipe empty
		}
	}()

	conn, err := net.Dial("unix", listenPath)
	assertNoError(err, t, "dialing")

	client := rpc2.NewClientFromConn(conn)
	state := <-client.Continue()

	if !state.Exited {
		t.Fatal("Program did not exit")
	}

	if err := client.Detach(true); err != nil {
		t.Logf("Failed to detach client: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Logf("Failed to wait for command: %v", err)
	}
}
