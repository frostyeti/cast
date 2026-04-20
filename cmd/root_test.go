package cmd

import (
	"bytes"
	"strings"
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

	if strings.Contains(output, "\n  completion  ") {
		t.Errorf("expected completion command to be hidden from help output, got: %s", output)
	}

	compBuf := new(bytes.Buffer)
	cmd.SetOut(compBuf)
	cmd.SetErr(compBuf)
	cmd.SetArgs([]string{"completion", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected completion command to run while hidden, got %v", err)
	}
	if !strings.Contains(strings.ToLower(compBuf.String()), "autocompletion") {
		t.Errorf("expected completion help output, got: %s", compBuf.String())
	}

	if strings.Contains(output, "\n  help        Help about any command\n") {
		t.Errorf("expected help subcommand to be hidden from help output, got: %s", output)
	}

	if !strings.Contains(output, "\n  task        Manage and run tasks\n") {
		t.Errorf("expected task command in root help output, got: %s", output)
	}

	if !strings.Contains(output, "\n  job         Manage and run jobs\n") {
		t.Errorf("expected job command in root help output, got: %s", output)
	}

	if strings.Contains(output, "\n  update      ") {
		t.Errorf("expected root update command to be removed, got: %s", output)
	}
}

func TestRootCommand_DoesNotPrintUsageOnExecutionError(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"self", "config", "get"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected command error")
	}

	output := buf.String()
	if strings.Contains(strings.ToLower(output), "usage:") {
		t.Fatalf("expected no usage output on error, got: %s", output)
	}
}
