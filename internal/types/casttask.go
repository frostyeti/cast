package types

import (
	"os"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type CastTaskInput struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

type CastTaskRuns struct {
	Using string   `yaml:"using" json:"using"` // "docker", "deno", "composite"
	Image string   `yaml:"image,omitempty" json:"image,omitempty"`
	Args  []string `yaml:"args,omitempty" json:"args,omitempty"`
	Main  string   `yaml:"main,omitempty" json:"main,omitempty"` // For deno scripts
	Steps []Task   `yaml:"steps,omitempty" json:"steps,omitempty"`
}

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
		return errors.NewYamlError(nil, "failed to parse casttask.yaml: "+err.Error())
	}
	return nil
}
