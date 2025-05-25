package main

import (
	"os"

	"github.com/hitzhangjie/tinydbg/cmds"
	"github.com/hitzhangjie/tinydbg/pkg/logflags"
	"github.com/hitzhangjie/tinydbg/pkg/version"
)

// Build is the git sha of this binaries build.
var Build string

func main() {
	if Build != "" {
		version.DelveVersion.Build = Build
	}

	const cgoCflagsEnv = "CGO_CFLAGS"
	if os.Getenv(cgoCflagsEnv) == "" {
		os.Setenv(cgoCflagsEnv, "-O0 -g")
	} else {
		logflags.WriteCgoFlagsWarning()
	}

	cmds.New(false).Execute()
}
