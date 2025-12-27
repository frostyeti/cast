package models

import "github.com/frostyeti/cast/internal/schemas"

type Workspace struct {
	Projects map[string]Project `yaml:"projects,omitempty"`
	Defaults schemas.Defaults   `yaml:"defaults,omitempty"`
	Env      schemas.EnvVars    `yaml:"env,omitempty"`
}
