package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hitzhangjie/dlv/pkg/goversion"
)

const DelveMainPackagePath = "github.com/go-delve/delve/cmd/dlv"

var Verbose bool
var NOTimeout bool
var TestIncludePIE bool
var TestSet, TestRegex, TestBackend, TestBuildMode string
var Tags *[]string
var Architecture string
var OS string

func NewMakeCommands() *cobra.Command {
	RootCommand := &cobra.Command{
		Use:   "make.go",
		Short: "make script for delve.",
	}

	RootCommand.AddCommand(&cobra.Command{
		Use:   "check-cert",
		Short: "Check certificate for macOS.",
		Run:   checkCertCmd,
	})

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build delve",
		Run: func(cmd *cobra.Command, args []string) {
			envflags := []string{}
			if len(Architecture) > 0 {
				envflags = append(envflags, "GOARCH="+Architecture)
			}
			if len(OS) > 0 {
				envflags = append(envflags, "GOOS="+OS)
			}
			if len(envflags) > 0 {
				executeEnv(envflags, "go", "build", "-ldflags", "-extldflags -static", buildFlags(), DelveMainPackagePath)
			} else {
				execute("go", "build", "-ldflags", "-extldflags -static", buildFlags(), DelveMainPackagePath)
			}
		},
	}
	Tags = buildCmd.PersistentFlags().StringArray("tags", []string{}, "Build tags")
	buildCmd.PersistentFlags().StringVar(&Architecture, "GOARCH", "", "Architecture to build for")
	buildCmd.PersistentFlags().StringVar(&OS, "GOOS", "", "OS to build for")
	RootCommand.AddCommand(buildCmd)

	RootCommand.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Installs delve",
		Run: func(cmd *cobra.Command, args []string) {
			execute("go", "install", buildFlags(), DelveMainPackagePath)
		},
	})

	RootCommand.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "Uninstalls delve",
		Run: func(cmd *cobra.Command, args []string) {
			execute("go", "clean", "-i", DelveMainPackagePath)
		},
	})

	test := &cobra.Command{
		Use:   "test",
		Short: "Tests delve",
		Long: `Tests delve.

Use the flags -s, -r and -b to specify which tests to run. Specifying nothing will run all tests relevant for the current environment (see testStandard).
`,
		Run: testCmd,
	}
	test.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Verbose tests")
	test.PersistentFlags().BoolVarP(&NOTimeout, "timeout", "t", false, "Set infinite timeouts")
	test.PersistentFlags().StringVarP(&TestSet, "test-set", "s", "", `Select the set of tests to run, one of either:
	all		tests all packages
	basic		tests proc, integration and terminal
	integration 	tests github.com/go-delve/delve/service/test
	package-name	test the specified package only
`)
	test.PersistentFlags().StringVarP(&TestRegex, "test-run", "r", "", `Only runs the tests matching the specified regex. This option can only be specified if testset is a single package`)
	test.PersistentFlags().StringVarP(&TestBackend, "test-backend", "b", "", `Runs tests for the specified backend only, one of either:
	default		the default backend
	lldb		lldb backend
	rr		rr backend

This option can only be specified if testset is basic or a single package.`)
	test.PersistentFlags().StringVarP(&TestBuildMode, "test-build-mode", "m", "", `Runs tests compiling with the specified build mode, one of either:
	normal		normal buildmode (default)
	pie		PIE buildmode
	
This option can only be specified if testset is basic or a single package.`)
	test.PersistentFlags().BoolVarP(&TestIncludePIE, "pie", "", true, "Standard testing should include PIE")

	RootCommand.AddCommand(test)

	RootCommand.AddCommand(&cobra.Command{
		Use:   "vendor",
		Short: "vendors dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			execute("go", "mod", "vendor")
		},
	})

	return RootCommand
}

func checkCert() bool {
	// If we're on OSX make sure the proper CERT env var is set.
	if os.Getenv("TRAVIS") == "true" || runtime.GOOS != "darwin" || os.Getenv("CERT") != "" {
		return true
	}

	x := exec.Command("_scripts/gencert.sh")
	x.Stdout = os.Stdout
	x.Stderr = os.Stderr
	x.Env = os.Environ()
	err := x.Run()
	if x.ProcessState != nil && !x.ProcessState.Success() {
		fmt.Printf("An error occurred when generating and installing a new certificate\n")
		return false
	}
	if err != nil {
		fmt.Printf("An error occoured when generating and installing a new certificate: %v\n", err)
		return false
	}
	os.Setenv("CERT", "dlv-cert")
	return true
}

func checkCertCmd(cmd *cobra.Command, args []string) {
	if !checkCert() {
		os.Exit(1)
	}
}

func strflatten(v []interface{}) []string {
	r := []string{}
	for _, s := range v {
		switch s := s.(type) {
		case []string:
			r = append(r, s...)
		case string:
			if s != "" {
				r = append(r, s)
			}
		}
	}
	return r
}

func executeq(env []string, cmd string, args ...interface{}) {
	x := exec.Command(cmd, strflatten(args)...)
	x.Stdout = os.Stdout
	x.Stderr = os.Stderr
	x.Env = os.Environ()
	for _, e := range env {
		x.Env = append(x.Env, e)
	}
	err := x.Run()
	if x.ProcessState != nil && !x.ProcessState.Success() {
		os.Exit(1)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func execute(cmd string, args ...interface{}) {
	fmt.Printf("%s %s\n", cmd, strings.Join(quotemaybe(strflatten(args)), " "))
	env := []string{}
	executeq(env, cmd, args...)
}

func executeEnv(env []string, cmd string, args ...interface{}) {
	fmt.Printf("%s %s %s\n", strings.Join(env, " "),
		cmd, strings.Join(quotemaybe(strflatten(args)), " "))
	executeq(env, cmd, args...)
}

func quotemaybe(args []string) []string {
	for i := range args {
		if strings.Index(args[i], " ") >= 0 {
			args[i] = fmt.Sprintf("%q", args[i])
		}
	}
	return args
}

func getoutput(cmd string, args ...interface{}) string {
	x := exec.Command(cmd, strflatten(args)...)
	x.Env = os.Environ()
	out, err := x.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing %s %v\n", cmd, args)
		log.Fatal(err)
	}
	if !x.ProcessState.Success() {
		fmt.Fprintf(os.Stderr, "Error executing %s %v\n", cmd, args)
		os.Exit(1)
	}
	return string(out)
}

func codesign(path string) {
	execute("codesign", "-s", os.Getenv("CERT"), path)
}

func installedExecutablePath() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return filepath.Join(gobin, "dlv")
	}
	gopath := strings.Split(getoutput("go", "env", "GOPATH"), ":")
	return filepath.Join(strings.TrimSpace(gopath[0]), "bin", "dlv")
}

func buildFlags() []string {
	buildSHA, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	ldFlags := "-X main.Build=" + strings.TrimSpace(string(buildSHA))
	return []string{fmt.Sprintf("-ldflags=%s", ldFlags)}
}

func testFlags() []string {
	_, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	testFlags := []string{"-count", "1", "-p", "1"}
	if Verbose {
		testFlags = append(testFlags, "-v")
	}
	if NOTimeout {
		testFlags = append(testFlags, "-timeout", "0")
	} else if os.Getenv("TRAVIS") == "true" {
		// Make test timeout shorter than Travis' own timeout so that Go can report which test hangs.
		testFlags = append(testFlags, "-timeout", "9m")
	}
	if len(os.Getenv("TEAMCITY_VERSION")) > 0 {
		testFlags = append(testFlags, "-json")
	}
	return testFlags
}

func testCmd(cmd *cobra.Command, args []string) {
	checkCertCmd(nil, nil)

	if TestSet == "" && TestBackend == "" && TestBuildMode == "" {
		if TestRegex != "" {
			fmt.Printf("Can not use --test-run without --test-set\n")
			os.Exit(1)
		}

		testStandard()
		return
	}

	if TestSet == "" {
		TestSet = "all"
	}

	if TestBackend == "" {
		TestBackend = "default"
	}

	if TestBuildMode == "" {
		TestBuildMode = "normal"
	}

	testCmdIntl(TestSet, TestRegex, TestBackend, TestBuildMode)
}

func testStandard() {
	fmt.Println("Testing default backend")
	testCmdIntl("all", "", "default", "normal")
	if inpath("lldb-server") && !goversion.VersionAfterOrEqual(runtime.Version(), 1, 14) {
		fmt.Println("\nTesting LLDB backend")
		testCmdIntl("basic", "", "lldb", "normal")
	}
	if TestIncludePIE {
		dopie := false
		switch runtime.GOOS {
		case "linux":
			dopie = true
		}
		if dopie {
			fmt.Println("\nTesting PIE buildmode, default backend")
			testCmdIntl("basic", "", "default", "pie")
			testCmdIntl("core", "", "default", "pie")
		}
	}
}

func testCmdIntl(testSet, testRegex, testBackend, testBuildMode string) {
	testPackages := testSetToPackages(testSet)
	if len(testPackages) == 0 {
		fmt.Printf("Unknown test set %q\n", testSet)
		os.Exit(1)
	}

	if testRegex != "" && len(testPackages) != 1 {
		fmt.Printf("Can not use test-run with test set %q\n", testSet)
		os.Exit(1)
	}

	buildModeFlag := ""
	if testBuildMode != "" && testBuildMode != "normal" {
		if testSet != "basic" && len(testPackages) != 1 {
			fmt.Printf("Can not use test-buildmode with test set %q\n", testSet)
			os.Exit(1)
		}
		buildModeFlag = "-test-buildmode=" + testBuildMode
	}

	if len(testPackages) > 3 {
		env := []string{}
		executeq(env, "go", "test", testFlags(), buildFlags(), testPackages, buildModeFlag)
	} else if testRegex != "" {
		execute("go", "test", testFlags(), buildFlags(), testPackages, "-run="+testRegex, buildModeFlag)
	} else {
		execute("go", "test", testFlags(), buildFlags(), testPackages, buildModeFlag)
	}
}

func testSetToPackages(testSet string) []string {
	switch testSet {
	case "", "all":
		return allPackages()

	case "basic":
		return []string{"github.com/go-delve/delve/pkg/proc", "github.com/go-delve/delve/service/test", "github.com/go-delve/delve/pkg/terminal"}

	case "integration":
		return []string{"github.com/go-delve/delve/service/test"}

	default:
		for _, pkg := range allPackages() {
			if pkg == testSet || strings.HasSuffix(pkg, "/"+testSet) {
				return []string{pkg}
			}
		}
		return nil
	}
}

func inpath(exe string) bool {
	path, _ := exec.LookPath(exe)
	return path != ""
}

func allPackages() []string {
	r := []string{}
	for _, dir := range strings.Split(getoutput("go", "list", "-mod=vendor", "./..."), "\n") {
		dir = strings.TrimSpace(dir)
		if dir == "" || strings.Contains(dir, "/vendor/") || strings.Contains(dir, "/_scripts") {
			continue
		}
		r = append(r, dir)
	}
	sort.Strings(r)
	return r
}

func main() {
	allPackages() // checks that vendor directory is synced as a side effect
	NewMakeCommands().Execute()
}
