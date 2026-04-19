package projects

import (
	"strings"
	"testing"
)

func TestCreateSSHAuth_ForceAgentWithoutAgentGivesClearError(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	_, _, err := createSSHAuth(sshAuthConfig{Host: "example", ForceAgent: true})
	if err == nil {
		t.Fatalf("expected force-agent error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ssh agent was required") {
		t.Fatalf("expected clear agent requirement message, got %v", err)
	}
}
