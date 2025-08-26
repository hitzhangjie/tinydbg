package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/hitzhangjie/tinydbg/cmds"
	"github.com/hitzhangjie/tinydbg/pkg/logflags"
	"github.com/hitzhangjie/tinydbg/pkg/version"
)

// Build is the git sha of this binaries build.
//
// note: we can pass -ldflags "-X main.Build=v0.0.1" to set the build version when building.
var Build string = "v0.0.1"

func main() {
	// current demo only supports linux/amd64
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		fmt.Fprintf(os.Stderr, "WARNING: tinydbg only supports linux/amd64")
		os.Exit(1)
	}

	// this build version will be dumped when generating the core file
	if Build != "" {
		version.DelveVersion.Build = Build
	}

	// if cgo used, pass "-O0 -g" to disable optimization and enable debugging symbols
	const cgoCflagsEnv = "CGO_CFLAGS"
	if os.Getenv(cgoCflagsEnv) == "" {
		_ = os.Setenv(cgoCflagsEnv, "-O0 -g")
	} else {
		logflags.WriteCgoFlagsWarning()
	}

	if err := cmds.New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "execution failed: %v\n", err)
		os.Exit(1)
	}
}
