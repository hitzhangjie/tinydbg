package debugger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hitzhangjie/dlv/pkg/gobuild"
	protest "github.com/hitzhangjie/dlv/pkg/proc/test"
	"github.com/hitzhangjie/dlv/service/api"
)

func TestDebugger_LaunchNoMain(t *testing.T) {
	fixturesDir := protest.FindFixturesDir()
	nomaindir := filepath.Join(fixturesDir, "nomaindir")
	debugname := "debug"
	exepath := filepath.Join(nomaindir, debugname)
	defer os.Remove(exepath)
	if err := gobuild.GoBuild(debugname, []string{nomaindir}); err != nil {
		t.Fatalf("go build error %v", err)
	}

	d := new(Debugger)
	_, err := d.Launch([]string{exepath}, ".")
	if err == nil {
		t.Fatalf("expected error but none was generated")
	}
	if err != api.ErrNotExecutable {
		t.Fatalf("expected error \"%v\" got \"%v\"", api.ErrNotExecutable, err)
	}
}
