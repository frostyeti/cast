package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
)

func TestServer_SSHStreamEndpoint_Errors(t *testing.T) {
	s := NewServer("127.0.0.1", 0)

	// Create a dummy project and register it
	proj := &projects.Project{
		Schema: types.Project{
			Id:   "test-proj",
			Name: "test-proj",
			Inventory: &types.Inventory{
				Hosts: map[string]types.HostInfo{
					"my-server": {
						Host: "127.0.0.1",
						// using an unused port to ensure connection fails
						Port:     func(i uint) *uint { return &i }(12345),
						Password: func(s string) *string { return &s }("secret"),
					},
				},
			},
		},
	}
	s.projects["test-proj"] = proj

	// Setup mux manually
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/projects/{id}/ssh/{host_alias}", s.handleSSHStream)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Test 404 Project
	req1, _ := http.NewRequest("GET", ts.URL+"/api/v1/projects/not-found/ssh/my-server", nil)
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp1.StatusCode)
	}

	// 2. Test 404 Host
	req2, _ := http.NewRequest("GET", ts.URL+"/api/v1/projects/test-proj/ssh/not-found", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp2.StatusCode)
	}

	// 3. Test 500 Connection Failed (no ws upgrade since connection fails before upgrade)
	req3, _ := http.NewRequest("GET", ts.URL+"/api/v1/projects/test-proj/ssh/my-server", nil)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", resp3.StatusCode)
	}
}
