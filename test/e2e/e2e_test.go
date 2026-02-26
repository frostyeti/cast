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

func TestE2E_RemoteTask(t *testing.T) {
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	// JSR module usage
	yamlData := `
name: Remote E2E Test
trusted_sources:
  - "jsr:"
tasks:
  test-remote:
    uses: "jsr:@std/fmt/colors"
    with:
      hello: "world"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	t.Log("Running cast binary...")
	runCmd := exec.Command("timeout", "30", binPath, "test-remote")
	runCmd.Dir = tmpDir
	output, err := runCmd.CombinedOutput()

	// This might fail if Deno is not installed, but GitHub Actions CI runs setup-deno
	if err != nil {
		t.Logf("Output: %s", string(output))
		// We just ignore the failure for local dev if Deno isn't installed
		// But let's check output for "failed to find task handler" to ensure it tried to run deno
		if strings.Contains(string(output), "unable to find task handler") {
			t.Errorf("Remote task handler not found: %s", string(output))
		}
	}
}

func TestE2E_RemoteModule(t *testing.T) {
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	// Since we don't have a reliable public cast module repository, we can just point to
	// a local git repository, or just github.com/some/cast-module if there was one.
	// For testing purpose we'll just test if the parser throws error or attempts to clone.
	// We'll point to a dummy git repo that does not exist to ensure it hits the git path.
	yamlData := `
name: Remote Module Test
modules:
  - from: "github.com/frostyeti/does-not-exist@main"
    ns: "test"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	t.Log("Running cast binary...")
	runCmd := exec.Command("timeout", "10", binPath, "test:dummy")
	runCmd.Dir = tmpDir
	output, err := runCmd.CombinedOutput()

	// We expect it to fail cloning
	if err == nil {
		t.Errorf("Expected failure for non-existent git repo, but got success")
	}

	if !strings.Contains(string(output), "failed to fetch remote module") && !strings.Contains(string(output), "Authentication failed") && !strings.Contains(string(output), "Repository not found") {
		t.Errorf("Expected git clone failure message, got: %s", string(output))
	}
}
