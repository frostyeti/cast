package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Workspace struct {
	Include []string          `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string          `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
}

func (w *Workspace) UnmarshalYAML(node *yaml.Node) error {
	if w == nil {
		w = &Workspace{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "invalid yaml node for Workspace")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		switch key {
		case "include":
			includes := []string{}
			switch valueNode.Kind {
			case yaml.ScalarNode:
				includes = append(includes, valueNode.Value)
			case yaml.SequenceNode:
				for _, item := range valueNode.Content {
					if item.Kind != yaml.ScalarNode {
						return errors.NewYamlError(item, "expected yaml scalar in 'include' sequence")
					}
					includes = append(includes, item.Value)
				}
			default:
				return errors.NewYamlError(valueNode, "expected yaml scalar or sequence for 'include' field")
			}
			w.Include = includes
		case "exclude":
			excludes := []string{}
			switch valueNode.Kind {
			case yaml.ScalarNode:
				excludes = append(excludes, valueNode.Value)
			case yaml.SequenceNode:
				for _, item := range valueNode.Content {
					if item.Kind != yaml.ScalarNode {
						return errors.NewYamlError(item, "expected yaml scalar in 'exclude' sequence")
					}
					excludes = append(excludes, item.Value)
				}
			default:
				return errors.NewYamlError(valueNode, "expected yaml scalar or sequence for 'exclude' field")
			}
			w.Exclude = excludes
		case "aliases":
			aliases := map[string]string{}
			if valueNode.Kind != yaml.MappingNode {
				return errors.NewYamlError(valueNode, "expected yaml mapping for 'aliases' field")
			}
			for j := 0; j < len(valueNode.Content); j += 2 {
				aliasKeyNode := valueNode.Content[j]
				aliasValueNode := valueNode.Content[j+1]

				if aliasKeyNode.Kind != yaml.ScalarNode || aliasValueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(aliasValueNode, "expected yaml scalars for alias key-value pair")
				}
				aliases[aliasKeyNode.Value] = aliasValueNode.Value
			}
			w.Aliases = aliases
		}
	}

	return nil
}
