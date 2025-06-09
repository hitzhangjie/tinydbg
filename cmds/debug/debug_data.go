package debug

func init() {
	registerDebugCmd(printCmd)
	registerDebugCmd(whatisCmd)
	registerDebugCmd(setCmd)
	registerDebugCmd(argsCmd)
	registerDebugCmd(localsCmd)
	registerDebugCmd(varsCmd)
	registerDebugCmd(regsCmd)
	registerDebugCmd(examinememCmd)
	registerDebugCmd(displayCmd)
}

var printCmd = func(c *DebugCommands) *command {
	return &command{
		aliases:         []string{"print", "p"},
		group:           dataCmds,
		allowedPrefixes: onPrefix | deferredPrefix,
		cmdFn:           c.printVar,
		helpMsg: `Evaluate an expression.

	[goroutine <n>] [frame <m>] print [%format] <expression>

See Documentation/cli/expr.md for a description of supported expressions.

The optional format argument is a format specifier, like the ones used by the fmt package. For example "print %x v" will print v as an hexadecimal number.`,
	}
}

// dataCommands returns all data-related commands
var whatisCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"whatis"},
		group:   dataCmds,
		cmdFn:   whatisCommand,
		helpMsg: `Prints type of an expression.

	whatis <expression>`,
	}
}

var setCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"set"},
		group:   dataCmds,
		cmdFn:   setVar,
		helpMsg: `Changes the value of a variable.

	[goroutine <n>] [frame <m>] set <variable> = <value>

See Documentation/cli/expr.md for a description of supported expressions. Only numerical variables and pointers can be changed.`,
	}
}

var argsCmd = func(c *DebugCommands) *command {
	return &command{
		aliases:         []string{"args"},
		allowedPrefixes: onPrefix | deferredPrefix,
		group:           dataCmds,
		cmdFn:           args,
		helpMsg: `Print function arguments.

	[goroutine <n>] [frame <m>] args [-v] [<regex>]

If regex is specified only function arguments with a name matching it will be returned. If -v is specified more information about each function argument will be shown.`,
	}
}

var localsCmd = func(c *DebugCommands) *command {
	return &command{
		aliases:         []string{"locals"},
		allowedPrefixes: onPrefix | deferredPrefix,
		group:           dataCmds,
		cmdFn:           locals,
		helpMsg: `Print local variables.

	[goroutine <n>] [frame <m>] locals [-v] [<regex>]

The name of variables that are shadowed in the current scope will be shown in parenthesis.

If regex is specified only local variables with a name matching it will be returned. If -v is specified more information about each local variable will be shown.`,
	}
}

var varsCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"vars"},
		cmdFn:   vars,
		group:   dataCmds,
		helpMsg: `Print package variables.

	vars [-v] [<regex>]

If regex is specified only package variables with a name matching it will be returned. If -v is specified more information about each package variable will be shown.`,
	}
}

var regsCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"regs"},
		cmdFn:   regs,
		group:   dataCmds,
		helpMsg: `Print contents of CPU registers.

	regs [-a]

Argument -a shows more registers. Individual registers can also be displayed by 'print' and 'display'. See Documentation/cli/expr.md.`,
	}
}

var examinememCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"examinemem", "x"},
		group:   dataCmds,
		cmdFn:   examineMemoryCmd,
		helpMsg: `Examine raw memory at the given address.

Examine memory:

	examinemem [-fmt <format>] [-count|-len <count>] [-size <size>] <address>
	examinemem [-fmt <format>] [-count|-len <count>] [-size <size>] -x <expression>

Format represents the data format and the value is one of this list (default hex): bin(binary), oct(octal), dec(decimal), hex(hexadecimal) and raw.
Length is the number of bytes (default 1) and must be less than or equal to 1000.
Address is the memory location of the target to examine. Please note '-len' is deprecated by '-count and -size'.
Expression can be an integer expression or pointer value of the memory location to examine.

For example:

    x -fmt hex -count 20 -size 1 0xc00008af38
    x -fmt hex -count 20 -size 1 -x 0xc00008af38 + 8
    x -fmt hex -count 20 -size 1 -x &myVar
    x -fmt hex -count 20 -size 1 -x myPtrVar`,
	}
}

var displayCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"display"},
		group:   dataCmds,
		cmdFn:   display,
		helpMsg: `Print value of an expression every time the program stops.

	display -a [%format] <expression>
	display -d <number>

The '-a' option adds an expression to the list of expression printed every time the program stops. The '-d' option removes the specified expression from the list.

If display is called without arguments it will print the value of all expression in the list.`,
	}
}
