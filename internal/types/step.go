package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Step struct {
	Id       *string `json:"id,omitempty"`
	Name     *string `json:"name,omitempty"`
	Uses     string  `json:"uses,omitempty"`
	Run      string  `json:"run,omitempty"`
	With     With    `json:"with,omitempty"`
	Env      Env     `json:"env,omitempty"`
	Cwd      *string `json:"cwd,omitempty"`
	Desc     *string `json:"desc,omitempty"`
	Force    *string `json:"force,omitempty"`
	TaskName *string `json:"task,omitempty"`
}

type Steps []Step

func (s *Step) UnmarshalYAML(node *yaml.Node) error {
	// If it's just a string, we treat it as a task reference for a job step
	if node.Kind == yaml.ScalarNode {
		val := node.Value
		s.TaskName = &val
		return nil
	}

	// Otherwise, fallback to standard struct unmarshaling
	type alias Step
	var tmp alias
	if err := node.Decode(&tmp); err != nil {
		return errors.NewYamlError(node, "failed to decode step: "+err.Error())
	}
	*s = Step(tmp)
	return nil
}
