// Package gobuild provides utilities for building programs and tests
// for the debugging session.
package gobuild

import (
	"os"
	"os/exec"

	"github.com/hitzhangjie/dlv/pkg/goversion"
	"github.com/hitzhangjie/dlv/pkg/log"
)

// Remove the file at path and issue a warning to stderr if this fails.
// This can be used to remove the temporary binary generated for the session.
func Remove(path string) {
	if err := os.Remove(path); err != nil {
		log.Error("could not remove %v: %v", path, err)
	}
}

// optflags generates default build flags to turn off optimization and inlining.
func optflags(args []string) []string {
	// after go1.9 building with -gcflags='-N -l' and -a simultaneously works.
	// after go1.10 specifying -a is unnecessary because of the new caching strategy,
	// but we should pass -gcflags=all=-N -l to have it applied to all packages
	// see https://github.com/golang/go/commit/5993251c015dfa1e905bdf44bdb41572387edf90

	ver, _ := goversion.Installed()
	switch {
	case ver.Major < 0 || ver.AfterOrEqual(goversion.GoVersion{Major: 1, Minor: 10, Rev: -1}):
		args = append(args, "-gcflags", "all=-N -l")
	case ver.AfterOrEqual(goversion.GoVersion{Major: 1, Minor: 9, Rev: -1}):
		args = append(args, "-gcflags", "-N -l", "-a")
	default:
		args = append(args, "-gcflags", "-N -l")
	}
	return args
}

// GoBuild builds non-test files in 'pkgs' and writes the output at 'debugname'.
func GoBuild(debugname string, pkgs []string) error {
	args := goBuildArgs(debugname, pkgs, false)
	return goCommandRun("build", args...)
}

// GoTestBuild builds test files 'pkgs' and writes the output at 'debugname'.
func GoTestBuild(debugname string, pkgs []string) error {
	args := goBuildArgs(debugname, pkgs, true)
	return goCommandRun("test", args...)
}

func goBuildArgs(debugname string, pkgs []string, isTest bool) []string {
	args := []string{"-o", debugname}
	if isTest {
		args = append([]string{"-c"}, args...)
	}
	args = optflags(args)
	args = append(args, pkgs...)
	return args
}

func goCommandRun(command string, args ...string) error {
	cmdArgs := []string{command}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Stderr = os.Stdout
	cmd.Stdout = os.Stderr
	return cmd.Run()
}
