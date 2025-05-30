package terminal

func init() {
	registerDebugCmd(stackCmd)
	registerDebugCmd(frameCmd)
	registerDebugCmd(upCmd)
	registerDebugCmd(downCmd)
	registerDebugCmd(deferredCmd)
}

var stackCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"stack", "bt"},
		allowedPrefixes: onPrefix,
		group:           stackCmds,
		cmdFn:           stackCommand,
		helpMsg: `Print stack trace.

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
`,
	}
}

var frameCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"frame"},
		group:   stackCmds,
		cmdFn: func(t *Term, ctx callContext, arg string) error {
			return c.frameCommand(t, ctx, arg, frameSet)
		},
		helpMsg: `Set the current frame, or execute command on a different frame.

	frame <m>
	frame <m> <command>

The first form sets frame used by subsequent commands such as "print" or "set".
The second form runs the command on the given frame.`,
	}
}

var upCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"up"},
		group:   stackCmds,
		cmdFn: func(t *Term, ctx callContext, arg string) error {
			return c.frameCommand(t, ctx, arg, frameUp)
		},
		helpMsg: `Move the current frame up.

	up [<m>]
	up [<m>] <command>

Move the current frame up by <m>. The second form runs the command on the given frame.`,
	}
}

var downCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"down"},
		group:   stackCmds,
		cmdFn: func(t *Term, ctx callContext, arg string) error {
			return c.frameCommand(t, ctx, arg, frameDown)
		},
		helpMsg: `Move the current frame down.

	down [<m>]
	down [<m>] <command>

Move the current frame down by <m>. The second form runs the command on the given frame.`,
	}
}

var deferredCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"deferred"},
		group:   stackCmds,
		cmdFn:   c.deferredCommand,
		helpMsg: `Executes command in the context of a deferred call.

	deferred <n> <command>

Executes the specified command (print, args, locals) in the context of the n-th deferred call in the current frame.`,
	}
}
