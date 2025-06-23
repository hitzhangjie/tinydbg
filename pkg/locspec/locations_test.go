package locspec

import (
	"testing"
)

func parseLocationSpecNoError(t *testing.T, locstr string) LocationSpec {
	spec, err := Parse(locstr)
	if err != nil {
		t.Fatalf("Error parsing %q: %v", locstr, err)
	}
	return spec
}

func assertNormalLocationSpec(t *testing.T, locstr string, tgt NormalLocationSpec) {
	spec := parseLocationSpecNoError(t, locstr)

	nls, ok := spec.(*NormalLocationSpec)
	if !ok {
		t.Fatalf("Location %q: expected NormalLocationSpec got %#v", locstr, spec)
	}

	if nls.Base != tgt.Base {
		t.Fatalf("Location %q: expected 'Base' %q got %q", locstr, tgt.Base, nls.Base)
	}

	if nls.LineOffset != tgt.LineOffset {
		t.Fatalf("Location %q: expected 'LineOffset' %d got %d", locstr, tgt.LineOffset, nls.LineOffset)
	}

	if tgt.FuncBase == nil {
		return
	}

	if nls.FuncBase == nil {
		t.Fatalf("Location %q: expected non-nil 'FuncBase'", locstr)
	}

	if *(tgt.FuncBase) != *(nls.FuncBase) {
		t.Fatalf("Location %q: expected 'FuncBase':\n%#v\ngot:\n%#v", locstr, tgt.FuncBase, nls.FuncBase)
	}
}

func TestFunctionLocationParsing(t *testing.T) {
	// Function locations, simple package names, no line offset
	assertNormalLocationSpec(t, "proc.(*Process).Continue", NormalLocationSpec{"proc.(*Process).Continue", &FuncLocationSpec{PackageName: "proc", ReceiverName: "Process", BaseName: "Continue"}, -1})
	assertNormalLocationSpec(t, "proc.Process.Continue", NormalLocationSpec{"proc.Process.Continue", &FuncLocationSpec{PackageName: "proc", ReceiverName: "Process", BaseName: "Continue"}, -1})
	assertNormalLocationSpec(t, "proc.Continue", NormalLocationSpec{"proc.Continue", &FuncLocationSpec{PackageOrReceiverName: "proc", BaseName: "Continue"}, -1})
	assertNormalLocationSpec(t, "(*Process).Continue", NormalLocationSpec{"(*Process).Continue", &FuncLocationSpec{ReceiverName: "Process", BaseName: "Continue"}, -1})
	assertNormalLocationSpec(t, "Continue", NormalLocationSpec{"Continue", &FuncLocationSpec{BaseName: "Continue"}, -1})

	// Function locations, simple package names, line offsets
	assertNormalLocationSpec(t, "proc.(*Process).Continue:10", NormalLocationSpec{"proc.(*Process).Continue", &FuncLocationSpec{PackageName: "proc", ReceiverName: "Process", BaseName: "Continue"}, 10})
	assertNormalLocationSpec(t, "proc.Process.Continue:10", NormalLocationSpec{"proc.Process.Continue", &FuncLocationSpec{PackageName: "proc", ReceiverName: "Process", BaseName: "Continue"}, 10})
	assertNormalLocationSpec(t, "proc.Continue:10", NormalLocationSpec{"proc.Continue", &FuncLocationSpec{PackageOrReceiverName: "proc", BaseName: "Continue"}, 10})
	assertNormalLocationSpec(t, "(*Process).Continue:10", NormalLocationSpec{"(*Process).Continue", &FuncLocationSpec{ReceiverName: "Process", BaseName: "Continue"}, 10})
	assertNormalLocationSpec(t, "Continue:10", NormalLocationSpec{"Continue", &FuncLocationSpec{BaseName: "Continue"}, 10})

	// Function locations, package paths, no line offsets
	assertNormalLocationSpec(t, "github.com/hitzhangjie/tinydbg/pkg/proc.(*Process).Continue", NormalLocationSpec{"github.com/hitzhangjie/tinydbg/pkg/proc.(*Process).Continue", &FuncLocationSpec{PackageName: "github.com/hitzhangjie/tinydbg/pkg/proc", ReceiverName: "Process", BaseName: "Continue"}, -1})
	assertNormalLocationSpec(t, "github.com/hitzhangjie/tinydbg/pkg/proc.Process.Continue", NormalLocationSpec{"github.com/hitzhangjie/tinydbg/pkg/proc.Process.Continue", &FuncLocationSpec{PackageName: "github.com/hitzhangjie/tinydbg/pkg/proc", ReceiverName: "Process", BaseName: "Continue"}, -1})
	assertNormalLocationSpec(t, "github.com/hitzhangjie/tinydbg/pkg/proc.Continue", NormalLocationSpec{"github.com/hitzhangjie/tinydbg/pkg/proc.Continue", &FuncLocationSpec{PackageName: "github.com/hitzhangjie/tinydbg/pkg/proc", BaseName: "Continue"}, -1})

	// Function locations, package paths, line offsets
	assertNormalLocationSpec(t, "github.com/hitzhangjie/tinydbg/pkg/proc.(*Process).Continue:10", NormalLocationSpec{"github.com/hitzhangjie/tinydbg/pkg/proc.(*Process).Continue", &FuncLocationSpec{PackageName: "github.com/hitzhangjie/tinydbg/pkg/proc", ReceiverName: "Process", BaseName: "Continue"}, 10})
	assertNormalLocationSpec(t, "github.com/hitzhangjie/tinydbg/pkg/proc.Process.Continue:10", NormalLocationSpec{"github.com/hitzhangjie/tinydbg/pkg/proc.Process.Continue", &FuncLocationSpec{PackageName: "github.com/hitzhangjie/tinydbg/pkg/proc", ReceiverName: "Process", BaseName: "Continue"}, 10})
	assertNormalLocationSpec(t, "github.com/hitzhangjie/tinydbg/pkg/proc.Continue:10", NormalLocationSpec{"github.com/hitzhangjie/tinydbg/pkg/proc.Continue", &FuncLocationSpec{PackageName: "github.com/hitzhangjie/tinydbg/pkg/proc", BaseName: "Continue"}, 10})
}

func assertSubstitutePathEqual(t *testing.T, expected string, substituted string) {
	t.Helper()
	if expected != substituted {
		t.Errorf("Expected substitutedPath to be %s got %s instead", expected, substituted)
	}
}

func TestSubstitutePathUnix(t *testing.T) {
	// Relative paths mapping
	assertSubstitutePathEqual(t, "/my/asb/folder/relative/path", SubstitutePath("relative/path", [][2]string{{"", "/my/asb/folder/"}}))
	assertSubstitutePathEqual(t, "/already/abs/path", SubstitutePath("/already/abs/path", [][2]string{{"", "/my/asb/folder/"}}))
	assertSubstitutePathEqual(t, "relative/path", SubstitutePath("/my/asb/folder/relative/path", [][2]string{{"/my/asb/folder/", ""}}))
	assertSubstitutePathEqual(t, "/another/folder/relative/path", SubstitutePath("/another/folder/relative/path", [][2]string{{"/my/asb/folder/", ""}}))
	assertSubstitutePathEqual(t, "my/path", SubstitutePath("relative/path/my/path", [][2]string{{"relative/path", ""}}))
	assertSubstitutePathEqual(t, "/abs/my/path", SubstitutePath("/abs/my/path", [][2]string{{"abs/my", ""}}))

	// Absolute paths mapping
	assertSubstitutePathEqual(t, "/new/mapping/path", SubstitutePath("/original/path", [][2]string{{"/original", "/new/mapping"}}))
	assertSubstitutePathEqual(t, "/no/change/path", SubstitutePath("/no/change/path", [][2]string{{"/original", "/new/mapping"}}))
	assertSubstitutePathEqual(t, "/folder/should_not_be_replaced/path", SubstitutePath("/folder/should_not_be_replaced/path", [][2]string{{"should_not_be_replaced", ""}}))

	// Mix absolute and relative mapping
	assertSubstitutePathEqual(t, "/new/mapping/path", SubstitutePath("/original/path", [][2]string{{"", "/my/asb/folder/"}, {"/my/asb/folder/", ""}, {"/original", "/new/mapping"}}))
	assertSubstitutePathEqual(t, "/my/asb/folder/path", SubstitutePath("path", [][2]string{{"/original", "/new/mapping"}, {"", "/my/asb/folder/"}, {"/my/asb/folder/", ""}}))
	assertSubstitutePathEqual(t, "path", SubstitutePath("/my/asb/folder/path", [][2]string{{"/original", "/new/mapping"}, {"/my/asb/folder/", ""}, {"", "/my/asb/folder/"}}))
}

type tRule struct {
	from string
	to   string
}

type tCase struct {
	rules []tRule
	path  string
	res   string
}

func platformCases() []tCase {
	casesUnix := []tCase{
		// Should not depend on separator at the end of rule path
		{[]tRule{{"/tmp/path", "/new/path2"}}, "/tmp/path/file.go", "/new/path2/file.go"},
		{[]tRule{{"/tmp/path/", "/new/path2/"}}, "/tmp/path/file.go", "/new/path2/file.go"},
		{[]tRule{{"/tmp/path/", "/new/path2"}}, "/tmp/path/file.go", "/new/path2/file.go"},
		{[]tRule{{"/tmp/path", "/new/path2/"}}, "/tmp/path/file.go", "/new/path2/file.go"},
		// Should apply to directory prefixes
		{[]tRule{{"/tmp/path", "/new/path2"}}, "/tmp/path-2/file.go", "/tmp/path-2/file.go"},
		// Should apply to exact matches
		{[]tRule{{"/tmp/path/file.go", "/new/path2/file2.go"}}, "/tmp/path/file.go", "/new/path2/file2.go"},
		// First matched rule should be used
		{[]tRule{
			{"/tmp/path1", "/new/path1"},
			{"/tmp/path2", "/new/path2"},
			{"/tmp/path2", "/new/path3"}}, "/tmp/path2/file.go", "/new/path2/file.go"},
	}
	casesLinux := []tCase{
		// Should be case-sensitive
		{[]tRule{{"/tmp/path", "/new/path2"}}, "/TmP/path/file.go", "/TmP/path/file.go"},
	}
	casesCross := []tCase{
		{[]tRule{{"C:\\some\\repo", "/go/src/github.com/some/repo/"}}, `c:\some\repo\folder\file.go`, "/go/src/github.com/some/repo/folder/file.go"},
	}

	r := append(casesUnix, casesLinux...)
	r = append(r, casesCross...)

	return r
}

func TestSubstitutePath(t *testing.T) {
	for _, c := range platformCases() {
		subRules := [][2]string{}
		for _, r := range c.rules {
			subRules = append(subRules, [2]string{r.from, r.to})
		}
		res := SubstitutePath(c.path, subRules)
		if c.res != res {
			t.Errorf("terminal.SubstitutePath(%q) => %q, want %q", c.path, res, c.res)
		}
	}
}
