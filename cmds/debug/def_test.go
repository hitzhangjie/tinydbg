package debug

import (
	"testing"
)

// break: break [--name=|-n=<name>]  [locspec] [if condition]
//
// Following input is valid:
// break main.main
// break main.go:13
// break 12345678
// break main.main if 1==2
// break main.go:13 if 1==2
// break if 1==2
// break 12345678 if 1==2
// break main.main if 1 == 2
// break --name=bp main.main if 1 == 2
// break --name bp main.main if 1 == 2
// break -n=bp main.main if 1 == 2
// break -n bp main.main if 1 == 2
func TestRegexp(t *testing.T) {
	type testCase struct {
		argstr  string
		name    string
		spec    string
		cond    string
		wantErr bool
	}
	cases := []testCase{
		{
			argstr:  "main.main",
			spec:    "main.main",
			cond:    "",
			wantErr: false,
		},
		{
			argstr:  "main.go:13",
			spec:    "main.go:13",
			cond:    "",
			wantErr: false,
		},
		{
			argstr:  "12345678",
			spec:    "12345678",
			cond:    "",
			wantErr: false,
		},
		{
			argstr:  "main.main if 1==2",
			spec:    "main.main",
			cond:    "1==2",
			wantErr: false,
		},
		{
			argstr:  "main.go:13 if 1==2",
			spec:    "main.go:13",
			cond:    "1==2",
			wantErr: false,
		},
		{
			argstr:  "if 1==2",
			spec:    "",
			cond:    "1==2",
			wantErr: false,
		},
		{
			argstr:  "12345678 if 1==2",
			spec:    "12345678",
			cond:    "1==2",
			wantErr: false,
		},
		{
			argstr:  "main.main if 1 == 2",
			spec:    "main.main",
			cond:    "1 == 2",
			wantErr: false,
		},
		{
			argstr:  "--name=bp main.main if 1 == 2",
			name:    "bp",
			spec:    "main.main",
			cond:    "1 == 2",
			wantErr: false,
		},
		{
			argstr:  "--name bp main.main if 1 == 2",
			name:    "bp",
			spec:    "main.main",
			cond:    "1 == 2",
			wantErr: false,
		},
		{
			argstr:  "-n=bp main.main if 1 == 2",
			name:    "bp",
			spec:    "main.main",
			cond:    "1 == 2",
			wantErr: false,
		},
		{
			argstr:  "-n bp main.main if 1 == 2",
			name:    "bp",
			spec:    "main.main",
			cond:    "1 == 2",
			wantErr: false,
		},
	}
	for _, c := range cases {
		name, spec, cond, err := parseBreakpointArgs(c.argstr)
		if (err != nil) != c.wantErr {
			t.Errorf("argstr: %s, want error: %v, got err: %v", c.argstr, c.wantErr, err)
		}
		if name != c.name {
			t.Errorf("argstr: %s, want name: %s, got %s", c.argstr, c.name, name)
		}
		if spec != c.spec {
			t.Errorf("argstr: %s, want spec: %s, got %s", c.argstr, c.spec, spec)
		}
		if cond != c.cond {
			t.Errorf("argstr: %s, want cond: %s, got %s", c.argstr, c.cond, cond)
		}
	}
}
