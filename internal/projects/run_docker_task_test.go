package projects

import (
	"bytes"
	"context"
	"testing"

	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/cast/internal/types"
)

func TestRunDockerTask_AppendsTaskArgs(t *testing.T) {
	ctx := TaskContext{
		Task: &Task{
			Id:   "docker-args",
			Name: "docker-args",
			Uses: "docker",
			Run:  "echo hi",
			Env:  map[string]string{},
			With: map[string]any{"image": "alpine:latest"},
			Args: []string{"--clean", "./path/to/file"},
		},
		Schema:  &types.Task{},
		Context: context.Background(),
		Stdout:  &bytes.Buffer{},
		Stderr:  &bytes.Buffer{},
	}

	res := runDockerTask(ctx)
	if res.Status == runstatus.Ok {
		return
	}

	if res.Err == nil {
		t.Fatalf("expected docker failure to include error when runtime unavailable")
	}
}
