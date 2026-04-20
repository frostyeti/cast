package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

// ProjectConfig holds root-level parser behavior for a project.
type ProjectConfig struct {
	Context      *string        `yaml:"context,omitempty" json:"context,omitempty"`
	Contexts     []string       `yaml:"contexts,omitempty" json:"contexts,omitempty"`
	Shell        *string        `yaml:"shell,omitempty" json:"shell,omitempty"`
	Substitution *bool          `yaml:"substitution,omitempty" json:"substitution,omitempty"`
	Values       map[string]any `yaml:"-" json:"values,omitempty"`
}

func (pc *ProjectConfig) UnmarshalYAML(node *yaml.Node) error {
	if pc == nil {
		pc = &ProjectConfig{}
	}

	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	sub := true
	ctx := "default"

	pc.Substitution = &sub
	pc.Context = &ctx
	if pc.Values == nil {
		pc.Values = map[string]any{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "expected yaml mapping for ProjectConfig")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "context":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'context' field")
			}
			pc.Context = &valueNode.Value
		case "contexts":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'contexts' field")
			}
			contexts := make([]string, 0, len(valueNode.Content))
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.NewYamlError(item, "expected yaml scalar in 'contexts' list")
				}
				contexts = append(contexts, item.Value)
			}
			pc.Contexts = contexts
		case "substitution":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'substitution' field")
			}
			substitutionValue := true
			if err := valueNode.Decode(&substitutionValue); err != nil {
				return errors.NewYamlError(valueNode, "expected yaml boolean for 'substitution' field")
			}
			pc.Substitution = &substitutionValue
		case "shell":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'shell' field")
			}
			pc.Shell = &valueNode.Value
		default:
			var value any
			if err := valueNode.Decode(&value); err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project config value for '"+keyNode.Value+"'")
			}
			pc.Values[keyNode.Value] = value
		}
	}

	return nil
}
