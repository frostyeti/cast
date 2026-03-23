package projects

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/runstatus"
)

func TestRunShellTaskGoTemplate(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	task := &Task{
		Id:       "tmpl-shell",
		Name:     "tmpl-shell",
		Uses:     "shell",
		Run:      "echo '{{ .env.MESSAGE }} from {{ .os }}/{{ .arch }}'",
		Template: "gotmpl",
		Env: map[string]string{
			"MESSAGE": "hello",
		},
		With: map[string]any{},
	}

	ctx := TaskContext{
		Task:    task,
		Context: context.Background(),
		Stdout:  stdout,
		Stderr:  stderr,
	}

	res := runShell(ctx)
	if res.Status != runstatus.Ok {
		t.Fatalf("expected status ok, got %s (err=%v, stderr=%q)", runstatus.ToString(res.Status), res.Err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	if !strings.Contains(out, "hello from") {
		t.Fatalf("expected rendered gotmpl output, got %q", out)
	}
}

func TestRunShellTask_AppendsArgsForSingleLineRun(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	task := &Task{
		Id:   "single-line-args",
		Name: "single-line-args",
		Uses: "shell",
		Run:  "echo deno test -A",
		Env:  map[string]string{},
		With: map[string]any{},
		Args: []string{"--clean", "./path/to/file"},
	}

	ctx := TaskContext{Task: task, Context: context.Background(), Stdout: stdout, Stderr: stderr}
	res := runShell(ctx)
	if res.Status != runstatus.Ok {
		t.Fatalf("expected status ok, got %s (err=%v stderr=%q)", runstatus.ToString(res.Status), res.Err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	if !strings.Contains(out, "deno test -A --clean ./path/to/file") {
		t.Fatalf("expected appended args in output, got %q", out)
	}
}

func TestRunShellTask_AppendsArgsForScriptFile(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "echo_args.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho script $1 $2\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	task := &Task{
		Id:   "script-file-args",
		Name: "script-file-args",
		Uses: "shell",
		Run:  scriptPath,
		Env:  map[string]string{},
		With: map[string]any{},
		Args: []string{"--clean", "./path/to/file"},
		Cwd:  tmpDir,
	}

	ctx := TaskContext{Task: task, Context: context.Background(), Stdout: stdout, Stderr: stderr}
	res := runShell(ctx)
	if res.Status != runstatus.Ok {
		t.Fatalf("expected status ok, got %s (err=%v stderr=%q)", runstatus.ToString(res.Status), res.Err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	if !strings.Contains(out, "script --clean ./path/to/file") {
		t.Fatalf("expected appended args for file script, got %q", out)
	}
}
