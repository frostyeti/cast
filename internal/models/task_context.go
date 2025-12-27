package models

import (
	"context"

	"github.com/frostyeti/cast/internal/schemas"
)

type TaskContext struct {
	Project     *Project
	Scope       *Scope
	ContextName string
	Context     context.Context
	Task        *Task
	Schema      *schemas.Task
	Args        []string
}
