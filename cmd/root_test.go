package cmd

import (
	"bytes"
	"testing"
)

func TestRootCommand(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	// We pass an unknown command to test error or basic usage
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error from help command, got %v", err)
	}

	output := buf.String()
	if len(output) == 0 {
		t.Errorf("expected help output, got empty string")
	}
}
