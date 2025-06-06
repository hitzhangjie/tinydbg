package cmds

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hitzhangjie/tinydbg/pkg/gobuild"
	"github.com/hitzhangjie/tinydbg/service/debugger"
	"github.com/spf13/cobra"
)

// 'debug' subcommand.
var debugCommand = &cobra.Command{
	Use:   "debug [package]",
	Short: "Compile and begin debugging main package in current directory, or the package specified.",
	Long: `Compiles your program with optimizations disabled, starts and attaches to it.

By default, with no arguments, Delve will compile the 'main' package in the
current directory, and begin to debug it. Alternatively you can specify a
package name and Delve will compile that package instead, and begin a new debug
session.`,
	Run:               debugCmd,
	ValidArgsFunction: cobra.NoFileCompletions,
}

func init() {
	debugCommand.Flags().String("output", "", "Output path for the binary.")
	must(debugCommand.MarkFlagFilename("output"))
	debugCommand.Flags().BoolVar(&continueOnStart, "continue", false, "Continue the debugged process on start.")
	debugCommand.Flags().StringVar(&tty, "tty", "", "TTY to use for the target program")
	must(debugCommand.MarkFlagFilename("tty"))
}

func debugCmd(cmd *cobra.Command, args []string) {
	status := func() int {
		dlvArgs, targetArgs := splitArgs(cmd, args)
		debugname, ok := buildBinary(cmd, dlvArgs, false)
		if !ok {
			return 1
		}
		defer gobuild.Remove(debugname)
		processArgs := append([]string{debugname}, targetArgs...)
		return execute(0, processArgs, conf, "", debugger.ExecutingGeneratedFile, dlvArgs, buildFlags)
	}()
	os.Exit(status)
}

func buildBinary(cmd *cobra.Command, args []string, isTest bool) (string, bool) {
	if isTest {
		panic("not supported mode: test")
	}
	var outputFlag = cmd.Flag("output").Value.String()
	var debugname string
	var err error

	if outputFlag == "" {
		if isTest {
			debugname = gobuild.DefaultDebugBinaryPath("debug.test")
		} else {
			debugname = gobuild.DefaultDebugBinaryPath("__debug_bin")
		}
	} else {
		debugname, err = filepath.Abs(outputFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return "", false
		}
	}

	err = gobuild.GoBuild(debugname, args, buildFlags)
	if err != nil {
		if outputFlag == "" {
			gobuild.Remove(debugname)
		}
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return "", false
	}
	return debugname, true
}
