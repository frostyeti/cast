package models

import "github.com/frostyeti/cast/internal/schemas"

type Workspace struct {
	Projects map[string]Project        `yaml:"projects,omitempty"`
	Defaults schemas.WorkspaceDefaults `yaml:"defaults,omitempty"`
	Env      schemas.EnvVars           `yaml:"env,omitempty"`
}

func NewWorkspace() *Workspace {
	return &Workspace{
		Projects: make(map[string]Project),
	}
}
