package proc_test

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/hitzhangjie/dlv/pkg/proc"
	"github.com/hitzhangjie/dlv/pkg/proc/native"
	proctest "github.com/hitzhangjie/dlv/pkg/proc/test"
)

type errIssue419 struct {
	pid int
	err error
}

func (npe errIssue419) Error() string {
	return fmt.Sprintf("Pid is zero or negative: %d", npe.pid)
}

func TestSignalDeath(t *testing.T) {
	if testBackend != "native" || runtime.GOOS != "linux" {
		t.Skip("skipped on non-linux non-native backends")
	}
	var buildFlags proctest.BuildFlags
	if buildMode == "pie" {
		buildFlags |= proctest.BuildModePIE
	}
	fixture := proctest.BuildFixture("loopprog", buildFlags)
	cmd := exec.Command(fixture.Path)
	stdout, err := cmd.StdoutPipe()
	assertNoError(err, t, "StdoutPipe")
	cmd.Stderr = os.Stderr
	assertNoError(cmd.Start(), t, "starting fixture")
	p, err := native.Attach(cmd.Process.Pid)
	assertNoError(err, t, "Attach")
	stdout.Close() // target will receive SIGPIPE later on
	err = p.Continue()
	t.Logf("error is %v", err)
	exitErr, isexited := err.(proc.ErrProcessExited)
	if !isexited {
		t.Fatal("did not exit")
	}
	if exitErr.Status != -int(unix.SIGPIPE) {
		t.Fatalf("expected SIGPIPE got %d\n", exitErr.Status)
	}
}
