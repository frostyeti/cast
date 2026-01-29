package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type ProjectConfig struct {
	Context      *string `yaml:"context,omitempty" json:"context,omitempty"`
	Substitution *bool   `yaml:"substitution,omitempty" json:"substitution,omitempty"`
}

func (pc *ProjectConfig) UnmarshalYAML(node *yaml.Node) error {
	if pc == nil {
		pc = &ProjectConfig{}
	}

	sub := true
	ctx := "default"

	pc.Substitution = &sub
	pc.Context = &ctx

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
		case "substitution":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'substitution' field")
			}
			substitutionValue := false
			if valueNode.Value == "true" || valueNode.Value == "yes" {
				substitutionValue = true
			}
			pc.Substitution = &substitutionValue
		default:
			return errors.NewYamlError(keyNode, "unknown field in ProjectConfig: "+keyNode.Value)
		}
	}

	return nil
}
