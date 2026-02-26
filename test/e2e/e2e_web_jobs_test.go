package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2E_WebJobExecutionWithLogs(t *testing.T) {
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile with multi-step job...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	yamlData := `
name: Web Job Log Test
id: web-job-log-test
tasks:
  step1:
    uses: bash
    run: echo "starting step 1" && echo "step1-done" > step1.txt
  step2:
    uses: bash
    run: echo "starting step 2" && echo "step2-done" > step2.txt
jobs:
  myjob:
    steps:
      - "step1"
      - "step2"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	// Run the cast web server in background
	runCmd := exec.Command(binPath, "web", "--port", "8083")
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

	// Trigger job
	resp, err := http.Post("http://127.0.0.1:8083/api/v1/projects/web-job-log-test/jobs/myjob/trigger", "application/json", nil)
	if err != nil {
		t.Fatalf("failed to post trigger job: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted for trigger job, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Wait for job to finish
	time.Sleep(3 * time.Second)

	// Verify files were created in order
	if _, err := os.Stat(filepath.Join(tmpDir, "step1.txt")); os.IsNotExist(err) {
		t.Errorf("step1.txt was not created")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "step2.txt")); os.IsNotExist(err) {
		t.Errorf("step2.txt was not created")
	}

	// Get job runs
	resp, err = http.Get("http://127.0.0.1:8083/api/v1/projects/web-job-log-test/jobs/myjob/runs")
	if err != nil {
		t.Fatalf("failed to get job runs: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK for job runs, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var runs []map[string]interface{}
	if err := json.Unmarshal(body, &runs); err != nil {
		t.Fatalf("failed to unmarshal job runs: %v\nBody: %s", err, string(body))
	}

	if len(runs) == 0 {
		t.Fatalf("expected at least 1 job run, got 0")
	}

	run := runs[0]
	if run["Status"] != "success" {
		t.Errorf("expected status 'success', got %v", run["Status"])
	}

	logs, ok := run["Logs"].(string)
	if !ok {
		t.Fatalf("expected Logs to be string, got %T", run["Logs"])
	}

	if !strings.Contains(logs, "starting step 1") {
		t.Errorf("logs missing step 1 output. Got: %s", logs)
	}
	if !strings.Contains(logs, "starting step 2") {
		t.Errorf("logs missing step 2 output. Got: %s", logs)
	}
}
