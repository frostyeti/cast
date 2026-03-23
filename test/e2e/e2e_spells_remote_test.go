package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_RemoteSpellFetchNoticeAndBunInputs(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".cast", "cache"), 0o755); err != nil {
		t.Fatalf("failed to create local .cast cache dir: %v", err)
	}

	castfilePath := filepath.Join(projectDir, "castfile")
	castfile := `
name: spell fetch notice e2e
trusted_sources:
  - "spell:"
tasks:
  hello:
    uses: "spell:examples/hello-world@head"

  bun-inputs:
    uses: "spell:examples/bun-with-inputs@head"
    with:
      greeting: "Hello Bun"
      language: "bun"
`
	if err := os.WriteFile(castfilePath, []byte(castfile), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	firstRun := exec.Command("timeout", "90", binPath, "hello")
	firstRun.Dir = projectDir
	firstOutput, err := firstRun.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run hello task: %v\n%s", err, string(firstOutput))
	}

	firstOutStr := string(firstOutput)
	fetchNotice := "Fetching task: spell:examples/hello-world@head"
	if !strings.Contains(firstOutStr, fetchNotice+"\n") || !strings.Contains(firstOutStr, "\n\nhello world") {
		t.Fatalf("expected fetch notice and blank separator before task output, got: %s", firstOutStr)
	}
	if !strings.Contains(firstOutStr, "hello world") {
		t.Fatalf("expected hello-world task output, got: %s", firstOutStr)
	}

	secondRun := exec.Command("timeout", "90", binPath, "hello")
	secondRun.Dir = projectDir
	secondOutput, err := secondRun.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to rerun hello task: %v\n%s", err, string(secondOutput))
	}

	secondOutStr := string(secondOutput)
	if strings.Contains(secondOutStr, fetchNotice) {
		t.Fatalf("expected no fetch notice on cached run, got: %s", secondOutStr)
	}
	if !strings.Contains(secondOutStr, "hello world") {
		t.Fatalf("expected hello-world output on cached run, got: %s", secondOutStr)
	}

	bunRun := exec.Command("timeout", "90", binPath, "bun-inputs")
	bunRun.Dir = projectDir
	bunOutput, bunErr := bunRun.CombinedOutput()
	if bunErr != nil {
		if !strings.Contains(string(bunOutput), "No such file or directory") &&
			!strings.Contains(string(bunOutput), "executable file not found") &&
			!strings.Contains(string(bunOutput), "command not found") {
			t.Fatalf("failed to run bun-inputs task: %v\n%s", bunErr, string(bunOutput))
		}
		t.Skipf("bun runtime unavailable in test environment: %s", string(bunOutput))
	}

	bunOutStr := string(bunOutput)
	if !strings.Contains(bunOutStr, "Hello Bun from bun") {
		t.Fatalf("expected bun-with-inputs output, got: %s", bunOutStr)
	}
}
