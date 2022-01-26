package cmds

import "github.com/spf13/cobra"

var rootCommand = &cobra.Command{
	Use:   "dlv",
	Short: "Delve is a debugger for the Go programming language.",
	Long: `Delve is a source level debugger for Go programs.

Delve enables you to interact with your program by controlling the execution of the process,
evaluating variables, and providing information of thread / goroutine state, CPU register state and more.

The goal of this tool is to provide a simple yet powerful interface for debugging Go programs.

Pass flags to the program you are debugging using "--",  for example:
"dlv exec ./hello -- server --config conf/config.toml"`,
	DisableAutoGenTag: true,
}

func init() {
	rootCommand.PersistentFlags().StringVarP(&addr, "listen", "l", "127.0.0.1:0", "Debugging server listen address.")
	rootCommand.PersistentFlags().BoolVarP(&headless, "headless", "", false, "Run debug server only, in headless mode.")
	rootCommand.PersistentFlags().BoolVarP(&acceptMulti, "accept-multiclient", "", false, "Allows a headless server to accept multiple client connections.")
	rootCommand.PersistentFlags().StringVar(&workingDir, "wd", "", "Working directory for running the program.")
	rootCommand.PersistentFlags().BoolVarP(&checkGoVersion, "check-go-version", "", true, "Exits if the version of Go in use is not compatible (too old or too new) with the version of Delve.")
	rootCommand.PersistentFlags().BoolVar(&disableASLR, "disable-aslr", false, "Disables address space randomization")
}
