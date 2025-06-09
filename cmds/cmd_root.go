package cmds

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/hitzhangjie/tinydbg/cmds/debug"
	"github.com/hitzhangjie/tinydbg/pkg/config"
	"github.com/hitzhangjie/tinydbg/pkg/logflags"
	"github.com/hitzhangjie/tinydbg/pkg/proc"
	"github.com/hitzhangjie/tinydbg/service"
	"github.com/hitzhangjie/tinydbg/service/api"
	"github.com/hitzhangjie/tinydbg/service/debugger"
	"github.com/hitzhangjie/tinydbg/service/rpc2"
	"github.com/hitzhangjie/tinydbg/service/rpccommon"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	// logFlag is whether to log debug statements.
	logFlag bool
	// logOutput is a comma separated list of components that should produce debug output.
	logOutput string
	// logDest is the file path or file descriptor where logs should go.
	logDest string
	// headless is whether to run without terminal.
	headless bool
	// continueOnStart is whether to continue the process on startup
	continueOnStart bool
	// acceptMulti allows multiple clients to connect to the same server
	acceptMulti bool
	// addr is the debugging server listen address.
	addr string
	// initFile is the path to initialization file.
	initFile string
	// buildFlags is the flags passed during compiler invocation.
	buildFlags string
	// workingDir is the working directory for running the program.
	workingDir string
	// tty is used to provide an alternate TTY for the program you wish to debug.
	tty string
	// disableASLR is used to disable ASLR
	disableASLR bool

	// rootCommand is the root of the command tree.
	rootCommand *cobra.Command

	traceAttachPid     int
	traceExecFile      string
	traceTestBinary    bool
	traceStackDepth    int
	traceShowTimestamp bool
	traceFollowCalls   int

	// redirect specifications for target process
	redirects []string

	allowNonTerminalInteractive bool

	conf        *config.Config
	loadConfErr error

	attachWaitFor         string
	attachWaitForInterval float64
	attachWaitForDuration float64
)

const longDesc = `tinydbg is a debugger for Go (trimmed from go-delve/delve), which only supports linux/amd64.

It enables you to interact with your program by controlling the execution of the process,
evaluating variables, and providing information of thread / goroutine state, CPU register state and more.

The goal of this tool is to provide a simple yet powerful interface for debugging Go programs.

Pass flags to the program you are debugging using ` + "`--`" + `, for example:

` + "`tinydbg exec ./hello -- server --config conf/config.toml`"

// New returns an initialized command tree.
func New() *cobra.Command {
	// Config setup and load.
	//
	// Delay reporting errors about configuration loading delayed until after the
	// server is started so that the "server listening at" message is always
	// the first thing emitted. Also, logflags hasn't been set up yet at this point.
	conf, loadConfErr = config.LoadConfig()
	buildFlagsDefault := ""

	// Main dlv root command.
	rootCommand = &cobra.Command{
		Use:   "tinydbg",
		Short: "tinydbg is a lightweight debugger trimmed from Delve (Dlv) for the Go programming language.",
		Long:  longDesc,
	}

	rootCommand.PersistentFlags().StringVarP(&addr, "listen", "l", "127.0.0.1:0", "Debugging server listen address. Prefix with 'unix:' to use a unix domain socket.")
	must(rootCommand.RegisterFlagCompletionFunc("listen", cobra.NoFileCompletions))

	rootCommand.PersistentFlags().BoolVarP(&logFlag, "log", "", false, "Enable debugging server logging.")
	rootCommand.PersistentFlags().StringVarP(&logOutput, "log-output", "", "", `Comma separated list of components that should produce debug output (see 'dlv help log')`)
	must(rootCommand.RegisterFlagCompletionFunc("log-output", cobra.FixedCompletions([]string{"debugger", "debuglineerr", "rpc", "fncall", "stack"}, cobra.ShellCompDirectiveNoFileComp)))
	rootCommand.PersistentFlags().StringVarP(&logDest, "log-dest", "", "", "Writes logs to the specified file or file descriptor (see 'dlv help log').")
	must(rootCommand.MarkPersistentFlagFilename("log-dest", "log"))

	rootCommand.PersistentFlags().BoolVarP(&headless, "headless", "", false, "Run debug server only, in headless mode. Server will accept JSON-RPC client connections.")
	rootCommand.PersistentFlags().BoolVarP(&acceptMulti, "accept-multiclient", "", false, "Allows a headless server to accept multiple client connections via JSON-RPC.")
	rootCommand.PersistentFlags().StringVar(&initFile, "init", "", "Init file, executed by the terminal client.")
	must(rootCommand.MarkPersistentFlagFilename("init"))
	rootCommand.PersistentFlags().StringVar(&buildFlags, "build-flags", buildFlagsDefault, "Build flags, to be passed to the compiler. For example: --build-flags=\"-tags=integration -mod=vendor -cover -v\"")
	must(rootCommand.RegisterFlagCompletionFunc("build-flags", cobra.NoFileCompletions))
	rootCommand.PersistentFlags().StringVar(&workingDir, "wd", "", "Working directory for running the program.")
	must(rootCommand.MarkPersistentFlagDirname("wd"))
	rootCommand.PersistentFlags().StringArrayVarP(&redirects, "redirect", "r", []string{}, "Specifies redirect rules for target process (see 'dlv help redirect')")
	must(rootCommand.MarkPersistentFlagFilename("redirect"))
	rootCommand.PersistentFlags().BoolVar(&allowNonTerminalInteractive, "allow-non-terminal-interactive", false, "Allows interactive sessions of Delve that don't have a terminal as stdin, stdout and stderr")
	rootCommand.PersistentFlags().BoolVar(&disableASLR, "disable-aslr", false, "Disables address space randomization")

	rootCommand.AddCommand(attachCommand)
	rootCommand.AddCommand(connectCommand)
	rootCommand.AddCommand(debugCommand)
	rootCommand.AddCommand(execCommand)
	rootCommand.AddCommand(traceCommand)
	rootCommand.AddCommand(coreCommand)
	rootCommand.AddCommand(logCommand)
	rootCommand.AddCommand(redirectCommand)
	rootCommand.AddCommand(substituteCommand)

	rootCommand.DisableAutoGenTag = true

	// hide specific flags for each command
	configUsageFunc(rootCommand)

	return rootCommand
}

func isBreakpointExistsErr(err error) bool {
	return strings.Contains(err.Error(), "Breakpoint exists")
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
	if runtime.GOOS == "windows" {
		// On windows Ctrl-C sent to inferior process is delivered
		// as SIGINT to delve. Ignore it instead of stopping the server
		// in order to be able to debug signal handlers.
		go func() {
			for range ch {
			}
		}()
		<-disconnectChan
	} else {
		select {
		case <-ch:
		case <-disconnectChan:
		}
	}
}

func splitArgs(cmd *cobra.Command, args []string) ([]string, []string) {
	if cmd.ArgsLenAtDash() >= 0 {
		return args[:cmd.ArgsLenAtDash()], args[cmd.ArgsLenAtDash():]
	}
	return args, []string{}
}

func connect(addr string, clientConn net.Conn, conf *config.Config) int {
	// Create and start a terminal - attach to running instance
	var client *rpc2.RPCClient
	if clientConn == nil {
		if clientConn = netDial(addr); clientConn == nil {
			return 1 // already logged
		}
	}
	client = rpc2.NewClientFromConn(clientConn)
	if client.IsMulticlient() {
		state, _ := client.GetStateNonBlocking()
		// The error return of GetState will usually be the ErrProcessExited,
		// which we don't care about. If there are other errors they will show up
		// later, here we are only concerned about stopping a running target so
		// that we can initialize our connection.
		if state != nil && state.Running {
			_, err := client.Halt()
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not halt: %v", err)
				return 1
			}
		}
	}
	session := debug.New(client, conf)
	session.InitFile = initFile
	status, err := session.Run()
	if err != nil {
		fmt.Println(err)
	}
	return status
}

func execute(attachPid int, processArgs []string, conf *config.Config, coreFile string, kind debugger.ExecuteKind, dlvArgs []string, buildFlags string) int {
	if err := logflags.Setup(logFlag, logOutput, logDest); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	defer logflags.Close()
	if loadConfErr != nil {
		logflags.DebuggerLogger().Errorf("%v", loadConfErr)
	}

	if headless && (initFile != "") {
		fmt.Fprint(os.Stderr, "Warning: init file ignored with --headless\n")
	}
	if continueOnStart {
		if !headless {
			fmt.Fprint(os.Stderr, "Error: --continue only works with --headless; use an init file\n")
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

	if !headless && !allowNonTerminalInteractive {
		for _, f := range []struct {
			name string
			file *os.File
		}{{"Stdin", os.Stdin}, {"Stdout", os.Stdout}, {"Stderr", os.Stderr}} {
			if f.file == nil {
				continue
			}
			if !isatty.IsTerminal(f.file.Fd()) {
				fmt.Fprintf(os.Stderr, "%s is not a terminal, use '-r' to specify redirects for the target process or --allow-non-terminal-interactive=true if you really want to specify a redirect for Delve\n", f.name)
				return 1
			}
		}
	}

	if len(redirects) > 0 && tty != "" {
		fmt.Fprintf(os.Stderr, "Can not use -r and --tty together\n")
		return 1
	}

	redirects, err := parseRedirects(redirects)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}

	var listener net.Listener
	var clientConn net.Conn

	// Make a TCP listener
	if headless {
		listener, err = netListen(addr)
	} else {
		listener, clientConn = service.ListenerPipe()
	}
	if err != nil {
		fmt.Printf("couldn't start listener: %s\n", err)
		return 1
	}
	defer listener.Close()

	disconnectChan := make(chan struct{})

	if workingDir == "" {
		workingDir = "."
	}

	// Create and start a debugger server
	server := rpccommon.NewServer(&service.Config{
		Listener:       listener,
		ProcessArgs:    processArgs,
		AcceptMulti:    acceptMulti,
		DisconnectChan: disconnectChan,
		Debugger: debugger.Config{
			AttachPid:             attachPid,
			WorkingDir:            workingDir,
			CoreFile:              coreFile,
			Foreground:            headless && tty == "",
			Packages:              dlvArgs,
			BuildFlags:            buildFlags,
			ExecuteKind:           kind,
			TTY:                   tty,
			Stdin:                 redirects[0],
			Stdout:                proc.OutputRedirect{Path: redirects[1]},
			Stderr:                proc.OutputRedirect{Path: redirects[2]},
			DisableASLR:           disableASLR,
			AttachWaitFor:         attachWaitFor,
			AttachWaitForInterval: attachWaitForInterval,
			AttachWaitForDuration: attachWaitForDuration,
		},
	})

	if err := server.Run(); err != nil {
		if errors.Is(err, api.ErrNotExecutable) {
			switch kind {
			case debugger.ExecutingGeneratedFile:
				fmt.Fprintln(os.Stderr, "Can not debug non-main package")
				return 1
			case debugger.ExecutingExistingFile:
				fmt.Fprintf(os.Stderr, "%s is not executable\n", processArgs[0])
				return 1
			default:
				// fallthrough
			}
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if headless {
		if continueOnStart {
			addr := listener.Addr().String()
			if _, isuds := listener.(*net.UnixListener); isuds {
				addr = "unix:" + addr
			}
			client := rpc2.NewClientFromConn(netDial(addr))
			client.Disconnect(true) // true = continue after disconnect
		}
		waitForDisconnectSignal(disconnectChan)
		err = server.Stop()
		if err != nil {
			fmt.Println(err)
		}

		return 0
	}

	return connect(listener.Addr().String(), clientConn, conf)
}

func parseRedirects(redirects []string) ([3]string, error) {
	r := [3]string{}
	names := [3]string{"stdin", "stdout", "stderr"}
	for _, redirect := range redirects {
		idx := 0
		for i, name := range names {
			pfx := name + ":"
			if strings.HasPrefix(redirect, pfx) {
				idx = i
				redirect = redirect[len(pfx):]
				break
			}
		}
		if r[idx] != "" {
			return r, fmt.Errorf("redirect error: %s redirected twice", names[idx])
		}
		r[idx] = redirect
	}
	return r, nil
}

const unixAddrPrefix = "unix:"

func netListen(addr string) (net.Listener, error) {
	if strings.HasPrefix(addr, unixAddrPrefix) {
		return net.Listen("unix", addr[len(unixAddrPrefix):])
	}
	return net.Listen("tcp", addr)
}

func netDial(addr string) net.Conn {
	var conn net.Conn
	var err error
	if strings.HasPrefix(addr, unixAddrPrefix) {
		conn, err = net.Dial("unix", addr[len(unixAddrPrefix):])
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		logflags.RPCLogger().Errorf("error dialing %s: %v", addr, err)
	}
	return conn
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
