package cmds

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/config"
	"github.com/hitzhangjie/dlv/pkg/gobuild"
	"github.com/hitzhangjie/dlv/service/debugger"
)

// 'test' subcommand.
var testCommand = &cobra.Command{
	Use:   "test [package]",
	Short: "Compile test binary and begin debugging program.",
	Long: `Compiles a test binary with optimizations disabled and begins a new debug session.

The test command allows you to begin a new debug session in the context of your
unit tests. By default Delve will debug the tests in the current directory.
Alternatively you can specify a package name, and Delve will debug the tests in
that package instead. Double-dashes ` + "`--`" + ` can be used to pass arguments to the test program:

dlv test [package] -- -test.v -other-argument

See also: 'go help testflag'.`,
	Run: testCmdRun,
}

func init() {
	testCommand.Flags().String("output", "debug.test", "Output path for the binary.")
	rootCommand.AddCommand(testCommand)
}

func testCmdRun(cmd *cobra.Command, args []string) {
	conf, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	status := func() int {
		dlvArgs, targetArgs := splitArgs(cmd, args)
		debugname, ok := buildBinary(cmd, dlvArgs, true)
		if !ok {
			return 1
		}
		defer gobuild.Remove(debugname)
		processArgs := append([]string{debugname}, targetArgs...)

		if workingDir == "" {
			if len(dlvArgs) == 1 {
				workingDir = getPackageDir(dlvArgs[0])
			} else {
				workingDir = "."
			}
		}

		return execute(0, processArgs, conf, "", debugger.ExecutingGeneratedTest, dlvArgs, buildFlags)
	}()
	os.Exit(status)
}

func getPackageDir(pkg string) string {
	out, err := exec.Command("go", "list", "--json", pkg).CombinedOutput()
	if err != nil {
		return "."
	}
	type listOut struct {
		Dir string `json:"Dir"`
	}
	var listout listOut
	err = json.Unmarshal(out, &listout)
	if err != nil {
		return "."
	}
	return listout.Dir
}
