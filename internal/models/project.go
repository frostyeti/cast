package models

import (
	"context"

	"github.com/frostyeti/cast/internal/schemas"
)

type CallTaskRunner = func(ctx *ProjectContext, tasks []string, remainingArgs []string) error

type Project struct {
	File           string
	Dir            string
	Id             string
	Name           string
	Desc           string
	Version        string
	Meta           map[string]interface{}
	Tasks          Tasks
	Env            Env
	CallTaskRunner CallTaskRunner
}

type RunTaskParams struct {
	RootContext   *WorkspaceContext
	TaskNames     []string
	RemainingArgs []string
	Context       context.Context
}

func (p *Project) FromCastConfig(config *schemas.CastConfig) {
	if p == nil {
		p = &Project{}
	}

	p.Id = config.Id
	p.Name = config.Name
	if config.Desc != nil {
		p.Desc = *config.Desc
	}
	if config.Version != nil {
		p.Version = *config.Version
	}

	p.Tasks = Tasks{
		Items: make(map[string]Task),
		keys:  []string{},
	}

	for _, key := range config.Tasks.Keys() {
		schema, _ := config.Tasks.Get(key)
		task := NewTaskFromSchema(&schema)
		p.Tasks.Set(task)
	}
}

func (p *Project) RunTasks(params RunTaskParams) error {
	pc := &ProjectContext{
		Project:         p,
		Args:            params.RemainingArgs,
		DisposableFiles: make(map[string]*DisposableFile),
	}
	err := pc.Init(params.RootContext)
	if err != nil {
		return err
	}

	if p.CallTaskRunner != nil {
		return p.CallTaskRunner(pc, params.TaskNames, params.RemainingArgs)
	}

	tasks, err := p.Tasks.FlattenTasks(params.TaskNames, pc.ContextName)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		println(task.Id)
	}

	return nil
}
