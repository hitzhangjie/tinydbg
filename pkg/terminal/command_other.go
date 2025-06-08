package terminal

func init() {
	registerDebugCmd(helpCmd)
	registerDebugCmd(exitCmd)
	registerDebugCmd(sourceCmd)
	registerDebugCmd(sourcesCmd)
	registerDebugCmd(configCmd)
	registerDebugCmd(editCmd)
	registerDebugCmd(librariesCmd)
	registerDebugCmd(dumpCmd)
	registerDebugCmd(transcriptCmd)
	registerDebugCmd(targetCmd)
}

var helpCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"help", "h"},
		cmdFn:   c.help,
		helpMsg: `Prints the help message.

	help [command]

Type "help" followed by the name of a command for more information about it.`,
	}
}

var exitCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"exit", "quit", "q"},
		cmdFn:   exitCommand,
		helpMsg: `Exit the debugger.
		
	exit [-c]
	
When connected to a headless instance started with the --accept-multiclient, pass -c to resume the execution of the target process before disconnecting.`,
	}
}

var sourceCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"source"},
		cmdFn:   c.sourceCommand,
		helpMsg: `Executes a file containing a list of delve commands

	source <path>

Note: tinydbg removes the support of the following two features:
1. If path ends with the .star extension it will be interpreted as a starlark script. See Documentation/cli/starlark.md for the syntax.
2. If path is a single '-' character an interactive starlark interpreter will start instead. Type 'exit' to exit.`,
	}
}

var sourcesCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"sources"},
		cmdFn:   sources,
		helpMsg: `Print list of source files.

	sources [<regex>]

If regex is specified only the source files matching it will be returned.`,
	}
}

var configCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"config"},
		cmdFn:   configureCmd,
		helpMsg: `Changes configuration parameters.

	config -list

Show all configuration parameters.

	config -save

Saves the configuration file to disk, overwriting the current configuration file.

	config <parameter> <value>

Changes the value of a configuration parameter.

	config substitute-path <from> <to>
	config substitute-path <from>
	config substitute-path -clear
	config substitute-path -guess

Adds or removes a path substitution rule, if -clear is used all
substitute-path rules are removed. Without arguments shows the current list
of substitute-path rules.
The -guess option causes Delve to try to guess your substitute-path
configuration automatically.
See also Documentation/cli/substitutepath.md for how the rules are applied.

	config alias <command> <alias>
	config alias <alias>

Defines <alias> as an alias to <command> or removes an alias.

	config debug-info-directories -add <path>
	config debug-info-directories -rm <path>
	config debug-info-directories -clear

Adds, removes or clears debug-info-directories.`,
	}
}

var editCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"edit", "ed"},
		cmdFn:   edit,
		helpMsg: `Open where you are in $DELVE_EDITOR or $EDITOR

	edit [locspec]
	
If locspec is omitted edit will open the current source file in the editor, otherwise it will open the specified location.`,
	}
}

var librariesCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"libraries"},
		cmdFn:   libraries,
		helpMsg: `List loaded dynamic libraries`,
	}
}

var dumpCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"dump"},
		cmdFn:   dump,
		helpMsg: `Creates a core dump from the current process state

	dump <output file>

The core dump is always written in ELF, even on systems (windows, macOS) where this is not customary. For environments other than linux/amd64 threads and registers are dumped in a format that only Delve can read back.`,
	}
}

var transcriptCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"transcript"},
		cmdFn:   transcript,
		helpMsg: `Appends command output to a file.

	transcript [-t] [-x] <output file>
	transcript -off

Output of Delve's command is appended to the specified output file. If '-t' is specified and the output file exists it is truncated. If '-x' is specified output to stdout is suppressed instead.

Using the -off option disables the transcript.`,
	}
}

var targetCmd = func(c *DebugSession) *command {
	return &command{
		aliases: []string{"target"},
		cmdFn:   target,
		helpMsg: `Manages child process debugging.

	target follow-exec [-on [regex]] [-off]

Enables or disables follow exec mode. When follow exec mode Delve will automatically attach to new child processes executed by the target process. An optional regular expression can be passed to 'target follow-exec', only child processes with a command line matching the regular expression will be followed.

	target list

List currently attached processes.

	target switch [pid]

Switches to the specified process.`,
	}
}
