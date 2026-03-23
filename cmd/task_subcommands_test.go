package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
)

func TestRootHelpIncludesTaskCommand(t *testing.T) {
	resetRootForTest()
	cmd := rootCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error from root help, got %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "\n  task        Manage and run tasks\n") {
		t.Fatalf("expected task command in root help output, got: %s", out)
	}
}

func TestTaskHelpIncludesSubcommands(t *testing.T) {
	resetRootForTest()
	cmd := rootCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"task", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error from task help, got %v", err)
	}

	out := buf.String()
	for _, sub := range []string{"add", "install", "update", "clear-cache", "run", "list", "exec"} {
		if !strings.Contains(out, "\n  "+sub+" ") {
			t.Fatalf("expected %s subcommand in task help output, got: %s", sub, out)
		}
	}
}

func TestTaskAddWithFlagsWritesTask(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\ntasks: {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	cmd := rootCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"task", "-p", projectFile, "add", "--name", "local-shell", "--uses", "shell", "--run", "echo hi"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("task add failed: %v", err)
	}

	content, err := os.ReadFile(projectFile)
	if err != nil {
		t.Fatalf("failed to read castfile: %v", err)
	}
	out := string(content)
	if !strings.Contains(out, "local-shell:") || !strings.Contains(out, "uses: shell") || !strings.Contains(out, "run: echo hi") {
		t.Fatalf("expected added task in castfile, got: %s", out)
	}
}

func TestTaskClearCacheRemovesLocalCache(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	projectFile := filepath.Join(projectDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	cacheDir := projects.ResolveVolatileRemoteTasksDir(projectDir)
	if err := os.MkdirAll(filepath.Join(cacheDir, "dummy"), 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	cmd := rootCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"task", "-p", projectFile, "clear-cache"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("task clear-cache failed: %v", err)
	}

	if _, err := os.Stat(cacheDir); err == nil {
		t.Fatalf("expected local cache dir to be removed")
	}
}

func resetRootForTest() {
	rootCmd.SetArgs(nil)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
}
