package cmds

import "github.com/spf13/cobra"

var redirectCommand = &cobra.Command{
	Use:   "redirect",
	Short: "Help about file redirection.",
	Long: `The standard file descriptors of the target process can be controlled using the '-r' and '--tty' arguments. 

The --tty argument allows redirecting all standard descriptors to a terminal, specified as an argument to --tty.

The syntax for '-r' argument is:

		-r [source:]destination

Where source is one of 'stdin', 'stdout' or 'stderr' and destination is the path to a file. If the source is omitted stdin is used implicitly.

File redirects can also be changed using the 'restart' command.
`,
}
