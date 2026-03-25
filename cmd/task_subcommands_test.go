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
	out, err := executeRootForTest([]string{"--help"}, "")
	if err != nil {
		t.Fatalf("expected no error from root help, got %v", err)
	}

	if !strings.Contains(out, "\n  task        Manage and run tasks\n") {
		t.Fatalf("expected task command in root help output, got: %s", out)
	}
	if !strings.Contains(out, "\n  self        Manage cast itself\n") {
		t.Fatalf("expected self command in root help output, got: %s", out)
	}
}

func TestTaskHelpIncludesSubcommands(t *testing.T) {
	out, err := executeRootForTest([]string{"task", "--help"}, "")
	if err != nil {
		t.Fatalf("expected no error from task help, got %v", err)
	}

	for _, sub := range []string{"add", "install", "update", "clear-cache", "run", "list", "exec"} {
		if !strings.Contains(out, "\n  "+sub+" ") {
			t.Fatalf("expected %s subcommand in task help output, got: %s", sub, out)
		}
	}
	if strings.Contains(out, "\n  job ") {
		t.Fatalf("did not expect job command under task namespace, got: %s", out)
	}
}

func TestJobHelpIncludesRunAndList(t *testing.T) {
	out, err := executeRootForTest([]string{"job", "--help"}, "")
	if err != nil {
		t.Fatalf("expected no error from job help, got %v", err)
	}

	for _, sub := range []string{"run", "list"} {
		if !strings.Contains(out, "\n  "+sub+" ") {
			t.Fatalf("expected %s subcommand in job help output, got: %s", sub, out)
		}
	}
}

func TestRootListRunsTaskWhenListTaskExists(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\ntasks:\n  list:\n    uses: shell\n    run: echo LIST_TASK_OVERRIDE\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"list", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("root list failed: %v", err)
	}
	if !strings.Contains(out, "LIST_TASK_OVERRIDE") {
		t.Fatalf("expected root list override output, got: %s", out)
	}
}

func TestTaskListDoesNotRunListTaskOverride(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\ntasks:\n  list:\n    uses: shell\n    run: echo SHOULD_NOT_RUN\n  other:\n    uses: shell\n    run: echo ok\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"task", "list", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("task list failed: %v", err)
	}

	if strings.Contains(out, "SHOULD_NOT_RUN") {
		t.Fatalf("expected task list to bypass list task override execution, got: %s", out)
	}
	if !strings.Contains(out, "list") || !strings.Contains(out, "other") {
		t.Fatalf("expected task list output to include task names, got: %s", out)
	}
}

func TestRootRunRunsTaskWhenRunTaskExists(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\ntasks:\n  run:\n    uses: shell\n    run: echo RUN_TASK_OVERRIDE\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"run", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("root run failed: %v", err)
	}
	if !strings.Contains(out, "RUN_TASK_OVERRIDE") {
		t.Fatalf("expected root run override output, got: %s", out)
	}
}

func TestTaskRunDoesNotRunRunTaskOverride(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\ntasks:\n  run:\n    uses: shell\n    run: echo SHOULD_NOT_RUN\n  demo:\n    uses: shell\n    run: echo DEMO_RUN\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"task", "run", "-p", projectFile, "demo"}, "")
	if err != nil {
		t.Fatalf("task run failed: %v", err)
	}

	if strings.Contains(out, "SHOULD_NOT_RUN") {
		t.Fatalf("expected task run to bypass run task override execution, got: %s", out)
	}
	if !strings.Contains(out, "DEMO_RUN") {
		t.Fatalf("expected demo task output, got: %s", out)
	}
}

func TestTaskAddWithFlagsWritesTask(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\ntasks: {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	_, err := executeRootForTest([]string{"task", "-p", projectFile, "add", "--name", "local-shell", "--uses", "shell", "--run", "echo hi"}, "")
	if err != nil {
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

	_, err := executeRootForTest([]string{"task", "-p", projectFile, "clear-cache"}, "")
	if err != nil {
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

func executeRootForTest(args []string, stdin string) (string, error) {
	resetRootForTest()
	clearDynamicSubcommands(rootCmd)
	projectOverride := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "-p" || args[i] == "--project" {
			if i+1 < len(args) {
				projectOverride = args[i+1]
			}
			break
		}
		if strings.HasPrefix(args[i], "--project=") {
			projectOverride = strings.TrimPrefix(args[i], "--project=")
			break
		}
	}
	if projectOverride != "" {
		_ = registerRootDynamicSubcommandsForProjectFile(projectOverride)
	}

	oldArgs := os.Args
	os.Args = append([]string{"cast"}, args...)
	defer func() {
		os.Args = oldArgs
	}()

	cmd := rootCmd
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)

	err := cmd.ExecuteContext(context.Background())
	return buf.String(), err
}
