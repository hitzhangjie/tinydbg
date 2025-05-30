package terminal

func init() {
	registerDebugCmd(restartCmd)
	registerDebugCmd(rebuildCmd)
	registerDebugCmd(continueCmd)
	registerDebugCmd(stepCmd)
	registerDebugCmd(stepInstructionCmd)
	registerDebugCmd(nextInstructionCmd)
	registerDebugCmd(nextCmd)
	registerDebugCmd(stepoutCmd)
	registerDebugCmd(callCmd)
}

var restartCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"restart", "r"},
		group:   runCmds,
		cmdFn:   restart,
		helpMsg: `Restart process.

For recorded targets the command takes the following forms:

	restart					resets to the start of the recording
	restart [checkpoint]			resets the recording to the given checkpoint
	restart -r [newargv...]	[redirects...]	re-records the target process
	
For live targets the command takes the following forms:

	restart [newargv...] [redirects...]	restarts the process

If newargv is omitted the process is restarted (or re-recorded) with the same argument vector.
If -noargs is specified instead, the argument vector is cleared.

A list of file redirections can be specified after the new argument list to override the redirections defined using the '--redirect' command line option. A syntax similar to Unix shells is used:

	<input.txt	redirects the standard input of the target process from input.txt
	>output.txt	redirects the standard output of the target process to output.txt
	2>error.txt	redirects the standard error of the target process to error.txt
`,
	}
}

var rebuildCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"rebuild"},
		group:           runCmds,
		cmdFn:           c.rebuild,
		allowedPrefixes: revPrefix,
		helpMsg:         "Rebuild the target executable and restarts it. It does not work if the executable was not built by delve.",
	}
}

var continueCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"continue", "c"},
		group:           runCmds,
		cmdFn:           c.cont,
		allowedPrefixes: revPrefix,
		helpMsg: `Run until breakpoint or program termination.

	continue [<locspec>]

Optional locspec argument allows you to continue until a specific location is reached. The program will halt if a breakpoint is hit before reaching the specified location.

For example:

	continue main.main
	continue encoding/json.Marshal
`,
	}
}

var stepCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"step", "s"},
		group:           runCmds,
		cmdFn:           c.step,
		allowedPrefixes: revPrefix,
		helpMsg:         "Single step through program.",
	}
}

var stepInstructionCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"step-instruction", "si", "stepi"},
		group:           runCmds,
		allowedPrefixes: revPrefix,
		cmdFn:           c.stepInstruction,
		helpMsg:         "Single step a single cpu instruction.",
	}
}

var nextInstructionCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"next-instruction", "ni", "nexti"},
		group:           runCmds,
		allowedPrefixes: revPrefix,
		cmdFn:           c.nextInstruction,
		helpMsg:         "Single step a single cpu instruction, skipping function calls.",
	}
}

var nextCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"next", "n"},
		group:           runCmds,
		cmdFn:           c.next,
		allowedPrefixes: revPrefix,
		helpMsg: `Step over to next source line.

	next [count]

Optional [count] argument allows you to skip multiple lines.
`,
	}
}

var stepoutCmd = func(c *DebugSession) *command {
	return &command{
		aliases:         []string{"stepout", "so"},
		group:           runCmds,
		allowedPrefixes: revPrefix,
		cmdFn:           c.stepout,
		helpMsg:         "Step out of the current function.",
	}
}

var callCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"call"},
		group:   runCmds,
		cmdFn:   c.call,
		helpMsg: `Resumes process, injecting a function call (EXPERIMENTAL!!!)
	
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
`,
	}
}
