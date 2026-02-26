package web

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
)

func TestLoadProject_Sanitization(t *testing.T) {
	tmpDir := t.TempDir()
	castfilePath := filepath.Join(tmpDir, "castfile.yaml")

	yamlContent := []byte(`
id: "@my-org/My.Project!"
jobs:
  "@job/1.0!":
    steps:
      - task: test
  "job-2":
    needs:
      - "@job/1.0!"
    steps:
      - task: test
`)
	if err := os.WriteFile(castfilePath, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	server := NewServer("127.0.0.1", 8080)
	server.loadProject(castfilePath)

	if len(server.projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(server.projects))
	}

	proj := server.projects["my-org-My-Project"]
	if proj == nil {
		t.Fatalf("project 'my-org-My-Project' not found, map has: %v", getKeys(server.projects))
	}

	if proj.Schema.Id != "my-org-My-Project" {
		t.Errorf("expected project ID to be 'my-org-My-Project', got %s", proj.Schema.Id)
	}

	job1, ok := proj.Schema.Jobs.Get("job-1-0")
	if !ok {
		t.Fatalf("job 'job-1-0' not found, job keys: %v", proj.Schema.Jobs.Keys())
	}

	if job1.Id != "job-1-0" {
		t.Errorf("expected job ID to be 'job-1-0', got %s", job1.Id)
	}

	job2, ok := proj.Schema.Jobs.Get("job-2")
	if !ok {
		t.Fatalf("job 'job-2' not found")
	}

	if job2.Needs == nil || len(*job2.Needs) == 0 {
		t.Fatalf("expected job2 to have needs")
	}

	need := (*job2.Needs)[0]
	if need.Id != "job-1-0" {
		t.Errorf("expected need ID to be 'job-1-0', got %s", need.Id)
	}
}

func getKeys(m map[string]*projects.Project) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
