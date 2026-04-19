package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_WorkspaceAliasDeepNesting(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	rootDir := filepath.Join(tmpDir, "workspace-root")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	var builder strings.Builder
	builder.WriteString("name: workspace-root\nworkspace:\n  include:\n    - apps/**\n  aliases:\n")

	for i := 1; i <= 24; i++ {
		segments := []string{"apps", fmt.Sprintf("layer-%02d", i), fmt.Sprintf("service-%02d", i)}
		serviceDir := filepath.Join(append([]string{rootDir}, segments...)...)
		if err := os.MkdirAll(serviceDir, 0o755); err != nil {
			t.Fatalf("mkdir service %d: %v", i, err)
		}
		castfile := fmt.Sprintf("name: svc-%02d\ntasks:\n  whoami:\n    uses: shell\n    run: echo svc-%02d\n", i, i)
		if err := os.WriteFile(filepath.Join(serviceDir, "castfile"), []byte(castfile), 0o644); err != nil {
			t.Fatalf("write nested castfile %d: %v", i, err)
		}
		if i == 24 {
			builder.WriteString(fmt.Sprintf("    deep-service: %s\n", filepath.ToSlash(filepath.Join(segments...))))
		}
	}

	if err := os.WriteFile(filepath.Join(rootDir, "castfile"), []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	cmd := exec.Command("timeout", "15", binPath, "-p", filepath.Join(rootDir, "castfile"), "@deep-service", "whoami")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("workspace alias run failed: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "svc-24") {
		t.Fatalf("expected deep workspace task output, got: %s", string(out))
	}
}
