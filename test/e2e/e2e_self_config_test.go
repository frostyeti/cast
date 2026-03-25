package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_SelfConfig_ContextDefault(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	projectFile := filepath.Join(tmpDir, "castfile")
	content := `
name: self-config-context
tasks:
  show:
    uses: shell
    run: echo "CTX=$CAST_CONTEXT"
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	setCmd := exec.Command("timeout", "10", binPath, "self", "config", "set", "-p", projectFile, "context", "prod")
	setCmd.Dir = tmpDir
	if out, err := setCmd.CombinedOutput(); err != nil {
		t.Fatalf("self config set failed: %v\n%s", err, string(out))
	}

	runCmd := exec.Command("timeout", "10", binPath, "-p", projectFile, "show")
	runCmd.Dir = tmpDir
	out, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run task failed: %v\n%s", err, string(out))
	}

	if !strings.Contains(string(out), "CTX=prod") {
		t.Fatalf("expected config context default to apply, got: %s", string(out))
	}
}
