package cmds

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/gobuild"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/service/debugger"
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
	Run: debugCmdRun,
}

func init() {
	debugCommand.Flags().String("output", "./__debug_bin", "Output path for the binary.")
	debugCommand.Flags().BoolVar(&continueOnStart, "continue", false, "Continue the debugged process on start.")
	debugCommand.Flags().StringVar(&tty, "tty", "", "TTY to use for the target program")
	rootCommand.AddCommand(debugCommand)
}

func debugCmdRun(cmd *cobra.Command, args []string) {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

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
	debugname, err := filepath.Abs(cmd.Flag("output").Value.String())
	if err != nil {
		log.Error("%v", err)
		return "", false
	}

	if isTest {
		err = gobuild.GoTestBuild(debugname, args, buildFlags)
	} else {
		err = gobuild.GoBuild(debugname, args, buildFlags)
	}
	if err != nil {
		log.Error("%v", err)
		return "", false
	}
	return debugname, true
}
