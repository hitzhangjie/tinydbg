package debug

func init() {
	registerDebugCmd(threadsCmd)
	registerDebugCmd(threadCmd)
	registerDebugCmd(goroutinesCmd)
	registerDebugCmd(goroutineCmd)
}

var threadsCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"threads"},
		group:   goroutineCmds,
		cmdFn:   threads,
		helpMsg: "Print out info for every traced thread.",
	}
}

var threadCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"thread", "tr"},
		group:   goroutineCmds,
		cmdFn:   thread,
		helpMsg: `Switch to the specified thread.

	thread <id>`,
	}
}

var goroutinesCmd = func(c *DebugCommands) *command {
	return &command{
		aliases: []string{"goroutines", "grs"},
		group:   goroutineCmds,
		cmdFn:   c.goroutines,
		helpMsg: `List program goroutines.

	goroutines [-u|-r|-g|-s] [-t [depth]] [-l] [-with loc expr] [-without loc expr] [-group argument] [-chan expr] [-exec command]

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

	Where:
	userloc: filter by the location of the topmost stackframe in user code
	curloc: filter by the location of the topmost stackframe (including frames inside private runtime functions)
	goloc: filter by the location of the go instruction that created the goroutine
	startloc: filter by the location of the start function
	
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

CHANNELS
	
To only show goroutines waiting to send to or receive from a specific channel use:

	goroutines -chan expr
	
Note that 'expr' must not contain spaces.

GROUPING

	goroutines -group (userloc|curloc|goloc|startloc|running|user)

	Where:
	userloc: groups goroutines by the location of the topmost stackframe in user code
	curloc: groups goroutines by the location of the topmost stackframe
	goloc: groups goroutines by the location of the go instruction that created the goroutine
	startloc: groups goroutines by the location of the start function
	running: groups goroutines by whether they are running or not
	user: groups goroutines by weather they are user or runtime goroutines


Groups goroutines by the given location, running status or user classification, up to 5 goroutines per group will be displayed as well as the total number of goroutines in the group.

	goroutines -group label key

Groups goroutines by the value of the label with the specified key.

EXEC

	goroutines -exec <command>

Runs the command on every goroutine.
`,
	}
}

var goroutineCmd = func(c *DebugCommands) *command {
	return &command{
		aliases:         []string{"goroutine", "gr"},
		group:           goroutineCmds,
		allowedPrefixes: onPrefix,
		cmdFn:           c.goroutine,
		helpMsg: `Shows or changes current goroutine

	goroutine
	goroutine <id>
	goroutine <id> <command>

Called without arguments it will show information about the current goroutine.
Called with a single argument it will switch to the specified goroutine.
Called with more arguments it will execute a command on the specified goroutine.`,
	}
}
