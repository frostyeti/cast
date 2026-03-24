package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDynamicSubcmds_HelpAndRun(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: subcmd-test
subcmds:
  - mysql
tasks:
  mysql:help:
    uses: shell
    help: |
      MYSQL HELP BLOCK
      usage: cast mysql <command>
    run: echo should-not-run
  mysql:test:
    uses: shell
    run: echo MYSQL_TEST_OK $1 $2
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"-p", projectFile, "mysql", "--help"}, "")
	if err != nil {
		t.Fatalf("mysql --help failed: %v", err)
	}
	if !strings.Contains(out, "MYSQL HELP BLOCK") {
		t.Fatalf("expected mysql help block output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"-p", projectFile, "mysql", "test"}, "")
	if err != nil {
		t.Fatalf("mysql test failed: %v; output: %s", err, out)
	}
	if !strings.Contains(out, "MYSQL_TEST_OK") {
		t.Fatalf("expected mysql test task output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"-p", projectFile, "mysql", "test", "--clean", "./path/to/file"}, "")
	if err != nil {
		t.Fatalf("mysql test with args failed: %v", err)
	}
	if !strings.Contains(out, "MYSQL_TEST_OK") {
		t.Fatalf("expected args passthrough for shell command, got: %s", out)
	}
	if !strings.Contains(out, "--clean") || !strings.Contains(out, "./path/to/file") {
		t.Fatalf("expected passthrough args to appear in output, got: %s", out)
	}
}

func TestDynamicSubcmds_TaskHelpFlag(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: subcmd-help-test
subcmds:
  - mysql
tasks:
  mysql:test:
    uses: shell
    help: |
      MYSQL TEST HELP TEXT
    run: echo SHOULD_NOT_RUN
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"-p", projectFile, "mysql", "test", "--help"}, "")
	if err != nil {
		t.Fatalf("mysql test --help failed: %v", err)
	}
	if !strings.Contains(out, "MYSQL TEST HELP TEXT") {
		t.Fatalf("expected task help output, got: %s", out)
	}
	if strings.Contains(out, "SHOULD_NOT_RUN") {
		t.Fatalf("expected help mode to avoid task execution, got: %s", out)
	}
}

func TestDynamicSubcmds_NestedTaskHelpFlag(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: nested-subcmd-help-test
subcmds:
  - test
tasks:
  test:bun:
    uses: shell
    help: |
      BUN HELP TEXT
    run: echo SHOULD_NOT_RUN
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"-p", projectFile, "test", "bun", "--help"}, "")
	if err != nil {
		t.Fatalf("test bun --help failed: %v", err)
	}
	if !strings.Contains(out, "BUN HELP TEXT") {
		t.Fatalf("expected nested task help output, got: %s", out)
	}
	if strings.Contains(out, "SHOULD_NOT_RUN") {
		t.Fatalf("expected help mode to avoid task execution, got: %s", out)
	}
}

func TestDirectTaskRunArgPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: direct-task-args
tasks:
  deno:test:
    uses: shell
    run: echo deno test -A $1 $2
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"run", "-p", projectFile, "deno:test", "--clean", "./path/to/file"}, "")
	if err != nil {
		t.Fatalf("direct task run failed: %v", err)
	}
	if !strings.Contains(out, "deno test -A") {
		t.Fatalf("expected base shell command in output, got: %s", out)
	}
	if !strings.Contains(out, "--clean") || !strings.Contains(out, "./path/to/file") {
		t.Fatalf("expected direct task arg passthrough, got: %s", out)
	}
}

func TestDynamicSubcmds_ContextFlagOverridesEnv(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: subcmd-context
subcmds:
  - mysql
tasks:
  mysql:test:
    uses: shell
    run: sh -c 'echo CTX=$CAST_CONTEXT'
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	old := os.Getenv("CAST_CONTEXT")
	_ = os.Setenv("CAST_CONTEXT", "dev")
	defer func() {
		_ = os.Setenv("CAST_CONTEXT", old)
	}()

	out, err := executeRootForTest([]string{"-p", projectFile, "-c", "prod", "mysql", "test"}, "")
	if err != nil {
		t.Fatalf("mysql test with context override failed: %v", err)
	}
	if !strings.Contains(out, "CTX=prod") {
		t.Fatalf("expected context override to apply, got: %s", out)
	}
}
