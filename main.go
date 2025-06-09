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
var Build string = "v0.0.1"

func main() {
	// current demo only supports linux/amd64
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {

		fmt.Fprintf(os.Stderr, "WARNING: tinydbg only supports linux/amd64")
		os.Exit(1)
	}

	if Build != "" {
		version.DelveVersion.Build = Build
	}

	const cgoCflagsEnv = "CGO_CFLAGS"
	if os.Getenv(cgoCflagsEnv) == "" {
		os.Setenv(cgoCflagsEnv, "-O0 -g")
	} else {
		logflags.WriteCgoFlagsWarning()
	}

	cmds.New().Execute()
}
