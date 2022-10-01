package main

import (
	"os"

	"github.com/sirupsen/logrus"
	_ "github.com/spf13/cobra/doc"

	"github.com/hitzhangjie/dlv/cmds"
)

const cgoCflagsEnv = "CGO_CFLAGS"

func main() {
	if v := os.Getenv(cgoCflagsEnv); v == "" {
		os.Setenv(cgoCflagsEnv, "-O0 -g")
	} else {
		log := logrus.WithFields(logrus.Fields{"layer": "dlv"})
		log.Warnln("CGO_CFLAGS set, cgo code may be optimized")
	}
	cmds.New().Execute()
}
