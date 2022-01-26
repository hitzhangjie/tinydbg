package debugger

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/creack/pty"
	"github.com/hitzhangjie/dlv/pkg/gobuild"
	protest "github.com/hitzhangjie/dlv/pkg/proc/test"
)

func TestDebugger_LaunchWithTTY(t *testing.T) {
	if os.Getenv("CI") == "true" {
		if _, err := exec.LookPath("lsof"); err != nil {
			t.Skip("skipping test in CI, system does not contain lsof")
		}
	}
	// Ensure no env meddling is leftover from previous tests.
	os.Setenv("GOOS", runtime.GOOS)
	os.Setenv("GOARCH", runtime.GOARCH)

	p, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	defer tty.Close()

	fixturesDir := protest.FindFixturesDir()
	buildtestdir := filepath.Join(fixturesDir, "buildtest")
	debugname := "debugtty"
	exepath := filepath.Join(buildtestdir, debugname)
	if err := gobuild.GoBuild(debugname, []string{buildtestdir}); err != nil {
		t.Fatalf("go build error %v", err)
	}
	defer os.Remove(exepath)
	var backend string
	protest.DefaultTestBackend(&backend)
	conf := &Config{}
	pArgs := []string{exepath}
	d, err := New(conf, pArgs)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("lsof", "-p", fmt.Sprintf("%d", d.ProcessPid()))
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(result, []byte(tty.Name())) {
		t.Fatal("process open file list does not contain expected tty")
	}
}
