// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra/doc"

	"github.com/hitzhangjie/dlv/pkg/cmds"
)

const defaultUsageDir = "./_docs/usage"

func main() {
	usageDir := defaultUsageDir
	if len(os.Args) > 1 {
		usageDir = os.Args[1]
	}
	root := cmds.New(true)
	doc.GenMarkdownTree(root, usageDir)
	// GenMarkdownTree ignores additional help topic commands, so we have to do this manually
	for _, cmd := range root.Commands() {
		if cmd.Run == nil {
			doc.GenMarkdownTree(cmd, usageDir)
		}
	}
	fh, err := os.OpenFile(filepath.Join(usageDir, "dlv.md"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		log.Fatalf("appending to dlv.md: %v", err)
	}
	defer fh.Close()
	fmt.Fprintln(fh, "* [dlv log](dlv_log.md)\t - Help about logging flags")
	fmt.Fprintln(fh, "* [dlv backend](dlv_backend.md)\t - Help about the `--backend` flag")
}
