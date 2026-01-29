package projects

import (
	"time"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/runstatus"
)

type TaskResult struct {
	Err       error
	Status    int
	StartedAt time.Time
	EndedAt   time.Time
	Message   string
	Output    map[string]interface{}
	Task      *Task
}

func (tr *TaskResult) Start() *TaskResult {
	tr.StartedAt = time.Now().UTC()
	return tr
}

func (tr *TaskResult) End() *TaskResult {
	tr.EndedAt = time.Now().UTC()
	return tr
}

func (tr *TaskResult) Ok() *TaskResult {
	tr.Status = runstatus.Ok
	tr.End()
	return tr
}

func (tr *TaskResult) Fail(err error) *TaskResult {
	tr.Err = errors.WithCause(err, tr.Err)
	tr.Status = runstatus.Error
	tr.End()
	return tr
}

func (tr *TaskResult) Skip(msg string) *TaskResult {
	tr.Status = runstatus.Skipped
	tr.Message = msg
	tr.End()
	return tr
}

func (tr *TaskResult) Cancel(msg string) *TaskResult {
	tr.Status = runstatus.Cancelled
	tr.Message = msg
	tr.End()
	return tr
}

func NewTaskResult() *TaskResult {
	return &TaskResult{
		Err:       nil,
		Status:    runstatus.None,
		StartedAt: time.Now().UTC(),
		EndedAt:   time.Now().UTC(),
		Message:   "",
		Output:    make(map[string]interface{}),
	}
}
