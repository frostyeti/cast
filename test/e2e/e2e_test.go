package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_BasicTask(t *testing.T) {
	// 1. Build the cast binary
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}
	t.Log("Build complete.")

	// 2. Create a temporary project with a simple task
	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	yamlData := `
name: E2E Test
tasks:
  say-hello:
    uses: shell
    run: echo "Hello E2E"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	// 3. Run the cast binary
	t.Log("Running cast binary...")
	runCmd := exec.Command("timeout", "5", binPath, "say-hello")
	runCmd.Dir = tmpDir
	output, err := runCmd.CombinedOutput()
	t.Logf("Run complete. Error: %v", err)
	if err != nil {
		t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
	}

	// 4. Verify output
	t.Log("Verifying output...")
	if !strings.Contains(string(output), "Hello E2E") {
		t.Errorf("expected output to contain 'Hello E2E', got: %s", string(output))
	}
}

func TestE2E_DockerTask(t *testing.T) {
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	yamlData := `
name: Docker E2E Test
tasks:
  test-docker:
    uses: docker
    with:
      image: alpine:latest
    run: echo "Hello from Docker E2E"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	t.Log("Running cast binary...")
	runCmd := exec.Command("timeout", "30", binPath, "test-docker")
	runCmd.Dir = tmpDir
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "Hello from Docker E2E") {
		t.Errorf("expected output to contain 'Hello from Docker E2E', got: %s", string(output))
	}
}
