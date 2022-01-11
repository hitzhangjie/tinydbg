package main

import (
	"os"

	"github.com/sirupsen/logrus"
	_ "github.com/spf13/cobra/doc"

	"github.com/hitzhangjie/dlv/pkg/cmds"
)

func main() {
	const cgoCflagsEnv = "CGO_CFLAGS"
	if os.Getenv(cgoCflagsEnv) == "" {
		os.Setenv(cgoCflagsEnv, "-O0 -g")
	} else {
		logrus.WithFields(logrus.Fields{"layer": "dlv"}).Warnln("CGO_CFLAGS already set, Cgo code could be optimized.")
	}
	cmds.New(false).Execute()
}
