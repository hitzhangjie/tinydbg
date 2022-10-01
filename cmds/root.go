package cmds

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/pkg/terminal"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
)

var (
	// debugger mode
	headless        bool // whether to run without terminal
	continueOnStart bool // whether to continue the process on startup
	acceptMulti     bool // whether allows multiple clients to connect to the same server

	// debugger settings
	addr        string // the debugging server listen address
	workingDir  string // the working directory for running the program
	disableASLR bool   // whether disables ASLR

	// checkGoVersion is true if the debugger should check the version of Go
	// used to compile the executable and refuse to work on incompatible
	// versions.
	checkGoVersion bool

	// trace settings
	traceAttachPid  int
	traceExecFile   string
	traceTestBinary bool
	traceStackDepth int
	traceUseEBPF    bool

	// logging level
	verbose bool
)

// New returns an initialized command tree.
func New() *cobra.Command {
	return rootCommand
}

var rootCommand = &cobra.Command{
	Use:   "dlv",
	Short: "Delve is a debugger for the Go programming language.",
	Long: `Delve is a source level debugger for Go programs.

Delve enables you to interact with your program by controlling the execution of the process,
evaluating variables, and providing information of thread / goroutine state, CPU register state and more.

The goal of this tool is to provide a simple yet powerful interface for debugging Go programs.

Pass flags to the program you are debugging using "--",  for example:
"dlv exec ./hello -- [args] [--config conf/config.toml]"`,
	DisableAutoGenTag: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetFlags(log.LVerbose)
		}
	},
}

func init() {
	rootCommand.PersistentFlags().StringVarP(&addr, "listen", "l", "127.0.0.1:0", "Debugging server listen address.")
	rootCommand.PersistentFlags().BoolVarP(&headless, "headless", "", false, "Run debug server only, in headless mode.")
	rootCommand.PersistentFlags().BoolVarP(&acceptMulti, "accept-multiclient", "", false, "Allows a headless server to accept multiple client connections.")
	rootCommand.PersistentFlags().StringVar(&workingDir, "wd", "", "Working directory for running the program.")
	rootCommand.PersistentFlags().BoolVarP(&checkGoVersion, "check-go-version", "", true, "Exits if the version of Go in use is not compatible (too old or too new) with the version of Delve.")
	rootCommand.PersistentFlags().BoolVar(&disableASLR, "disable-aslr", false, "Disables address space randomization")
	rootCommand.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose logging message")
}

func execute(attachPid int, processArgs []string, coreFile string, kind debugger.ExecuteKind, dlvArgs []string) int {
	var err error

	if continueOnStart {
		if !headless {
			fmt.Fprint(os.Stderr, "Error: --continue only works with --headless\n")
			return 1
		}
		if !acceptMulti {
			fmt.Fprint(os.Stderr, "Error: --continue requires --accept-multiclient\n")
			return 1
		}
	}

	if !headless && acceptMulti {
		fmt.Fprint(os.Stderr, "Warning accept-multi: ignored\n")
		// acceptMulti won't work in normal (non-headless) mode because we always
		// call server.Stop after the terminal client exits.
		acceptMulti = false
	}

	if workingDir == "" {
		workingDir = "."
	}

	var listener net.Listener
	var clientConn net.Conn

	// Make a TCP listener
	if headless {
		listener, err = net.Listen("tcp", addr)
	} else {
		listener, clientConn = service.ListenerPipe()
	}
	if err != nil {
		log.Error("couldn't start listener: %s", err)
		return 1
	}
	defer listener.Close()

	disconnectChan := make(chan struct{})

	// Create and start a debugger server
	var server service.Server
	server = service.NewServer(&service.Config{
		Listener:       listener,
		ProcessArgs:    processArgs,
		AcceptMulti:    acceptMulti,
		DisconnectChan: disconnectChan,
		DebuggerConfig: debugger.Config{
			AttachPid:      attachPid,
			WorkingDir:     workingDir,
			CoreFile:       coreFile,
			Foreground:     headless,
			Packages:       dlvArgs,
			ExecuteKind:    kind,
			CheckGoVersion: checkGoVersion,
			DisableASLR:    disableASLR,
		},
	})

	if err := server.Run(); err != nil {
		if err == api.ErrNotExecutable {
			switch kind {
			case debugger.ExecutingGeneratedFile:
				fmt.Fprintln(os.Stderr, "Can not debug non-main package")
				return 1
			case debugger.ExecutingExistingFile:
				log.Error("%s is not executable", processArgs[0])
				return 1
			default:
			}
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var status int
	if headless {
		if continueOnStart {
			client := service.NewClient(listener.Addr().String())
			client.Disconnect(true) // true = continue after disconnect
		}
		waitForDisconnectSignal(disconnectChan)
		err = server.Stop()
		if err != nil {
			log.Error("%v", err)
		}

		return status
	}

	return connect(listener.Addr().String(), clientConn, kind)
}

// waitForDisconnectSignal is a blocking function that waits for either
// a SIGINT (Ctrl-C) or SIGTERM (kill -15) OS signal or for disconnectChan
// to be closed by the server when the client disconnects.
// Note that in headless mode, the debugged process is foregrounded
// (to have control of the tty for debugging interactive programs),
// so SIGINT gets sent to the debuggee and not to delve.
func waitForDisconnectSignal(disconnectChan chan struct{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ch:
	case <-disconnectChan:
	}
}

func splitArgs(cmd *cobra.Command, args []string) ([]string, []string) {
	if cmd.ArgsLenAtDash() >= 0 {
		return args[:cmd.ArgsLenAtDash()], args[cmd.ArgsLenAtDash():]
	}
	return args, []string{}
}

func connect(addr string, clientConn net.Conn, kind debugger.ExecuteKind) int {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	// Create and start a terminal - attach to running instance
	var client service.Client
	if clientConn != nil {
		client = service.NewClientFromConn(clientConn)
	} else {
		client = service.NewClient(addr)
	}
	if client.IsMulticlient() {
		state, _ := client.GetStateNonBlocking()
		// The error return of GetState will usually be the ErrProcessExited,
		// which we don't care about. If there are other errors they will show up
		// later, here we are only concerned about stopping a running target so
		// that we can initialize our connection.
		if state != nil && state.Running {
			_, err := client.Halt()
			if err != nil {
				log.Error("could not halt: %v", err)
				return 1
			}
		}
	}
	term := terminal.New(client, conf)
	status, err := term.Run()
	if err != nil {
		log.Error("%v", err)
	}
	return status
}
