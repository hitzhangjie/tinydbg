package native

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	sys "golang.org/x/sys/unix"

	"github.com/hitzhangjie/dlv/pkg/proc"
	"github.com/hitzhangjie/dlv/pkg/proc/linutil"
)

// Launch creates and begins debugging a new process. First entry in
// `cmd` is the program to run, and then rest are the arguments
// to be supplied to that process. `wd` is working directory of the program.
// If the DWARF information cannot be found in the binary, Delve will look
// for external debug files in the directories passed in.
func Launch(cmd []string, wd string, flags proc.LaunchFlags) (*proc.Target, error) {
	var (
		process *exec.Cmd
		err     error
	)

	foreground := flags&proc.LaunchForeground != 0

	dbp := newProcess(0)
	defer func() {
		if err != nil && dbp.pid != 0 {
			_ = dbp.Detach(true)
		}
	}()
	dbp.execPtraceFunc(func() {
		if flags&proc.LaunchDisableASLR != 0 {
			oldPersonality, _, err := syscall.Syscall(sys.SYS_PERSONALITY, personalityGetPersonality, 0, 0)
			if err == syscall.Errno(0) {
				newPersonality := oldPersonality | _ADDR_NO_RANDOMIZE
				syscall.Syscall(sys.SYS_PERSONALITY, newPersonality, 0, 0)
				defer syscall.Syscall(sys.SYS_PERSONALITY, oldPersonality, 0, 0)
			}
		}

		process = exec.Command(cmd[0])
		process.Args = cmd
		process.Stdin = os.Stdin
		process.Stdout = os.Stdout
		process.Stderr = os.Stderr
		process.SysProcAttr = &syscall.SysProcAttr{
			Ptrace:     true,
			Setpgid:    true,
			Foreground: foreground,
		}
		if foreground {
			signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN)
		}
		if err := attachProcessToTTY(process); err != nil {
			return
		}
		if wd != "" {
			process.Dir = wd
		}
		err = process.Start()
	})
	if err != nil {
		return nil, err
	}
	dbp.pid = process.Process.Pid
	dbp.childProcess = true
	_, _, err = dbp.wait(process.Process.Pid, 0)
	if err != nil {
		return nil, fmt.Errorf("waiting for target execve failed: %s", err)
	}
	tgt, err := dbp.initialize(cmd[0])
	if err != nil {
		return nil, err
	}
	return tgt, nil
}

// Attach to an existing process with the given PID.
// Usually, go compiler/linker generate DWARF in the binary on Linux.
// While on Darwin, DWARF may be generated into separate files.
// here, we only care about the 1st occasion.
func Attach(pid int) (*proc.Target, error) {
	dbp := newProcess(pid)

	var err error
	dbp.execPtraceFunc(func() { err = ptraceAttach(dbp.pid) })
	if err != nil {
		return nil, err
	}
	_, _, err = dbp.wait(dbp.pid, 0)
	if err != nil {
		return nil, err
	}

	tgt, err := dbp.initialize(findExecutable("", dbp.pid))
	if err != nil {
		_ = dbp.Detach(false)
		return nil, err
	}

	// ElfUpdateSharedObjects can only be done after we initialize because it
	// needs an initialized BinaryInfo object to work.
	err = linutil.ElfUpdateSharedObjects(dbp)
	if err != nil {
		return nil, err
	}
	return tgt, nil
}

func findExecutable(path string, pid int) string {
	if path == "" {
		path = fmt.Sprintf("/proc/%d/exe", pid)
	}
	return path
}

func attachProcessToTTY(process *exec.Cmd) error {
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	process.SysProcAttr.Setpgid = false
	process.SysProcAttr.Setsid = true
	process.SysProcAttr.Setctty = true

	return nil
}
