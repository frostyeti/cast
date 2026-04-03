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

func TestRunTask_ProjectCastDirMatchesCastfileDirectory(t *testing.T) {
	projectDir := t.TempDir()
	projectFile := filepath.Join(projectDir, "castfile.yaml")

	content := `
name: project-env
tasks:
  print:
    uses: bash
    run: echo "CAST_DIR=$CAST_DIR CAST_FILE=$CAST_FILE CAST_PARENT_DIR=$CAST_PARENT_DIR"
`

	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	proj := &projects.Project{}
	if err := proj.LoadFromYaml(projectFile); err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	var stdout bytes.Buffer
	_, err := proj.RunTask(projects.RunTasksParams{
		Targets:     []string{"print"},
		Context:     context.Background(),
		ContextName: "default",
		Stdout:      &stdout,
		Stderr:      &stdout,
	})
	if err != nil {
		t.Fatalf("failed to run task: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "CAST_DIR="+projectDir) {
		t.Fatalf("expected CAST_DIR to match project dir, got: %s", output)
	}
	if !strings.Contains(output, "CAST_FILE="+projectFile) {
		t.Fatalf("expected CAST_FILE to match castfile path, got: %s", output)
	}
	if !strings.Contains(output, "CAST_PARENT_DIR="+projectDir) {
		t.Fatalf("expected CAST_PARENT_DIR to match project dir, got: %s", output)
	}
}

func TestLoadFromYaml_ImportedModuleSetsModuleDirsWithoutTaskEnv(t *testing.T) {
	rootDir := t.TempDir()
	moduleDir := filepath.Join(rootDir, "module")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}

	moduleFile := filepath.Join(moduleDir, "castfile.yaml")
	moduleContent := `
name: sample-module
tasks:
  show:
    uses: bash
    run: echo "CAST_DIR=$CAST_DIR CAST_MODULE_DIR=$CAST_MODULE_DIR CAST_PARENT_DIR=$CAST_PARENT_DIR CAST_MODULE_FILE=$CAST_MODULE_FILE"
`
	if err := os.WriteFile(moduleFile, []byte(moduleContent), 0o644); err != nil {
		t.Fatalf("failed to write module castfile: %v", err)
	}

	projectFile := filepath.Join(rootDir, "castfile.yaml")
	projectContent := `
name: root-project
imports:
  - from: ./module
    namespace: mod
`
	if err := os.WriteFile(projectFile, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("failed to write root castfile: %v", err)
	}

	proj := &projects.Project{}
	if err := proj.LoadFromYaml(projectFile); err != nil {
		t.Fatalf("failed to load project: %v", err)
	}
	if err := proj.Init(); err != nil {
		t.Fatalf("failed to init project: %v", err)
	}

	task, ok := proj.Tasks.Get("mod-show")
	if !ok {
		t.Fatalf("expected imported module task to exist")
	}

	if task.Env == nil {
		t.Fatalf("expected imported module task env to be initialized")
	}
	if got := task.Env.Get("CAST_DIR"); got != moduleDir {
		t.Fatalf("expected CAST_DIR=%s, got %s", moduleDir, got)
	}
	if got := task.Env.Get("CAST_MODULE_DIR"); got != moduleDir {
		t.Fatalf("expected CAST_MODULE_DIR=%s, got %s", moduleDir, got)
	}
	if got := task.Env.Get("CAST_PARENT_DIR"); got != rootDir {
		t.Fatalf("expected CAST_PARENT_DIR=%s, got %s", rootDir, got)
	}
	if got := task.Env.Get("CAST_MODULE_FILE"); got != moduleFile {
		t.Fatalf("expected CAST_MODULE_FILE=%s, got %s", moduleFile, got)
	}
}
