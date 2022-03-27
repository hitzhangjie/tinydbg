package native

import (
	"fmt"

	sys "golang.org/x/sys/unix"

	"github.com/hitzhangjie/dlv/pkg/proc"
)

type waitStatus sys.WaitStatus

// osSpecificDetails hold Linux specific
// process details.
type osSpecificDetails struct {
	delayedSignal       int
	running             bool
	setbp               bool
	phantomBreakpointPC uint64
}

func (t *nativeThread) stop() (err error) {
	err = sys.Tgkill(t.dbp.pid, t.ID, sys.SIGSTOP)
	if err != nil {
		err = fmt.Errorf("stop err %s on thread %d", err, t.ID)
		return
	}
	return
}

func (t *nativeThread) resume() error {
	sig := t.os.delayedSignal
	t.os.delayedSignal = 0
	return t.resumeWithSig(sig)
}

func (t *nativeThread) resumeWithSig(sig int) (err error) {
	t.os.running = true
	t.dbp.execPtraceFunc(func() { err = ptraceCont(t.ID, sig) })
	return
}

func (t *nativeThread) singleStep() (err error) {
	sig := 0
	for {
		t.dbp.execPtraceFunc(func() { err = ptraceSingleStep(t.ID, sig) })
		sig = 0
		if err != nil {
			return err
		}
		wpid, status, err := t.dbp.waitFast(t.ID)
		if err != nil {
			return err
		}
		if (status == nil || status.Exited()) && wpid == t.dbp.pid {
			t.dbp.postExit()
			rs := 0
			if status != nil {
				rs = status.ExitStatus()
			}
			return proc.ErrProcessExited{Pid: t.dbp.pid, Status: rs}
		}
		if wpid == t.ID {
			switch s := status.StopSignal(); s {
			case sys.SIGTRAP:
				return nil
			case sys.SIGSTOP:
				// delayed SIGSTOP, ignore it
			case sys.SIGILL, sys.SIGBUS, sys.SIGFPE, sys.SIGSEGV, sys.SIGSTKFLT:
				// propagate signals that can have been caused by the current instruction
				sig = int(s)
			default:
				// delay propagation of all other signals
				t.os.delayedSignal = int(s)
			}
		}
	}
}
