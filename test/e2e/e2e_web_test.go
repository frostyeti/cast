package e2e_test

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestE2E_WebModeAndCron(t *testing.T) {
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile with jobs and schedule...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	yamlData := `
name: Web E2E Test
id: web-e2e
on:
  schedule:
    crons: ["* * * * *"] # Every minute
tasks:
  test-task:
    uses: bash
    run: echo "test task executed"
jobs:
  default:
    steps:
      - "test-task"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	// Run the cast web server in background
	runCmd := exec.Command(binPath, "web", "--port", "8082")
	runCmd.Dir = tmpDir

	if err := runCmd.Start(); err != nil {
		t.Fatalf("failed to start web server: %v", err)
	}

	// Ensure server is killed after test
	defer func() {
		if runCmd.Process != nil {
			runCmd.Process.Kill()
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Test 1: Health check
	resp, err := http.Get("http://127.0.0.1:8082/health")
	if err != nil {
		t.Fatalf("failed to get /health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /health, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test 2: Get projects
	resp, err = http.Get("http://127.0.0.1:8082/api/v1/projects")
	if err != nil {
		t.Fatalf("failed to get /api/v1/projects: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for /projects, got %d", resp.StatusCode)
	}
	var projects []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&projects)
	resp.Body.Close()

	found := false
	for _, p := range projects {
		if p["id"] == "web-e2e" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("project 'web-e2e' not found in projects list")
	}

	// Test 3: Trigger task
	resp, err = http.Post("http://127.0.0.1:8082/api/v1/projects/web-e2e/tasks/test-task/trigger", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to post trigger task: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted for trigger task, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test 4: Trigger job
	resp, err = http.Post("http://127.0.0.1:8082/api/v1/projects/web-e2e/jobs/default/trigger", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to post trigger job: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted for trigger job, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
