package web

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebhook_Trigger(t *testing.T) {
	tmpDir := t.TempDir()
	castfilePath := filepath.Join(tmpDir, "castfile.yaml")

	yamlContent := []byte(`
id: my-proj
on:
  webhooks:
    hook1:
      job: my-job
      secret: mysecret
    hook2:
      task: my-task
      token: mytoken
jobs:
  my-job:
    steps:
      - run: echo hello
tasks:
  my-task:
    run: echo hello
`)
	if err := os.WriteFile(castfilePath, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	server := NewServer("127.0.0.1", 8080)
	server.loadProject(castfilePath)

	// Test 1: GitHub HMAC Secret
	body := []byte(`{"ref":"refs/heads/main"}`)
	req, err := http.NewRequest("POST", "/api/webhooks/hook1", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.SetPathValue("webhookId", "hook1")

	// Sign the payload
	mac := hmac.New(sha256.New, []byte("mysecret"))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Hub-Signature-256", sig)

	rr := httptest.NewRecorder()
	server.handleWebhook(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %v: %s", rr.Code, rr.Body.String())
	}

	if !strings.Contains(rr.Body.String(), "Triggered job my-job") {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}

	// Test 2: Invalid Secret
	req2, _ := http.NewRequest("POST", "/api/webhooks/hook1", bytes.NewReader(body))
	req2.SetPathValue("webhookId", "hook1")
	req2.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	rr2 := httptest.NewRecorder()
	server.handleWebhook(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %v", rr2.Code)
	}

	// Test 3: Bearer Token Task
	req3, _ := http.NewRequest("POST", "/api/webhooks/hook2", nil)
	req3.SetPathValue("webhookId", "hook2")
	req3.Header.Set("Authorization", "Bearer mytoken")

	rr3 := httptest.NewRecorder()
	server.handleWebhook(rr3, req3)

	if rr3.Code != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %v", rr3.Code)
	}

	if !strings.Contains(rr3.Body.String(), "Triggered task my-task") {
		t.Errorf("unexpected body: %s", rr3.Body.String())
	}

	// Test 4: Invalid Token
	req4, _ := http.NewRequest("POST", "/api/webhooks/hook2", nil)
	req4.SetPathValue("webhookId", "hook2")
	req4.Header.Set("Authorization", "Bearer invalid")

	rr4 := httptest.NewRecorder()
	server.handleWebhook(rr4, req4)

	if rr4.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %v", rr4.Code)
	}
}
