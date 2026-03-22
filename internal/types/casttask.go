package types

import (
	"os"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

// CastTaskInput defines a remote task input parameter.
// Required inputs must be supplied by the caller.
type CastTaskInput struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

// CastTaskRuns declares how a remote task is executed.
// Using supports docker, deno, bun, or composite.
type CastTaskRuns struct {
	Using string   `yaml:"using" json:"using"` // "docker", "deno", "composite", "bun"
	Image string   `yaml:"image,omitempty" json:"image,omitempty"`
	Args  []string `yaml:"args,omitempty" json:"args,omitempty"`
	Main  string   `yaml:"main,omitempty" json:"main,omitempty"` // For deno scripts
	Steps []Task   `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// CastTask is an experimental remote task definition.
// It is stored in a cast.task file.
type CastTask struct {
	Name        string                   `yaml:"name" json:"name"`
	Description string                   `yaml:"description,omitempty" json:"description,omitempty"`
	Inputs      map[string]CastTaskInput `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Runs        CastTaskRuns             `yaml:"runs" json:"runs"`
}

func (c *CastTask) ReadFromYaml(file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		return errors.NewYamlError(nil, "failed to parse cast.task: "+err.Error())
	}
	return nil
}
