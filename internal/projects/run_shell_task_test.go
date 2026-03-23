package projects

import (
	"bytes"
	"context"
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
