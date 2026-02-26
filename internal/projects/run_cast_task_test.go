package projects_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
)

func TestCastCrossProjectTask(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create target project
	targetDir := filepath.Join(tmpDir, "target")
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	targetFile := filepath.Join(targetDir, "castfile.yaml")
	targetYaml := `
name: target-project
env:
  TARGET_ENV: "set-in-target"
tasks:
  say-hello:
    uses: bash
    run: echo "Hello from target. TARGET_ENV=$TARGET_ENV PARENT_ENV=$PARENT_ENV"
`
	if err := os.WriteFile(targetFile, []byte(targetYaml), 0644); err != nil {
		t.Fatalf("failed to write target castfile: %v", err)
	}

	// 2. Create parent project
	parentFile := filepath.Join(tmpDir, "castfile.yaml")
	parentYaml := `
name: parent-project
env:
  PARENT_ENV: "set-in-parent"
tasks:
  call-target:
    uses: cast
    with:
      dir: ./target
      task: say-hello
`
	if err := os.WriteFile(parentFile, []byte(parentYaml), 0644); err != nil {
		t.Fatalf("failed to write parent castfile: %v", err)
	}

	// 3. Run parent task
	parentProj := &projects.Project{}
	err = parentProj.LoadFromYaml(parentFile)
	if err != nil {
		t.Fatalf("failed to load parent project: %v", err)
	}

	var stdout bytes.Buffer
	params := projects.RunTasksParams{
		Targets:     []string{"call-target"},
		Context:     context.Background(),
		ContextName: "default",
		Stdout:      &stdout,
		Stderr:      &stdout,
	}

	_, err = parentProj.RunTask(params)
	if err != nil {
		t.Fatalf("failed to run parent task: %v", err)
	}

	output := stdout.String()

	if !strings.Contains(output, "Hello from target") {
		t.Errorf("expected output to contain 'Hello from target', got: %s", output)
	}

	// Environment vars should carry over
	if !strings.Contains(output, "TARGET_ENV=set-in-target") {
		t.Errorf("expected target env to be present, got: %s", output)
	}
	if !strings.Contains(output, "PARENT_ENV=set-in-parent") {
		t.Errorf("expected parent env to be present, got: %s", output)
	}
}
