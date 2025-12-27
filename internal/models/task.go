package models

import (
	"time"

	"github.com/frostyeti/cast/internal/id"
	"github.com/frostyeti/cast/internal/schemas"
)

type Task struct {
	evaluated bool
	Schema    *schemas.Task
	Id        string
	Name      string
	Slug      string
	Help      string
	Env       Env
	Desc      string
	Run       string
	Uses      string
	Args      []string
	Hosts     schemas.Hosts
	Cwd       string
	Timeout   time.Duration
	Needs     Needs
	Inputs    Inputs
	Hooks     *Hooks
	Force     bool
}

func NewTaskFromSchema(schema *schemas.Task) *Task {
	uid := schema.Id
	name := schema.Name
	slug := schema.Slug

	if uid == "" && name != "" {
		uid = id.Convert(name)
	}

	if name == "" && uid != "" {
		name = uid
	}

	if slug == "" && uid != "" {
		slug = id.Slugify(uid)
	}

	uses := ""
	if schema.Uses != nil {
		uses = *schema.Uses
	}

	run := ""
	if schema.Run != nil {
		run = *schema.Run
	}

	needs := make(Needs, 0)
	for _, need := range schema.Needs {
		n := Need{
			Id:       need.Id,
			Parallel: need.Parallel,
		}

		needs = append(needs, n)
	}

	var hooks *Hooks
	if schema.Hooks != nil {
		hooks = &Hooks{}
		hooks.After = schema.Hooks.After
		hooks.Before = schema.Hooks.Before
	}

	task := &Task{
		Schema: schema,
		Id:     uid,
		Name:   name,
		Slug:   slug,
		Uses:   uses,
		Run:    run,
		Needs:  needs,
		Hooks:  hooks,
	}

	return task
}
