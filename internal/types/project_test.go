package types_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostyeti/cast/internal/types"
	"go.yaml.in/yaml/v4"
)

func TestProjectUnmarshal(t *testing.T) {
	yamlData := `
id: my-project
name: My Project
version: 1.0.0
description: A sample project
env:
  TEST_ENV: my-value
tasks:
  test-task:
    run: echo "test"
`
	var p types.Project
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("failed to unmarshal project: %v", err)
	}

	if p.Id != "my-project" {
		t.Errorf("expected id 'my-project', got '%s'", p.Id)
	}
	if p.Name != "My Project" {
		t.Errorf("expected name 'My Project', got '%s'", p.Name)
	}
	if p.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", p.Version)
	}
	if p.Desc != "A sample project" {
		t.Errorf("expected description 'A sample project', got '%s'", p.Desc)
	}
	if p.Env == nil {
		t.Fatalf("expected env to be populated, got nil")
	}
	if p.Tasks == nil {
		t.Fatalf("expected tasks to be populated, got nil")
	}
}

func TestProjectReadFromYaml(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "castfile.yaml")
	yamlData := `
name: Read Test Project
`
	err := os.WriteFile(yamlFile, []byte(yamlData), 0644)
	if err != nil {
		t.Fatalf("failed to write test yaml file: %v", err)
	}

	p := types.NewProject()
	err = p.ReadFromYaml(yamlFile)
	if err != nil {
		t.Fatalf("failed to read from yaml: %v", err)
	}

	if p.Name != "Read Test Project" {
		t.Errorf("expected name 'Read Test Project', got '%s'", p.Name)
	}
	if p.File != yamlFile {
		t.Errorf("expected file path '%s', got '%s'", yamlFile, p.File)
	}
}
