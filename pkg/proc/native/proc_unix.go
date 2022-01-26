package native

import (
	"os"
	"os/exec"
)

func attachProcessToTTY(process *exec.Cmd) error {
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	process.SysProcAttr.Setpgid = false
	process.SysProcAttr.Setsid = true
	process.SysProcAttr.Setctty = true

	return nil
}
