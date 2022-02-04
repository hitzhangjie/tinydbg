package main_test

import (
	"bytes"
	"flag"
	"go/ast"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/hitzhangjie/dlv/pkg/goversion"
	proctest "github.com/hitzhangjie/dlv/pkg/proc/test"
)

var ldFlags string

func init() {
	ldFlags = os.Getenv("CGO_LDFLAGS")
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(proctest.RunTestsWithFixtures(m))
}

func assertNoError(err error, t testing.TB, s string) {
	t.Helper()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fname := filepath.Base(file)
		t.Fatalf("failed assertion at %s:%d: %s - %s\n", fname, line, s, err)
	}
}

func projectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	gopaths := strings.FieldsFunc(os.Getenv("GOPATH"), func(r rune) bool { return r == os.PathListSeparator })
	for _, curpath := range gopaths {
		// Detects "gopath mode" when GOPATH contains several paths ex. "d:\\dir\\gopath;f:\\dir\\gopath2"
		if strings.Contains(wd, curpath) {
			return filepath.Join(curpath, "src", "github.com", "go-delve", "delve")
		}
	}
	val, err := exec.Command("go", "list", "-mod=", "-m", "-f", "{{ .Dir }}").Output()
	if err != nil {
		panic(err) // the Go tool was tested to work earlier
	}
	return strings.TrimSuffix(string(val), "\n")
}

func getDlvBin(t *testing.T) (string, string) {
	// In case this was set in the environment
	// from getDlvBinEBPF lets clear it here so
	// we can ensure we don't get build errors
	// depending on the test ordering.
	os.Setenv("CGO_LDFLAGS", ldFlags)
	return getDlvBinInternal(t)
}

func getDlvBinEBPF(t *testing.T) (string, string) {
	os.Setenv("CGO_LDFLAGS", "/usr/lib/libbpf.a")
	return getDlvBinInternal(t, "-tags", "ebpf")
}

func getDlvBinInternal(t *testing.T, goflags ...string) (string, string) {
	tmpdir, err := ioutil.TempDir("", "TestDlv")
	if err != nil {
		t.Fatal(err)
	}

	dlvbin := filepath.Join(tmpdir, "dlv")
	args := append([]string{"build", "-o", dlvbin}, goflags...)
	args = append(args, "github.com/hitzhangjie/dlv")

	out, err := exec.Command("go", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("go build -o %v github.com/hitzhangjie/dlv/cmd/dlv: %v\n%s", dlvbin, err, string(out))
	}

	return dlvbin, tmpdir
}

func getMethods(pkg *types.Package, typename string) map[string]*types.Func {
	r := make(map[string]*types.Func)
	mset := types.NewMethodSet(types.NewPointer(pkg.Scope().Lookup(typename).Type()))
	for i := 0; i < mset.Len(); i++ {
		fn := mset.At(i).Obj().(*types.Func)
		r[fn.Name()] = fn
	}
	return r
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
	fset := &token.FileSet{}
	cfg := &packages.Config{
		Mode: packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedName | packages.NeedCompiledGoFiles | packages.NeedTypes,
		Fset: fset,
	}
	pkgs, err := packages.Load(cfg, "github.com/hitzhangjie/dlv/service/rpcv2")
	if err != nil {
		t.Fatal(err)
	}
	var clientAst *ast.File
	var serverMethods map[string]*types.Func
	var info *types.Info
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg.PkgPath != "github.com/hitzhangjie/dlv/service/rpcv2" {
			return true
		}
		t.Logf("package found: %v", pkg.PkgPath)
		serverMethods = getMethods(pkg.Types, "RPCServer")
		info = pkg.TypesInfo
		for i := range pkg.Syntax {
			t.Logf("file %q", pkg.CompiledGoFiles[i])
			if strings.HasSuffix(pkg.CompiledGoFiles[i], "client.go") {
				clientAst = pkg.Syntax[i]
				break
			}
		}
		return true
	}, nil)

	errcount := 0

	for _, decl := range clientAst.Decls {
		fndecl := publicMethodOf(decl, "rpcClient")
		if fndecl == nil {
			continue
		}

		switch fndecl.Name.Name {
		case "Continue", "Rewind":
			// wrappers over continueDir
			continue
		case "SetReturnValuesLoadConfig", "Disconnect":
			// support functions
			continue
		}

		if fndecl.Name.Name == "Continue" || fndecl.Name.Name == "Rewind" || fndecl.Name.Name == "DirectionCongruentContinue" {
			// using continueDir
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

func TestTracePid(t *testing.T) {
	if runtime.GOOS == "linux" {
		bs, _ := ioutil.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
		if bs == nil || strings.TrimSpace(string(bs)) != "0" {
			t.Logf("can not run TestAttachDetach: %v\n", bs)
			return
		}
	}

	dlvbin, tmpdir := getDlvBin(t)
	defer os.RemoveAll(tmpdir)

	expected := []byte("goroutine(1): main.A() => ()\n")

	// make process run
	fix := proctest.BuildFixture("issue2023", 0)
	targetCmd := exec.Command(fix.Path)
	assertNoError(targetCmd.Start(), t, "execute issue2023")

	if targetCmd.Process == nil || targetCmd.Process.Pid == 0 {
		t.Fatal("expected target process runninng")
	}
	defer targetCmd.Process.Kill()

	// dlv attach the process by pid
	cmd := exec.Command(dlvbin, "trace", "-p", strconv.Itoa(targetCmd.Process.Pid), "main.A")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer rdr.Close()

	assertNoError(cmd.Start(), t, "running trace")

	output, err := ioutil.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}

	cmd.Wait()
}

func TestTraceEBPF(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("cannot run test in CI, requires kernel compiled with btf support")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("not implemented on non linux/amd64 systems")
	}
	if !goversion.VersionAfterOrEqual(runtime.Version(), 1, 16) {
		t.Skip("requires at least Go 1.16 to run test")
	}
	usr, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if usr.Uid != "0" {
		t.Skip("test must be run as root")
	}

	dlvbin, tmpdir := getDlvBinEBPF(t)
	defer os.RemoveAll(tmpdir)

	expected := []byte("> (1) main.foo(99, 9801)\n=> \"9900\"")

	fixtures := proctest.FindFixturesDir()
	cmd := exec.Command(dlvbin, "trace", "--ebpf", "--output", filepath.Join(tmpdir, "__debug"), filepath.Join(fixtures, "issue573.go"), "foo")
	rdr, err := cmd.StderrPipe()
	assertNoError(err, t, "stderr pipe")
	defer rdr.Close()

	assertNoError(cmd.Start(), t, "running trace")

	output, err := ioutil.ReadAll(rdr)
	assertNoError(err, t, "ReadAll")

	if !bytes.Contains(output, expected) {
		t.Fatalf("expected:\n%s\ngot:\n%s", string(expected), string(output))
	}
	cmd.Wait()
}
