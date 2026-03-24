package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectTaskHelpPrefersTaskHelpText(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: direct-help
tasks:
  test:bun:
    uses: shell
    help: |
      BUN TASK HELP
    desc: Bun task description
    run: echo SHOULD_NOT_RUN
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"-p", projectFile, "test:bun", "--help"}, "")
	if err != nil {
		t.Fatalf("direct task --help failed: %v", err)
	}

	if !strings.Contains(out, "BUN TASK HELP") {
		t.Fatalf("expected task help output, got: %s", out)
	}
	if strings.Contains(out, "SHOULD_NOT_RUN") {
		t.Fatalf("expected help interception to avoid task execution, got: %s", out)
	}
}

func TestDirectTaskHelpFallsBackToDesc(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: direct-help-desc
tasks:
  test:bun:
    uses: shell
    desc: Bun task description fallback
    run: echo SHOULD_NOT_RUN
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"-p", projectFile, "test:bun", "--help"}, "")
	if err != nil {
		t.Fatalf("direct task --help failed: %v", err)
	}

	if !strings.Contains(out, "Bun task description fallback") {
		t.Fatalf("expected task desc output when help absent, got: %s", out)
	}
	if strings.Contains(out, "SHOULD_NOT_RUN") {
		t.Fatalf("expected help interception to avoid task execution, got: %s", out)
	}
}
