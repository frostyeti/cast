package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelfConfigSetGetRmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: demo\n"), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	if _, err := executeRootForTest([]string{"self", "config", "set", "-p", projectFile, "context", "prod"}, ""); err != nil {
		t.Fatalf("self config set context failed: %v", err)
	}
	if _, err := executeRootForTest([]string{"self", "config", "set", "-p", projectFile, "feature_flags", "[\"alpha\"]"}, ""); err != nil {
		t.Fatalf("self config set custom failed: %v", err)
	}

	out, err := executeRootForTest([]string{"self", "config", "get", "-p", projectFile, "context"}, "")
	if err != nil {
		t.Fatalf("self config get context failed: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected context value in output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"self", "config", "get", "-p", projectFile, "feature_flags"}, "")
	if err != nil {
		t.Fatalf("self config get custom failed: %v", err)
	}
	if !strings.Contains(out, "alpha") {
		t.Fatalf("expected custom value in output, got: %s", out)
	}

	if _, err := executeRootForTest([]string{"self", "config", "rm", "-p", projectFile, "feature_flags"}, ""); err != nil {
		t.Fatalf("self config rm custom failed: %v", err)
	}

	_, err = executeRootForTest([]string{"self", "config", "get", "-p", projectFile, "feature_flags"}, "")
	if err == nil {
		t.Fatalf("expected get on removed key to fail")
	}
}

func TestSelfContextUseGetShowRmCommands(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: demo\n"), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	if _, err := executeRootForTest([]string{"self", "context", "use", "-p", projectFile, "prod"}, ""); err != nil {
		t.Fatalf("self context use failed: %v", err)
	}

	out, err := executeRootForTest([]string{"self", "context", "get", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("self context get failed: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected context value in output, got: %s", out)
	}

	out, err = executeRootForTest([]string{"self", "context", "show", "-p", projectFile}, "")
	if err != nil {
		t.Fatalf("self context show failed: %v", err)
	}
	if !strings.Contains(out, "prod") {
		t.Fatalf("expected context show output, got: %s", out)
	}

	if _, err := executeRootForTest([]string{"self", "context", "rm", "-p", projectFile}, ""); err != nil {
		t.Fatalf("self context rm failed: %v", err)
	}
}
