package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_RemoteCastTaskWithSemver(t *testing.T) {
	// 1. Build the cast binary
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	// 2. Setup a dummy "remote" Git repository
	t.Log("Setting up dummy remote Git repository...")
	repoDir := filepath.Join(tmpDir, "remote-repo")
	os.MkdirAll(repoDir, 0755)

	// init git
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	// create cast.task
	casttaskYaml := `
name: Semver Task
description: A test remote task
inputs:
  greeting:
    description: Greeting message
    required: true
runs:
  using: deno
  main: mod.ts
`
	if err := os.WriteFile(filepath.Join(repoDir, "cast.task"), []byte(casttaskYaml), 0644); err != nil {
		t.Fatalf("failed to write cast.task: %v", err)
	}

	// create mod.ts
	modTs := `
export function run() {
	const greeting = Deno.env.get("INPUT_GREETING");
	console.log(greeting + " from v1.2.3");
}
`
	if err := os.WriteFile(filepath.Join(repoDir, "mod.ts"), []byte(modTs), 0644); err != nil {
		t.Fatalf("failed to write mod.ts: %v", err)
	}

	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "Initial commit")
	runGit(t, repoDir, "tag", "v1.2.3") // Semantic version tag

	// 3. Create the project that uses the remote task
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(projectDir, 0755)

	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(projectDir, "castfile")
	// Using file:// protocol which should go through our remote logic
	repoUrl := "file://" + filepath.ToSlash(repoDir)
	yamlData := `
name: Caller Project
trusted_sources:
  - "file://"
tasks:
  call-remote:
    uses: "` + repoUrl + `@v1"  # Uses semver resolution ~v1.x.x -> v1.2.3
    with:
      greeting: "Hello Semver"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	// 4. Run the task
	t.Log("Running cast binary...")
	runCmd := exec.Command("timeout", "30", binPath, "call-remote")
	runCmd.Dir = projectDir
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
	}

	outStr := string(output)
	if !strings.Contains(outStr, "Hello Semver from v1.2.3") {
		t.Errorf("expected output to contain 'Hello Semver from v1.2.3', got: %s", outStr)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
