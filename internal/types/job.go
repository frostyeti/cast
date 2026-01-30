package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Job struct {
	Id      string  `json:"id"`
	Name    string  `json:"name,omitempty"`
	Desc    string  `json:"desc,omitempty"`
	Needs   *Needs  `json:"needs,omitempty"`
	Steps   []Step  `json:"steps,omitempty"`
	If      *string `json:"if,omitempty"`
	Timeout *string `json:"timeout,omitempty"`
	Cwd     *string `json:"cwd,omitempty"`
}

func NewJob() *Job {
	return &Job{}
}

func (j *Job) UnmarshalYAML(node *yaml.Node) error {
	if j == nil {
		j = &Job{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "job must be a mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			j.Id = valueNode.Value
		case "name":
			j.Name = valueNode.Value
		case "desc":
			j.Desc = valueNode.Value
		case "needs":
			needs := NewNeeds()
			err := needs.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			j.Needs = needs
		case "steps":
			var steps []Step
			err := valueNode.Decode(&steps)
			if err != nil {
				return errors.YamlErrorf(valueNode, "failed to decode steps: %v", err)
			}
			j.Steps = steps
		case "if":
			ifStr := valueNode.Value
			j.If = &ifStr
		case "timeout":
			timeoutStr := valueNode.Value
			j.Timeout = &timeoutStr
		case "cwd":
			cwdStr := valueNode.Value
			j.Cwd = &cwdStr
		default:
			return errors.YamlErrorf(keyNode, "unknown field %q in Job", keyNode.Value)
		}
	}

	return nil
}
