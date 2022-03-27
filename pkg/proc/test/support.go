package test

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/hitzhangjie/dlv/pkg/goversion"
	"github.com/hitzhangjie/dlv/pkg/log"
)

// EnableRace allows to configure whether the race detector is enabled on target process.
var EnableRace = flag.Bool("racetarget", false, "Enables race detector on inferior process")

var runningWithFixtures bool

// Fixture is a test binary.
type Fixture struct {
	// Name is the short name of the fixture.
	Name string
	// Path is the absolute path to the test binary.
	Path string
	// Source is the absolute path of the test binary source.
	Source string
	// BuildDir is the directory where the build command was run.
	BuildDir string
}

// FixtureKey holds the name and builds flags used for a test fixture.
type fixtureKey struct {
	Name  string
	Flags BuildFlags
}

// Fixtures is a map of fixtureKey{ Fixture.Name, buildFlags } to Fixture.
var fixtures = make(map[fixtureKey]Fixture)

// PathsToRemove is a list of files and directories to remove after running all the tests
var PathsToRemove []string

// FindFixturesDir will search for the directory holding all test fixtures
// beginning with the current directory and searching up 10 directories.
func FindFixturesDir() string {
	parent := ".."
	fixturesDir := "_fixtures"
	for depth := 0; depth < 10; depth++ {
		if _, err := os.Stat(fixturesDir); err == nil {
			break
		}
		fixturesDir = filepath.Join(parent, fixturesDir)
	}
	return fixturesDir
}

// BuildFlags used to build fixture.
type BuildFlags uint32

const (
	// LinkStrip enables '-ldflags="-s"'.
	LinkStrip BuildFlags = 1 << iota
	// EnableCGOOptimization will build CGO code with optimizations.
	EnableCGOOptimization
	// EnableInlining will build a binary with inline optimizations turned on.
	EnableInlining
	// EnableOptimization will build a binary with default optimizations.
	EnableOptimization
	// EnableDWZCompression will enable DWZ compression of DWARF sections.
	EnableDWZCompression
	BuildModePIE
	BuildModePlugin
	BuildModeExternalLinker
	AllNonOptimized
)

// BuildFixture will compile the fixture 'name' using the provided build flags.
func BuildFixture(name string, flags BuildFlags) Fixture {
	if !runningWithFixtures {
		panic("RunTestsWithFixtures not called")
	}
	fk := fixtureKey{name, flags}
	if f, ok := fixtures[fk]; ok {
		return f
	}

	if flags&EnableCGOOptimization == 0 {
		os.Setenv("CGO_CFLAGS", "-O0 -g")
	}

	fixturesDir := FindFixturesDir()

	// Make a (good enough) random temporary file name
	r := make([]byte, 4)
	rand.Read(r)
	dir := fixturesDir
	path := filepath.Join(fixturesDir, name+".go")
	if name[len(name)-1] == '/' {
		dir = filepath.Join(dir, name)
		path = ""
		name = name[:len(name)-1]
	}
	tmpfile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.%s", name, hex.EncodeToString(r)))

	buildFlags := []string{"build"}
	var ver goversion.GoVersion
	if flags&LinkStrip != 0 {
		buildFlags = append(buildFlags, "-ldflags=-s")
	}
	gcflagsv := []string{}
	if flags&EnableInlining == 0 {
		gcflagsv = append(gcflagsv, "-l")
	}
	if flags&EnableOptimization == 0 {
		gcflagsv = append(gcflagsv, "-N")
	}
	var gcflags string
	if flags&AllNonOptimized != 0 {
		gcflags = "-gcflags=all=" + strings.Join(gcflagsv, " ")
	} else {
		gcflags = "-gcflags=" + strings.Join(gcflagsv, " ")
	}
	buildFlags = append(buildFlags, gcflags, "-o", tmpfile)
	if *EnableRace {
		buildFlags = append(buildFlags, "-race")
	}
	if flags&BuildModePIE != 0 {
		buildFlags = append(buildFlags, "-buildmode=pie")
	} else {
		buildFlags = append(buildFlags, "-buildmode=exe")
	}
	if flags&BuildModePlugin != 0 {
		buildFlags = append(buildFlags, "-buildmode=plugin")
	}
	if flags&BuildModeExternalLinker != 0 {
		buildFlags = append(buildFlags, "-ldflags=-linkmode=external")
	}
	if ver.IsDevel() || ver.AfterOrEqual(goversion.GoVersion{Major: 1, Minor: 11, Rev: -1}) {
		if flags&EnableDWZCompression != 0 {
			buildFlags = append(buildFlags, "-ldflags=-compressdwarf=false")
		}
	}
	if path != "" {
		buildFlags = append(buildFlags, name+".go")
	}

	cmd := exec.Command("go", buildFlags...)
	cmd.Dir = dir

	// Build the test binary
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Error("Error compiling %s: %s", path, err)
		log.Error("%s", string(out))
		os.Exit(1)
	}

	if flags&EnableDWZCompression != 0 {
		cmd := exec.Command("dwz", tmpfile)
		if out, err := cmd.CombinedOutput(); err != nil {
			if regexp.MustCompile(`dwz: Section offsets in (.*?) not monotonically increasing`).FindString(string(out)) == "" {
				log.Error("Error running dwz on %s: %s", tmpfile, err)
				log.Error("%s\n", string(out))
				os.Exit(1)
			}
		}
	}

	source, _ := filepath.Abs(path)
	source = filepath.ToSlash(source)
	sympath, err := filepath.EvalSymlinks(source)
	if err == nil {
		source = strings.Replace(sympath, "\\", "/", -1)
	}

	absdir, _ := filepath.Abs(dir)

	fixture := Fixture{Name: name, Path: tmpfile, Source: source, BuildDir: absdir}

	fixtures[fk] = fixture
	return fixtures[fk]
}

// RunTestsWithFixtures will pre-compile test fixtures before running test
// methods. Test binaries are deleted before exiting.
func RunTestsWithFixtures(m *testing.M) int {
	runningWithFixtures = true
	defer func() {
		runningWithFixtures = false
	}()
	status := m.Run()

	// Remove the fixtures.
	for _, f := range fixtures {
		os.Remove(f.Path)
	}

	for _, p := range PathsToRemove {
		fi, err := os.Stat(p)
		if err != nil {
			panic(err)
		}
		if fi.IsDir() {
			SafeRemoveAll(p)
		} else {
			os.Remove(p)
		}
	}
	return status
}

// SafeRemoveAll removes dir and its contents but only as long as dir does
// not contain directories.
func SafeRemoveAll(dir string) {
	dh, err := os.Open(dir)
	if err != nil {
		return
	}
	defer dh.Close()
	fis, err := dh.Readdir(-1)
	if err != nil {
		return
	}
	for _, fi := range fis {
		if fi.IsDir() {
			return
		}
	}
	for _, fi := range fis {
		if err := os.Remove(filepath.Join(dir, fi.Name())); err != nil {
			return
		}
	}
	os.Remove(dir)
}

// WithPlugins builds the fixtures in plugins as plugins and returns them.
// The test calling WithPlugins will be skipped if the current combination
// of OS, architecture and version of GO doesn't support plugins or
// debugging plugins.
func WithPlugins(t *testing.T, flags BuildFlags, plugins ...string) []Fixture {
	if !goversion.VersionAfterOrEqual(runtime.Version(), 1, 12) {
		t.Skip("versions of Go before 1.12 do not include debug information in packages that import plugin (or they do but it's wrong)")
	}
	if runtime.GOOS != "linux" {
		t.Skip("only supported on linux")
	}

	r := make([]Fixture, len(plugins))
	for i := range plugins {
		r[i] = BuildFixture(plugins[i], flags|BuildModePlugin)
	}
	return r
}

var hasCgo = func() bool {
	out, err := exec.Command("go", "env", "CGO_ENABLED").CombinedOutput()
	if err != nil {
		panic(err)
	}
	if strings.TrimSpace(string(out)) != "1" {
		return false
	}
	_, err = exec.LookPath("gcc")
	return err == nil
}()

func MustHaveCgo(t *testing.T) {
	if !hasCgo {
		t.Skip("Cgo not enabled")
	}
}

func RegabiSupported() bool {
	// Tracks regabiSupported variable in ParseGOEXPERIMENT internal/buildcfg/exp.go
	switch {
	case !goversion.VersionAfterOrEqual(runtime.Version(), 1, 17): // < 1.17
		return false
	case goversion.VersionAfterOrEqual(runtime.Version(), 1, 17):
		return runtime.GOARCH == "amd64" && runtime.GOOS == "linux"
	default: // >= 1.18
		return runtime.GOARCH == "amd64"
	}
}
