package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/types"
)

func TestNormalizeSubcmdPath(t *testing.T) {
	got := normalizeSubcmdPath("::dn::nuget::")
	if got != "dn:nuget" {
		t.Fatalf("normalizeSubcmdPath returned %q, want dn:nuget", got)
	}
}

func TestBuildSubcmdTree(t *testing.T) {
	schema := &types.Project{}
	schema.Subcmds = []string{"dn", "dn:nuget"}
	schema.Tasks = types.NewTaskMap()

	schema.Tasks.Set(&types.Task{Id: "dn:test", Name: "dn:test", Uses: ptrString("shell"), Run: ptrString("echo test")})
	schema.Tasks.Set(&types.Task{Id: "dn:publish", Name: "dn:publish", Uses: ptrString("shell"), Run: ptrString("echo publish")})
	schema.Tasks.Set(&types.Task{Id: "dn:help", Name: "dn:help", Help: ptrString("dn help text"), Uses: ptrString("shell"), Run: ptrString("echo ignored")})
	schema.Tasks.Set(&types.Task{Id: "dn:nuget:pack", Name: "dn:nuget:pack", Uses: ptrString("shell"), Run: ptrString("echo pack")})

	root := buildSubcmdTree(schema)
	if root == nil {
		t.Fatalf("expected subcmd tree to be created")
	}

	dnNode, ok := root.children["dn"]
	if !ok {
		t.Fatalf("expected dn node")
	}
	if dnNode.tasks["test"] != "dn:test" {
		t.Fatalf("expected dn:test task mapping")
	}
	if dnNode.tasks["publish"] != "dn:publish" {
		t.Fatalf("expected dn:publish task mapping")
	}

	nugetNode, ok := dnNode.children["nuget"]
	if !ok {
		t.Fatalf("expected dn:nuget node")
	}
	if nugetNode.tasks["pack"] != "dn:nuget:pack" {
		t.Fatalf("expected dn:nuget:pack task mapping")
	}
}

func TestParseContextFromArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "short flag", args: []string{"run", "-c", "dev", "task"}, want: "dev"},
		{name: "long flag", args: []string{"--context", "qa", "task"}, want: "qa"},
		{name: "long flag equals", args: []string{"--context=prod", "task"}, want: "prod"},
		{name: "missing", args: []string{"task"}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseContextFromArgs(tc.args); got != tc.want {
				t.Fatalf("parseContextFromArgs(%v)=%q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func TestTaskHelpText(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: test
tasks:
  dn:help:
    uses: shell
    help: |
      DN HELP TEXT
    run: echo ignored
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	got := taskHelpText(projectFile, "dn:help")
	if !strings.Contains(got, "DN HELP TEXT") {
		t.Fatalf("expected help text, got %q", got)
	}
}

func ptrString(v string) *string {
	return &v
}
