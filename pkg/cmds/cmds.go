package cmds

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/log"
	"github.com/hitzhangjie/dlv/service"
	"github.com/hitzhangjie/dlv/service/api"
	"github.com/hitzhangjie/dlv/service/debugger"
	"github.com/hitzhangjie/dlv/service/rpccommon"
	"github.com/hitzhangjie/dlv/service/rpcv2"
)

var (
	// debugger mode
	headless        bool // whether to run without terminal
	continueOnStart bool // whether to continue the process on startup
	acceptMulti     bool // whether allows multiple clients to connect to the same server

	// debugger settins
	addr                        string   // the debugging server listen address
	initFile                    string   // the path to initialization file
	buildFlags                  string   // the flags passed during compiler invocation
	workingDir                  string   // the working directory for running the program
	tty                         string   // provide an alternate TTY for the program you wish to debug
	disableASLR                 bool     // whether disables ASLR
	backend                     string   // backend selection
	redirects                   []string // redirect specifications for target process
	allowNonTerminalInteractive bool

	// checkGoVersion is true if the debugger should check the version of Go
	// used to compile the executable and refuse to work on incompatible
	// versions.
	checkGoVersion bool

	// dapClientAddr is dap subcommand's flag that specifies the address of a DAP client.
	// If it is specified, the dap server starts a debug session by dialing to the client.
	// The dap server will serve only for the debug session.
	dapClientAddr string

	// checkLocalConnUser is true if the debugger should check that local
	// connections come from the same user that started the headless server
	checkLocalConnUser bool

	// trace settings
	traceAttachPid  int
	traceExecFile   string
	traceTestBinary bool
	traceStackDepth int
	traceUseEBPF    bool
)

// New returns an initialized command tree.
func New() *cobra.Command {
	return rootCommand
}

// 返回错误码给os.Exit(?)
func execute(attachPid int, processArgs []string, conf *config.Config, coreFile string, kind debugger.ExecuteKind, dlvArgs []string, buildFlags string) int {
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
				log.Error("%s is not a terminal, use '-r' to specify redirects for the target process or --allow-non-terminal-interactive=true if you really want to specify a redirect for Delve", f.name)
				return 1
			}
		}
	}

	if len(redirects) > 0 && tty != "" {
		log.Error("Can not use -r and --tty together")
		return 1
	}

	redirects, err := parseRedirects(redirects)
	if err != nil {
		log.Error("%v", err)
		return 1
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

	var server service.Server

	disconnectChan := make(chan struct{})

	if workingDir == "" {
		workingDir = "."
	}

	// Create and start a debugger server
	server = rpccommon.NewServer(&service.Config{
		Listener:           listener,
		ProcessArgs:        processArgs,
		AcceptMulti:        acceptMulti,
		CheckLocalConnUser: checkLocalConnUser,
		DisconnectChan:     disconnectChan,
		Debugger: debugger.Config{
			AttachPid:            attachPid,
			WorkingDir:           workingDir,
			Backend:              backend,
			CoreFile:             coreFile,
			Foreground:           headless && tty == "",
			Packages:             dlvArgs,
			BuildFlags:           buildFlags,
			ExecuteKind:          kind,
			DebugInfoDirectories: conf.DebugInfoDirectories,
			CheckGoVersion:       checkGoVersion,
			TTY:                  tty,
			Redirects:            redirects,
			DisableASLR:          disableASLR,
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
			client := rpcv2.NewClient(listener.Addr().String())
			client.Disconnect(true) // true = continue after disconnect
		}
		waitForDisconnectSignal(disconnectChan)
		err = server.Stop()
		if err != nil {
			log.Error("%v", err)
		}

		return status
	}

	return connect(listener.Addr().String(), clientConn, conf, kind)
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
