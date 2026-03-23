package e2e_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func waitForHTTPStatus(t *testing.T, url string, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error
	var lastStatus int

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
		} else {
			lastStatus = resp.StatusCode
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == want {
				return
			}
			lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s status %d: %v (last status %d)", url, want, lastErr, lastStatus)
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else {
			lastErr = err
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for file %s: %v", path, lastErr)
}
