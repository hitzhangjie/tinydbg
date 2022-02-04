package terminal

var (
	helpCmdHelpMsg = `Prints the help message.

	help [command]

Type "help" followed by the name of a command for more information about it.`

	breakCmdHelpMsg = `Sets a breakpoint.

	break [name] <linespec>

See $GOPATH/src/github.com/hitzhangjie/dlv/Documentation/cli/locspec.md for the syntax of linespec.

See also: "help on", "help cond" and "help clear"`

	traceCmdHelpMsg = `Set tracepoint.

	trace [name] <linespec>

A tracepoint is a breakpoint that does not stop the execution of the program, instead when the tracepoint is hit a notification is displayed. See $GOPATH/src/github.com/hitzhangjie/dlv/Documentation/cli/locspec.md for the syntax of linespec.

See also: "help on", "help cond" and "help clear"`

	watchCmdHelpMsg = `Set watchpoint.

	watch [-r|-w|-rw] <expr>

	-r	stops when the memory location is read
	-w	stops when the memory location is written
	-rw	stops when the memory location is read or written

The memory location is specified with the same expression language used by 'print', for example:

	watch v

will watch the address of variable 'v'.

Note that writes that do not change the value of the watched memory address might not be reported.

See also: "help print".`

	restartCmdHelpMsg = `Restart process.

For recorded targets the command takes the following forms:

	restart							resets to the start of the recording
	restart [checkpoint]			resets the recording to the given checkpoint
	restart -r [newargv...]			re-records the target process

For live targets the command takes the following forms:

	restart [newargv...] 			restarts the process

If newargv is omitted the process is restarted (or re-recorded) with the same argument vector.
If -noargs is specified instead, the argument vector is cleared.
`
	rebuildCmdHelpMsg = "Rebuild the target executable and restarts it. It does not work if the executable was not built by delve."

	continueCmdHelpMsg = `Run until breakpoint or program termination.

	continue [<linespec>]

Optional linespec argument allows you to continue until a specific location is reached. The program will halt if a breakpoint is hit before reaching the specified location.

For example:

	continue main.main
	continue encoding/json.Marshal
`
	stepCmdHelpMsg = "Single step through program."

	stepInstCmdHelpMsg = "Single step a single cpu instruction."

	nextCmdHelpMsg = `Step over to next source line.

	next [count]

Optional [count] argument allows you to skip multiple lines.
`
	stepOutCmdHelpMsg = "Step out of the current function."

	callCmdHelpMsg = `Resumes process, injecting a function call (EXPERIMENTAL!!!)

	call [-unsafe] <function call expression>

Current limitations:
- only pointers to stack-allocated objects can be passed as argument.
- only some automatic type conversions are supported.
- functions can only be called on running goroutines that are not
  executing the runtime.
- the current goroutine needs to have at least 256 bytes of free space on
  the stack.
- functions can only be called when the goroutine is stopped at a safe
  point.
- calling a function will resume execution of all goroutines.
- only supported on linux's native backend.
`
	threadsCmdHelpMsg = "Print out info for every traced thread."

	threadCmdHelpMsg = `Switch to the specified thread.

	thread <id>`

	clearCmdHelpMsg = `Deletes breakpoint.

	clear <breakpoint name or id>`
	clearallCmdHelpMsg = `Deletes multiple breakpoints.

	clearall [<linespec>]

If called with the linespec argument it will delete all the breakpoints matching the linespec. If linespec is omitted all breakpoints are deleted.`

	toggleCmdHelpMsg = `Toggles on or off a breakpoint.

toggle <breakpoint name or id>`

	goroutinesCmdHelpMsg = `List program goroutines.

	goroutines [-u|-r|-g|-s] [-t [depth]] [-l] [-with loc expr] [-without loc expr] [-group argument]

Print out info for every goroutine. The flag controls what information is shown along with each goroutine:

	-u	displays location of topmost stackframe in user code (default)
	-r	displays location of topmost stackframe (including frames inside private runtime functions)
	-g	displays location of go instruction that created the goroutine
	-s	displays location of the start function
	-t	displays goroutine's stacktrace (an optional depth value can be specified, default: 10)
	-l	displays goroutine's labels

If no flag is specified the default is -u, i.e. the first frame within the first 30 frames that is not executing a runtime private function.

FILTERING

If -with or -without are specified only goroutines that match the given condition are returned.

To only display goroutines where the specified location contains (or does not contain, for -without and -wo) expr as a substring, use:

	goroutines -with (userloc|curloc|goloc|startloc) expr
	goroutines -w (userloc|curloc|goloc|startloc) expr
	goroutines -without (userloc|curloc|goloc|startloc) expr
	goroutines -wo (userloc|curloc|goloc|startloc) expr

To only display goroutines that have (or do not have) the specified label key and value, use:


	goroutines -with label key=value
	goroutines -without label key=value

To only display goroutines that have (or do not have) the specified label key, use:

	goroutines -with label key
	goroutines -without label key

To only display goroutines that are running (or are not running) on a OS thread, use:


	goroutines -with running
	goroutines -without running

To only display user (or runtime) goroutines, use:

	goroutines -with user
	goroutines -without user

GROUPING

	goroutines -group (userloc|curloc|goloc|startloc|running|user)

Groups goroutines by the given location, running status or user classification, up to 5 goroutines per group will be displayed as well as the total number of goroutines in the group.

	goroutines -group label key

Groups goroutines by the value of the label with the specified key.
`
	goroutineCmdHelpMsg = `Shows or changes current goroutine

	goroutine
	goroutine <id>
	goroutine <id> <command>

Called without arguments it will show information about the current goroutine.
Called with a single argument it will switch to the specified goroutine.
Called with more arguments it will execute a command on the specified goroutine.`
	breakpointsCmdHelpMsg = `Print out info for active breakpoints.

	breakpoints [-a]

Specifying -a prints all physical breakpoint, including internal breakpoints.`

	printCmdHelpMsg = `Evaluate an expression.

	[goroutine <n>] [frame <m>] print [%format] <expression>

See $GOPATH/src/github.com/hitzhangjie/dlv/Documentation/cli/expr.md for a description of supported expressions.

The optional format argument is a format specifier, like the ones used by the fmt package. For example "print %x v" will print v as an hexadecimal number.`

	whatisCmdHelpMsg = `Prints type of an expression.

	whatis <expression>`

	setCmdHelpMsg = `Changes the value of a variable.

	[goroutine <n>] [frame <m>] set <variable> = <value>

See $GOPATH/src/github.com/hitzhangjie/dlv/Documentation/cli/expr.md for a description of supported expressions. Only numerical variables and pointers can be changed.`

	sourcesCmdHelpMsg = `Print list of source files.

	sources [<regex>]

If regex is specified only the source files matching it will be returned.`

	funcsCmdHelpMsg = `Print list of functions.

	funcs [<regex>]

If regex is specified only the functions matching it will be returned.`

	typesCmdHelpMsg = `Print list of types

	types [<regex>]

If regex is specified only the types matching it will be returned.`
	argsCmdHelpMsg = `Print function arguments.

	[goroutine <n>] [frame <m>] args [-v] [<regex>]

If regex is specified only function arguments with a name matching it will be returned. If -v is specified more information about each function argument will be shown.`

	localsCmdHelpMsg = `Print local variables.

	[goroutine <n>] [frame <m>] locals [-v] [<regex>]

The name of variables that are shadowed in the current scope will be shown in parenthesis.

If regex is specified only local variables with a name matching it will be returned. If -v is specified more information about each local variable will be shown.`

	varsCmdHelpMsg = `Print package variables.

	vars [-v] [<regex>]

If regex is specified only package variables with a name matching it will be returned. If -v is specified more information about each package variable will be shown.`

	regsCmdHelpMsg = `Print contents of CPU registers.

	regs [-a]

Argument -a shows more registers. Individual registers can also be displayed by 'print' and 'display'. See $GOPATH/src/github.com/hitzhangjie/dlv/Documentation/cli/expr.md.`

	exitCmdHelpMsg = `Exit the debugger.

	exit [-c]

When connected to a headless instance started with the --accept-multiclient, pass -c to resume the execution of the target process before disconnecting.`

	listCmdHelpMsg = `Show source code.

	[goroutine <n>] [frame <m>] list [<linespec>]

Show source around current point or provided linespec.

For example:

	frame 1 list 69
	list testvariables.go:10000
	list main.main:30
	list 40`

	stackCmdHelpMsg = `Print stack trace.

	[goroutine <n>] [frame <m>] stack [<depth>] [-full] [-offsets] [-defer] [-a <n>] [-adepth <depth>] [-mode <mode>]

	-full		every stackframe is decorated with the value of its local variables and arguments.
	-offsets	prints frame offset of each frame.
	-defer		prints deferred function call stack for each frame.
	-a <n>		prints stacktrace of n ancestors of the selected goroutine (target process must have tracebackancestors enabled)
	-adepth <depth>	configures depth of ancestor stacktrace
	-mode <mode>	specifies the stacktrace mode, possible values are:
			normal	- attempts to automatically switch between cgo frames and go frames
			simple	- disables automatic switch between cgo and go
			fromg	- starts from the registers stored in the runtime.g struct
`
	frameCmdHelpMsg = `Set the current frame, or execute command on a different frame.

	frame <m>
	frame <m> <command>

The first form sets frame used by subsequent commands such as "print" or "set".
The second form runs the command on the given frame.`

	upCmdHelpMsg = `Move the current frame up.

	up [<m>]
	up [<m>] <command>

Move the current frame up by <m>. The second form runs the command on the given frame.`

	downCmdHelpMsg = `Move the current frame down.

	down [<m>]
	down [<m>] <command>

Move the current frame down by <m>. The second form runs the command on the given frame.`

	deferredCmdHelpMsg = `Executes command in the context of a deferred call.

	deferred <n> <command>

Executes the specified command (print, args, locals) in the context of the n-th deferred call in the current frame.`

	sourceCmdHelpMsg = `Executes a file containing a list of delve commands

	source <path>

If path ends with the .star extension it will be interpreted as a starlark script. See $GOPATH/src/github.com/hitzhangjie/dlv/Documentation/cli/starlark.md for the syntax.

If path is a single '-' character an interactive starlark interpreter will start instead. Type 'exit' to exit.`

	disassCmdHelpMsg = `Disassembler.

	[goroutine <n>] [frame <m>] disassemble [-a <start> <end>] [-l <locspec>]

If no argument is specified the function being executed in the selected stack frame will be executed.

	-a <start> <end>	disassembles the specified address range
	-l <locspec>		disassembles the specified function`

	onCmdHelpMsg = `Executes a command when a breakpoint is hit.

	on <breakpoint name or id> <command>
	on <breakpoint name or id> -edit


Supported commands: print, stack, goroutine, trace and cond.
To convert a breakpoint into a tracepoint use:

	on <breakpoint name or id> trace

The command 'on <bp> cond <cond-arguments>' is equivalent to 'cond <bp> <cond-arguments>'.

The command 'on x -edit' can be used to edit the list of commands executed when the breakpoint is hit.`

	conditionCmdHelpMsg = `Set breakpoint condition.

	condition <breakpoint name or id> <boolean expression>.
	condition -hitcount <breakpoint name or id> <operator> <argument>

Specifies that the breakpoint, tracepoint or watchpoint should break only if the boolean expression is true.

With the -hitcount option a condition on the breakpoint hit count can be set, the following operators are supported

	condition -hitcount bp > n
	condition -hitcount bp >= n
	condition -hitcount bp < n
	condition -hitcount bp <= n
	condition -hitcount bp == n
	condition -hitcount bp != n
	condition -hitcount bp % n

The '% n' form means we should stop at the breakpoint when the hitcount is a multiple of n.`

	configCmdHelpMsg = `Changes configuration parameters.

	config -list

Show all configuration parameters.

	config -save

Saves the configuration file to disk, overwriting the current configuration file.

	config <parameter> <value>

Changes the value of a configuration parameter.

	config substitute-path <from> <to>
	config substitute-path <from>

Adds or removes a path substitution rule.

	config alias <command> <alias>
	config alias <alias>

Defines <alias> as an alias to <command> or removes an alias.`

	editCmdHelpMsg = `Open where you are in $DELVE_EDITOR or $EDITOR

	edit [locspec]

If locspec is omitted edit will open the current source file in the editor, otherwise it will open the specified location.`

	librariesCmdHelpMsg = `List loaded dynamic libraries`

	examinememCmdHelpMsg = `Examine raw memory at the given address.

Examine memory:

	examinemem [-fmt <format>] [-count|-len <count>] [-size <size>] <address>
	examinemem [-fmt <format>] [-count|-len <count>] [-size <size>] -x <expression>

Format represents the data format and the value is one of this list (default hex): bin(binary), oct(octal), dec(decimal), hex(hexadecimal), addr(address).
Length is the number of bytes (default 1) and must be less than or equal to 1000.
Address is the memory location of the target to examine. Please note '-len' is deprecated by '-count and -size'.
Expression can be an integer expression or pointer value of the memory location to examine.

For example:

    x -fmt hex -count 20 -size 1 0xc00008af38
    x -fmt hex -count 20 -size 1 -x 0xc00008af38 + 8
    x -fmt hex -count 20 -size 1 -x &myVar
    x -fmt hex -count 20 -size 1 -x myPtrVar`

	displayCmdHelpMsg = `Print value of an expression every time the program stops.

	display -a [%format] <expression>
	display -d <number>

The '-a' option adds an expression to the list of expression printed every time the program stops. The '-d' option removes the specified expression from the list.

If display is called without arguments it will print the value of all expression in the list.`

	dumpCmdHelpMsg = `Creates a core dump from the current process state

	dump <output file>

The core dump is always written in ELF, even on systems (windows, macOS) where this is not customary. For environments other than linux/amd64 threads and registers are dumped in a format that only Delve can read back.`

	rewindCmdHelpMsg = "Run backwards until breakpoint or program termination."

	checkpointCmdHelpMsg = `Creates a checkpoint at the current position.

	checkpoint [note]

The "note" is arbitrary text that can be used to identify the checkpoint, if it is not specified it defaults to the current filename:line position.`

	checkpointsCmdHelpMsg = "Print out info for existing checkpoints."

	clearcheckCmdHelpMsg = `Deletes checkpoint.

	clear-checkpoint <id>`

	revCmdHelpMsg = `Reverses the execution of the target program for the command specified.
Currently, only the rev step-instruction command is supported.`
)
