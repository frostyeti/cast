//go:build integration
// +build integration

package projects_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
)

func TestProjectLoading_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "castfile")

	yamlData := `
name: Integration Test
tasks:
  hello:
    run: echo "Hello, Integration"
`
	err := os.WriteFile(yamlFile, []byte(yamlData), 0644)
	if err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	proj := &projects.Project{}
	err = proj.LoadFromYaml(yamlFile)
	if err != nil {
		t.Fatalf("failed to read from yaml: %v", err)
	}

	// Just check if we loaded tasks successfully
	if proj.Schema.Tasks == nil || proj.Schema.Tasks.Len() == 0 {
		t.Errorf("expected tasks to be loaded")
	}
}
