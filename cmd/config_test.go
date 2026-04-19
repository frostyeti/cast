package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootConfigSetGetRm_BuiltIn(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	if _, err := executeRootForTest([]string{"config", "set", "-p", projectFile, "context", "prod"}, ""); err != nil {
		t.Fatalf("root config set failed: %v", err)
	}

	out, err := executeRootForTest([]string{"config", "get", "-p", projectFile, "context"}, "")
	if err != nil {
		t.Fatalf("root config get failed: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected root config get context output, got: %s", out)
	}

	if _, err := executeRootForTest([]string{"config", "rm", "-p", projectFile, "context"}, ""); err != nil {
		t.Fatalf("root config rm failed: %v", err)
	}

	_, err = executeRootForTest([]string{"config", "get", "-p", projectFile, "context"}, "")
	if err == nil {
		t.Fatalf("expected get for removed context key to fail")
	}
}

func TestRootConfig_OverridesWithConfigTask(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: test\ntasks:\n  config:\n    uses: shell\n    run: echo CONFIG_TASK_OVERRIDE\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"config", "-p", projectFile, "set", "context", "prod"}, "")
	if err != nil {
		t.Fatalf("root config override command failed: %v", err)
	}
	if !strings.Contains(out, "CONFIG_TASK_OVERRIDE") {
		t.Fatalf("expected config task override output, got: %s", out)
	}
}

func TestRootConfig_OverridesWithConfigSubcmds(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: test\nsubcmds:\n  - config\ntasks:\n  config:help:\n    uses: shell\n    help: |\n      CONFIG SUBCMD HELP\n    run: echo SHOULD_NOT_RUN\n  config:set:\n    uses: shell\n    run: echo CONFIG_SET_OVERRIDE\n  config:get:\n    uses: shell\n    run: echo CONFIG_GET_OVERRIDE\n  config:rm:\n    uses: shell\n    run: echo CONFIG_RM_OVERRIDE\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"config", "-p", projectFile, "--help"}, "")
	if err != nil {
		t.Fatalf("root config --help failed: %v", err)
	}
	if !strings.Contains(out, "CONFIG SUBCMD HELP") {
		t.Fatalf("expected config:help output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"config", "-p", projectFile, "set", "context", "prod"}, "")
	if err != nil {
		t.Fatalf("root config set override failed: %v", err)
	}
	if !strings.Contains(out, "CONFIG_SET_OVERRIDE") {
		t.Fatalf("expected config:set override output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"config", "-p", projectFile, "get", "context"}, "")
	if err != nil {
		t.Fatalf("root config get override failed: %v", err)
	}
	if !strings.Contains(out, "CONFIG_GET_OVERRIDE") {
		t.Fatalf("expected config:get override output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"config", "-p", projectFile, "rm", "context"}, "")
	if err != nil {
		t.Fatalf("root config rm override failed: %v", err)
	}
	if !strings.Contains(out, "CONFIG_RM_OVERRIDE") {
		t.Fatalf("expected config:rm override output, got: %s", out)
	}
}

func TestSelfConfig_NotOverriddenByRootConfigTask(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: test\ntasks:\n  config:\n    uses: shell\n    run: echo ROOT_CONFIG_OVERRIDE\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	if _, err := executeRootForTest([]string{"self", "config", "set", "-p", projectFile, "context", "prod"}, ""); err != nil {
		t.Fatalf("self config set should not be overridden, got: %v", err)
	}

	out, err := executeRootForTest([]string{"self", "config", "get", "-p", projectFile, "context"}, "")
	if err != nil {
		t.Fatalf("self config get should not be overridden, got: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected self config value output, got: %s", out)
	}
	if strings.Contains(out, "ROOT_CONFIG_OVERRIDE") {
		t.Fatalf("expected self config to bypass root config override task, got: %s", out)
	}
}

func TestRootContextSetGetShowRm_BuiltIn(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	if _, err := executeRootForTest([]string{"context", "use", "-p", projectFile, "prod"}, ""); err != nil {
		t.Fatalf("root context use failed: %v", err)
	}

	out, err := executeRootForTest([]string{"context", "get", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("root context get failed: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected root context get output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"context", "show", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("root context show failed: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected root context show output, got: %s", out)
	}

	if _, err := executeRootForTest([]string{"context", "rm", "-p", projectFile}, ""); err != nil {
		t.Fatalf("root context rm failed: %v", err)
	}
}

func TestRootContext_OverridesWithContextTask(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: test\ntasks:\n  context:\n    uses: shell\n    run: echo CONTEXT_TASK_OVERRIDE\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"context", "use", "-p", projectFile, "prod"}, "")
	if err != nil {
		t.Fatalf("root context override command failed: %v", err)
	}
	if !strings.Contains(out, "CONTEXT_TASK_OVERRIDE") {
		t.Fatalf("expected context task override output, got: %s", out)
	}
}

func TestSelfContext_NotOverriddenByRootContextTask(t *testing.T) {
	resetRootForTest()
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: test\ntasks:\n  context:\n    uses: shell\n    run: echo ROOT_CONTEXT_OVERRIDE\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	if _, err := executeRootForTest([]string{"self", "context", "use", "-p", projectFile, "prod"}, ""); err != nil {
		t.Fatalf("self context use should not be overridden, got: %v", err)
	}

	out, err := executeRootForTest([]string{"self", "context", "get", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("self context get should not be overridden, got: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected self context value output, got: %s", out)
	}
	if strings.Contains(out, "ROOT_CONTEXT_OVERRIDE") {
		t.Fatalf("expected self context to bypass root context override task, got: %s", out)
	}
}
