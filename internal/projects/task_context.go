package projects

import (
	"context"

	"github.com/frostyeti/cast/internal/types"
)

type TaskContext struct {
	Schema      *types.Task
	Task        *Task
	Context     context.Context
	Args        []string
	ContextName string
	Outputs     map[string]any
}
