package cmds

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/hitzhangjie/tinydbg/cmds/debug"
	"github.com/hitzhangjie/tinydbg/pkg/config"
	"github.com/hitzhangjie/tinydbg/pkg/logflags"
	"github.com/hitzhangjie/tinydbg/service"
	"github.com/hitzhangjie/tinydbg/service/api"
	"github.com/hitzhangjie/tinydbg/service/debugger"
	"github.com/hitzhangjie/tinydbg/service/rpc2"
	"github.com/hitzhangjie/tinydbg/service/rpccommon"
	"github.com/spf13/cobra"
)

// 'trace' subcommand.
var traceCommand = &cobra.Command{
	Use:   "trace <regexp>",
	Short: "Begin tracing program.",
	Long: `Trace program execution.

The trace sub command will set a tracepoint on every function matching the
provided regular expression and output information when tracepoint is hit.  This
is useful if you do not want to begin an entire debug session, but merely want
to know what functions your process is executing.

The output of the trace sub command is printed to stderr, so if you would like to
only see the output of the trace operations you can redirect stdout.`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(traceCmd(cmd, args, conf))
	},
	ValidArgsFunction: cobra.NoFileCompletions,
}

func init() {
	traceCommand.Flags().IntVarP(&traceAttachPid, "pid", "p", 0, "Pid to attach to.")
	must(traceCommand.RegisterFlagCompletionFunc("pid", cobra.NoFileCompletions))
	traceCommand.Flags().StringVarP(&traceExecFile, "exec", "e", "", "Binary file to exec and trace.")
	must(traceCommand.MarkFlagFilename("exec"))
	traceCommand.Flags().BoolVarP(&traceShowTimestamp, "timestamp", "", false, "Show timestamp in the output.")
	traceCommand.Flags().IntVarP(&traceStackDepth, "stack", "s", 0, "Show stack trace with given depth.")
	must(traceCommand.RegisterFlagCompletionFunc("stack", cobra.NoFileCompletions))
	traceCommand.Flags().IntVarP(&traceFollowCalls, "follow-calls", "", 0, "Trace all children of the function to the required depth.")
}

func traceCmd(cmd *cobra.Command, args []string, conf *config.Config) int {
	status := func() int {
		err := logflags.Setup(enableLogging, enableLoggers, logDest)
		defer logflags.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		if loadConfErr != nil {
			logflags.LogDebuggerLogger().Errorf("%v", loadConfErr)
		}

		if headless {
			fmt.Fprintf(os.Stderr, "Warning: headless mode not supported with trace\n")
		}
		if acceptMulti {
			fmt.Fprintf(os.Stderr, "Warning: accept multiclient mode not supported with trace")
		}

		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "you must supply a regexp for functions to trace\n")
			return 1
		}
		regexp := args[0]

		var processArgs []string
		if traceAttachPid == 0 && traceExecFile == "" {
			fmt.Fprintln(os.Stderr, "Either --pid or --exec must be specified")
			return 1
		}

		if traceAttachPid == 0 {
			processArgs = append([]string{traceExecFile}, args[1:]...)
		}

		// Make a local in-memory connection that client and server use to communicate
		listener, clientConn := service.ListenerPipe()
		defer listener.Close()

		if workingDir == "" {
			workingDir = "."
		}

		// Create and start a debug server
		server := rpccommon.NewServer(&service.Config{
			Listener:    listener,
			ProcessArgs: processArgs,
			APIVersion:  2,
			Debugger: debugger.Config{
				AttachPid:  traceAttachPid,
				WorkingDir: workingDir,
			},
		})
		if err := server.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		client := rpc2.NewClientFromConn(clientConn)
		defer client.Detach(true)

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT)

		go func() {
			<-ch
			client.Halt()
		}()
		funcs, err := client.ListFunctions(regexp, traceFollowCalls)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		success := false
		for i := range funcs {
			// Fall back to breakpoint based tracing if we get an error.
			var stackdepth int
			// Default size of stackdepth to trace function calls and descendants=20
			stackdepth = traceStackDepth
			if traceFollowCalls > 0 && stackdepth == 0 {
				stackdepth = 20
			}
			_, err = client.CreateBreakpoint(&api.Breakpoint{
				FunctionName:     funcs[i],
				Tracepoint:       true,
				Line:             -1,
				Stacktrace:       stackdepth,
				LoadArgs:         &debug.ShortLoadConfig,
				TraceFollowCalls: traceFollowCalls,
				RootFuncName:     regexp,
			})

			if err != nil && !isBreakpointExistsErr(err) {
				fmt.Fprintf(os.Stderr, "unable to set tracepoint on function %s: %#v\n", funcs[i], err)
				continue
			} else {
				success = true
			}
			addrs, err := client.FunctionReturnLocations(funcs[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "unable to set tracepoint on function %s: %#v\n", funcs[i], err)
				continue
			}
			for i := range addrs {
				_, err = client.CreateBreakpoint(&api.Breakpoint{
					Addr:             addrs[i],
					TraceReturn:      true,
					Stacktrace:       stackdepth,
					Line:             -1,
					LoadArgs:         &debug.ShortLoadConfig,
					TraceFollowCalls: traceFollowCalls,
					RootFuncName:     regexp,
				})
				if err != nil && !isBreakpointExistsErr(err) {
					fmt.Fprintf(os.Stderr, "unable to set tracepoint on function %s: %#v\n", funcs[i], err)
				} else {
					success = true
				}
			}
		}
		if !success {
			fmt.Fprintln(os.Stderr, "no breakpoints set")
			return 1
		}
		cmds := debug.NewDebugCommands(client)
		cfg := &config.Config{
			TraceShowTimestamp: traceShowTimestamp,
		}
		t := debug.New(client, cfg)
		t.SetTraceNonInteractive()
		t.RedirectTo(os.Stderr)
		defer t.Close()

		err = cmds.Call("continue", t)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			if !strings.Contains(err.Error(), "exited") {
				return 1
			}
		}
		return 0
	}()
	return status
}
