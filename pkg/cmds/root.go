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
	buildFlagsDefault := ""

	rootCommand.PersistentFlags().StringVarP(&addr, "listen", "l", "127.0.0.1:0", "Debugging server listen address.")

	rootCommand.PersistentFlags().BoolVarP(&log, "log", "", false, "Enable debugging server logging.")
	rootCommand.PersistentFlags().StringVarP(&logOutput, "log-output", "", "", `Comma separated list of components that should produce debug output (see 'dlv help log')`)
	rootCommand.PersistentFlags().StringVarP(&logDest, "log-dest", "", "", "Writes logs to the specified file or file descriptor (see 'dlv help log').")

	rootCommand.PersistentFlags().BoolVarP(&headless, "headless", "", false, "Run debug server only, in headless mode.")
	rootCommand.PersistentFlags().BoolVarP(&acceptMulti, "accept-multiclient", "", false, "Allows a headless server to accept multiple client connections.")
	rootCommand.PersistentFlags().StringVar(&initFile, "init", "", "Init file, executed by the terminal client.")
	rootCommand.PersistentFlags().StringVar(&buildFlags, "build-flags", buildFlagsDefault, "Build flags, to be passed to the compiler. For example: --build-flags=\"-tags=integration -mod=vendor -cover -v\"")
	rootCommand.PersistentFlags().StringVar(&workingDir, "wd", "", "Working directory for running the program.")
	rootCommand.PersistentFlags().BoolVarP(&checkGoVersion, "check-go-version", "", true, "Exits if the version of Go in use is not compatible (too old or too new) with the version of Delve.")
	rootCommand.PersistentFlags().BoolVarP(&checkLocalConnUser, "only-same-user", "", true, "Only connections from the same user that started this instance of Delve are allowed to connect.")
	rootCommand.PersistentFlags().StringVar(&backend, "backend", "default", `Backend selection (see 'dlv help backend').`)
	rootCommand.PersistentFlags().StringArrayVarP(&redirects, "redirect", "r", []string{}, "Specifies redirect rules for target process (see 'dlv help redirect')")
	rootCommand.PersistentFlags().BoolVar(&allowNonTerminalInteractive, "allow-non-terminal-interactive", false, "Allows interactive sessions of Delve that don't have a terminal as stdin, stdout and stderr")
	rootCommand.PersistentFlags().BoolVar(&disableASLR, "disable-aslr", false, "Disables address space randomization")
}

func init() {

}
