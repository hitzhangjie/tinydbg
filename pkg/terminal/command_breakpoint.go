package terminal

func init() {
	registerDebugCmd(breakpointCmd)
	registerDebugCmd(traceCmd)
	registerDebugCmd(watchCmd)
	registerDebugCmd(clearCmd)
	registerDebugCmd(clearallCmd)
	registerDebugCmd(toggleCmd)
	registerDebugCmd(breakpointsCmd)
	registerDebugCmd(onCmd)
	registerDebugCmd(conditionCmd)
}

var breakpointCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"break", "b"},
		group:   breakCmds,
		cmdFn:   breakpoint,
		helpMsg: `Sets a breakpoint.

	break [name] [locspec] [if <condition>]

Locspec is a location specifier in the form of:

  * *<address> Specifies the location of memory address address. address can be specified as a decimal, hexadecimal or octal number
  * <filename>:<line> Specifies the line in filename. filename can be the partial path to a file or even just the base name as long as the expression remains unambiguous.
  * <line> Specifies the line in the current file
  * +<offset> Specifies the line offset lines after the current one
  * -<offset> Specifies the line offset lines before the current one
  * <function>[:<line>] Specifies the line inside function.
      The full syntax for function is <package>.(*<receiver type>).<function name> however the only required element is the function name,
      everything else can be omitted as long as the expression remains unambiguous. For setting a breakpoint on an init function (ex: main.init),
      the <filename>:<line> syntax should be used to break in the correct init function at the correct location.
  * /<regex>/ Specifies the location of all the functions matching regex

If locspec is omitted a breakpoint will be set on the current line.

If you would like to assign a name to the breakpoint you can do so with the form:

	break mybpname main.go:4

Finally, you can assign a condition to the newly created breakpoint by using the 'if' postfix form, like so:

	break main.go:55 if i == 5

Alternatively you can set a condition on a breakpoint after created by using the 'on' command.

See also: "help on", "help cond" and "help clear"`,
	}
}

var traceCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"trace", "t"},
		group:           breakCmds,
		cmdFn:           tracepoint,
		allowedPrefixes: onPrefix,
		helpMsg: `Set tracepoint.

	trace [name] [locspec]

A tracepoint is a breakpoint that does not stop the execution of the program, instead when the tracepoint is hit a notification is displayed. See Documentation/cli/locspec.md for the syntax of locspec. If locspec is omitted a tracepoint will be set on the current line.

See also: "help on", "help cond" and "help clear"`,
	}
}

var watchCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"watch"},
		group:   breakCmds,
		cmdFn:   watchpoint,
		helpMsg: `Set watchpoint.
	
	watch [-r|-w|-rw] <expr>
	
	-r	stops when the memory location is read
	-w	stops when the memory location is written
	-rw	stops when the memory location is read or written

The memory location is specified with the same expression language used by 'print', for example:

	watch v
	watch -w *(*int)(0x1400007c018)

will watch the address of variable 'v' and writes to an int at addr '0x1400007c018'.

Note that writes that do not change the value of the watched memory address might not be reported.

See also: "help print".`,
	}
}

var clearCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"clear"},
		group:   breakCmds,
		cmdFn:   clear,
		helpMsg: `Deletes breakpoint.

	clear <breakpoint name or id>`,
	}
}

var clearallCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"clearall"},
		group:   breakCmds,
		cmdFn:   clearAll,
		helpMsg: `Deletes multiple breakpoints.

	clearall [<locspec>]

If called with the locspec argument it will delete all the breakpoints matching the locspec. If locspec is omitted all breakpoints are deleted.`,
	}
}

// FIXME 删掉它
var toggleCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"toggle"},
		group:   breakCmds,
		cmdFn:   toggle,
		helpMsg: `Toggles on or off a breakpoint.

	toggle <breakpoint name or id>`,
	}
}

var breakpointsCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"breakpoints", "bp"},
		group:   breakCmds,
		cmdFn:   breakpoints,
		helpMsg: `Print out info for active breakpoints.
	
	breakpoints [-a]

Specifying -a prints all physical breakpoint, including internal breakpoints.`,
	}
}

var onCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"on"},
		group:   breakCmds,
		cmdFn:   c.onCmd,
		helpMsg: `Executes a command when a breakpoint is hit.

	on <breakpoint name or id> <command>
	on <breakpoint name or id> -edit
	

Supported commands: print, stack, goroutine, trace and cond. 
To convert a breakpoint into a tracepoint use:
	
	on <breakpoint name or id> trace

The command 'on <bp> cond <cond-arguments>' is equivalent to 'cond <bp> <cond-arguments>'.

The command 'on x -edit' can be used to edit the list of commands executed when the breakpoint is hit.`,
	}
}

var conditionCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"condition", "cond"},
		group:           breakCmds,
		cmdFn:           condition,
		allowedPrefixes: onPrefix,
		helpMsg: `Set breakpoint condition.

	condition <breakpoint name or id> <boolean expression>.
	condition -hitcount <breakpoint name or id> <operator> <argument>.
	condition -per-g-hitcount <breakpoint name or id> <operator> <argument>.
	condition -clear <breakpoint name or id>.

Specifies that the breakpoint, tracepoint or watchpoint should break only if the boolean expression is true.

See Documentation/cli/expr.md for a description of supported expressions and Documentation/cli/cond.md for a description of how breakpoint conditions are evaluated.

With the -hitcount option a condition on the breakpoint hit count can be set, the following operators are supported

	condition -hitcount bp > n
	condition -hitcount bp >= n
	condition -hitcount bp < n
	condition -hitcount bp <= n
	condition -hitcount bp == n
	condition -hitcount bp != n
	condition -hitcount bp % n

The -per-g-hitcount option works like -hitcount, but use per goroutine hitcount to compare with n.

With the -clear option a condition on the breakpoint can removed.
	
The '% n' form means we should stop at the breakpoint when the hitcount is a multiple of n.

Examples:

	cond 2 i == 10				breakpoint 2 will stop when variable i equals 10
	cond name runtime.curg.goid == 5	breakpoint 'name' will stop only on goroutine 5
	cond -clear 2				the condition on breakpoint 2 will be removed
`,
	}
}
