package web

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
)

func TestServer_StreamEndpoint(t *testing.T) {
	s := NewServer("127.0.0.1", 0)

	// Create a dummy project and register it
	proj := &projects.Project{
		Schema: types.Project{
			Id:    "test-proj",
			Name:  "test-proj",
			Jobs:  types.NewJobMap(),
			Tasks: types.NewTaskMap(),
		},
	}

	taskName := "dummy"
	runCmd := "echo hello"
	proj.Schema.Tasks.Add(&types.Task{
		Id:   taskName,
		Name: taskName,
		Run:  &runCmd,
	})

	proj.Schema.Jobs.Add(&types.Job{
		Id:   "dummy-job",
		Name: "dummy-job",
		Steps: []types.Step{
			{
				TaskName: &taskName,
			},
		},
	})
	s.projects["test-proj"] = proj

	// Setup mux manually
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/projects/{id}/jobs/{jobId}/trigger", s.handleTriggerJob)
	mux.HandleFunc("GET /api/v1/streams/{runId}", s.handleStream)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Trigger a job to get a runId
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/projects/test-proj/jobs/dummy-job/trigger", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to trigger job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected status 202, got %d", resp.StatusCode)
	}

	var triggerResp map[string]string
	json.NewDecoder(resp.Body).Decode(&triggerResp)
	runID := triggerResp["runId"]

	if runID == "" {
		t.Fatalf("Expected non-empty runId")
	}

	// Give it a tiny moment to write logs
	time.Sleep(100 * time.Millisecond)

	// 2. Connect to the stream endpoint
	streamReq, _ := http.NewRequest("GET", ts.URL+"/api/v1/streams/"+runID, nil)

	// Use a context with timeout so we don't hang forever if something is broken
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	streamReq = streamReq.WithContext(ctx)

	streamResp, err := http.DefaultClient.Do(streamReq)
	if err != nil {
		t.Fatalf("Failed to connect to stream: %v", err)
	}
	defer streamResp.Body.Close()

	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected stream status 200, got %d", streamResp.StatusCode)
	}

	if streamResp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("Expected Content-Type text/event-stream, got %s", streamResp.Header.Get("Content-Type"))
	}

	scanner := bufio.NewScanner(streamResp.Body)
	gotData := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			gotData = true
			break // As long as we get some data (history or error), it works
		}
	}

	if !gotData {
		t.Fatalf("Expected to receive data events from stream")
	}
}
