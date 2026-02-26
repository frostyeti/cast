package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_DotnetRemoteTasks(t *testing.T) {
	// 1. Build the cast binary
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	// 2. We'll use the existing test fixtures for the remote tasks
	pwd, _ := os.Getwd()
	fixtureGitRepo := filepath.Join(pwd, "fixtures", "cast-dotnet-git")

	// Create individual git repos for each task to work around remote.go's lack of file:// subpath support
	tasks := []string{"build", "test", "pack", "publish", "clean"}
	repos := make(map[string]string)

	for _, task := range tasks {
		repoPath := filepath.Join(tmpDir, "repo-"+task)
		os.MkdirAll(repoPath, 0755)
		exec.Command("cp", "-r", filepath.Join(fixtureGitRepo, task)+"/.", repoPath).Run()

		runGit(t, repoPath, "init")
		runGit(t, repoPath, "config", "user.name", "Test User")
		runGit(t, repoPath, "config", "user.email", "test@example.com")
		runGit(t, repoPath, "add", ".")
		runGit(t, repoPath, "commit", "-m", "Initial commit")
		runGit(t, repoPath, "tag", "v1.0.0")
		repos[task] = repoPath
	}

	// 3. Create the project that uses the remote task
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(projectDir, 0755)

	// Create a dummy dotnet project
	t.Log("Creating dummy .NET project...")
	runCmd := exec.Command("dotnet", "new", "console", "-n", "DemoApp")
	runCmd.Dir = projectDir
	if output, err := runCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create dotnet project: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(projectDir, "castfile")

	yamlData := `
name: Dotnet Tasks Project
trusted_sources:
  - "file://"
tasks:
  build:
    uses: "file://` + filepath.ToSlash(repos["build"]) + `@v1.0.0"
    with:
      configuration: "Release"
      no-restore: "true"
      project: "DemoApp"
  
  test:
    uses: "file://` + filepath.ToSlash(repos["test"]) + `@v1.0.0"
    with:
      configuration: "Release"
      no-build: "true"
      project: "DemoApp"

  pack:
    uses: "file://` + filepath.ToSlash(repos["pack"]) + `@v1.0.0"
    with:
      configuration: "Release"
      no-build: "true"
      project: "DemoApp"
      output: "./nupkgs"
      
  publish:
    uses: "file://` + filepath.ToSlash(repos["publish"]) + `@v1.0.0"
    with:
      configuration: "Release"
      project: "DemoApp"
      output: "./publish"
      
  clean:
    uses: "file://` + filepath.ToSlash(repos["clean"]) + `@v1.0.0"
    with:
      configuration: "Release"
      project: "DemoApp"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	// 4. Run the tasks

	t.Log("Running dotnet build task...")
	runCmd = exec.Command("timeout", "30", binPath, "build")
	runCmd.Dir = projectDir
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast build task: %v\n%s", err, string(output))
	}
	outStr := string(output)
	if !strings.Contains(outStr, "Running: dotnet build DemoApp -c Release") {
		t.Errorf("expected build output to contain 'Running: dotnet build DemoApp -c Release', got: %s", outStr)
	}

	t.Log("Running dotnet test task...")
	runCmd = exec.Command("timeout", "30", binPath, "test")
	runCmd.Dir = projectDir
	output, err = runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast test task: %v\n%s", err, string(output))
	}
	outStr = string(output)
	if !strings.Contains(outStr, "Running: dotnet test DemoApp -c Release") {
		t.Errorf("expected test output to contain 'Running: dotnet test DemoApp -c Release', got: %s", outStr)
	}

	t.Log("Running dotnet pack task...")
	runCmd = exec.Command("timeout", "30", binPath, "pack")
	runCmd.Dir = projectDir
	output, err = runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast pack task: %v\n%s", err, string(output))
	}
	outStr = string(output)
	if !strings.Contains(outStr, "Running: dotnet pack DemoApp -c Release -o ./nupkgs") {
		t.Errorf("expected pack output to contain 'Running: dotnet pack DemoApp -c Release -o ./nupkgs', got: %s", outStr)
	}

	t.Log("Running dotnet publish task...")
	runCmd = exec.Command("timeout", "30", binPath, "publish")
	runCmd.Dir = projectDir
	output, err = runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast publish task: %v\n%s", err, string(output))
	}
	outStr = string(output)
	if !strings.Contains(outStr, "Running: dotnet publish DemoApp -c Release -o ./publish") {
		t.Errorf("expected publish output to contain 'Running: dotnet publish DemoApp -c Release -o ./publish', got: %s", outStr)
	}

	t.Log("Running dotnet clean task...")
	runCmd = exec.Command("timeout", "30", binPath, "clean")
	runCmd.Dir = projectDir
	output, err = runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run cast clean task: %v\n%s", err, string(output))
	}
	outStr = string(output)
	if !strings.Contains(outStr, "Running: dotnet clean DemoApp -c Release") {
		t.Errorf("expected clean output to contain 'Running: dotnet clean DemoApp -c Release', got: %s", outStr)
	}
}
