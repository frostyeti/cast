package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanAppendShellArgs(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "cmd.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	relScript := "cmd.sh"
	cases := []struct {
		name string
		run  string
		cwd  string
		want bool
	}{
		{name: "single line command", run: "deno test -A", cwd: "", want: true},
		{name: "multiline script", run: "echo one\necho two", cwd: "", want: false},
		{name: "operator script", run: "echo one && echo two", cwd: "", want: false},
		{name: "script file path", run: scriptPath, cwd: "", want: true},
		{name: "relative script file", run: relScript, cwd: tmpDir, want: true},
		{name: "positional vars allowed", run: "echo $1 $2", cwd: "", want: true},
		{name: "named vars blocked", run: "echo $CAST_CONTEXT", cwd: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canAppendShellArgs(tc.run, tc.cwd); got != tc.want {
				t.Fatalf("canAppendShellArgs(%q)=%v, want %v", tc.run, got, tc.want)
			}
		})
	}
}
