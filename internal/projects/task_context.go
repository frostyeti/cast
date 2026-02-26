package projects

import (
	"context"
	"io"

	"github.com/frostyeti/cast/internal/types"
)

type TaskContext struct {
	Project     *Project
	Schema      *types.Task
	Task        *Task
	Context     context.Context
	Args        []string
	ContextName string
	Outputs     map[string]any
	Stdout      io.Writer
	Stderr      io.Writer
}
