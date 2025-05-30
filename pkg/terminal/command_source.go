package terminal

func init() {
	registerDebugCmd(funcsCmd)
	registerDebugCmd(typesCmd)
	registerDebugCmd(packagesCmd)
	registerDebugCmd(listCmd)
	registerDebugCmd(disassembleCmd)
}

var funcsCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"funcs"},
		cmdFn:   funcs,
		group:   sourceCmds,
		helpMsg: `Print list of functions.

	funcs [<regex>]

If regex is specified only the functions matching it will be returned.`,
	}
}

var typesCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"types"},
		cmdFn:   types,
		group:   sourceCmds,
		helpMsg: `Print list of types

	types [<regex>]

If regex is specified only the types matching it will be returned.`,
	}
}

var packagesCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"packages"},
		cmdFn:   packages,
		group:   sourceCmds,
		helpMsg: `Print list of packages.

	packages [<regex>]

If regex is specified only the packages matching it will be returned.`,
	}
}

var listCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"list", "ls", "l"},
		cmdFn:   listCommand,
		group:   sourceCmds,
		helpMsg: `Show source code.

	[goroutine <n>] [frame <m>] list [<locspec>]

Show source around current point or provided locspec.

For example:

	frame 1 list 69
	list testvariables.go:10000
	list main.main:30
	list 40`,
	}
}

var disassembleCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"disassemble", "disass"},
		cmdFn:   disassCommand,
		group:   sourceCmds,
		helpMsg: `Disassembler.

	[goroutine <n>] [frame <m>] disassemble [-a <start> <end>] [-l <locspec>]

If no argument is specified the function being executed in the selected stack frame will be executed.

	-a <start> <end>	disassembles the specified address range
	-l <locspec>		disassembles the specified function`,
	}
}
