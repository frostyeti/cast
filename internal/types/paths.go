package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type EnvPath struct {
	Path   string `json:"path,omitempty"`
	OS     string `json:"os,omitempty"`
	Append bool   `json:"append,omitempty"`
}

type Paths []EnvPath

func (p *Paths) UnmarshalYAML(node *yaml.Node) error {
	if p == nil {
		p = &Paths{}
	}

	if node.Kind == yaml.SequenceNode {
		for _, pathNode := range node.Content {
			switch pathNode.Kind {
			case yaml.ScalarNode:
				*p = append(*p, EnvPath{Path: pathNode.Value})
			case yaml.MappingNode:
				var envPath EnvPath
				if err := pathNode.Decode(&envPath); err != nil {
					return err
				}
				*p = append(*p, envPath)
			default:
				return errors.NewYamlError(pathNode, "invalid yaml node for EnvPath entry")
			}
		}
		return nil
	}

	return errors.NewYamlError(node, "invalid yaml node for Paths")
}

func (ep *EnvPath) UnmarshalYAML(node *yaml.Node) error {
	if ep == nil {
		ep = &EnvPath{}
	}

	if node.Kind == yaml.ScalarNode {
		ep.Path = node.Value
		return nil
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			key := keyNode.Value
			switch key {
			case "value", "path":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(valueNode, "expected yaml scalar for 'path' field")
				}
				ep.Path = valueNode.Value
			case "windows", "win", "win32":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(valueNode, "expected yaml scalar for 'os' field")
				}
				ep.OS = "windows"
				ep.Path = valueNode.Value
			case "linux", "unix":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(valueNode, "expected yaml scalar for 'os' field")
				}
				ep.OS = "linux"
				ep.Path = valueNode.Value
			case "mac", "macos", "darwin", "osx":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(valueNode, "expected yaml scalar for 'os' field")
				}
				ep.OS = "darwin"
				ep.Path = valueNode.Value
			case "os":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(valueNode, "expected yaml scalar for 'os' field")
				}
				ep.OS = valueNode.Value
			case "append":
				var appendBool bool
				if err := valueNode.Decode(&appendBool); err != nil {
					return errors.NewYamlError(valueNode, "expected yaml boolean for 'append' field")
				}
				ep.Append = appendBool
			default:
				return errors.YamlErrorf(keyNode, "unexpected field '%s' in EnvPath entry", key)
			}
		}
		return nil
	}

	return errors.NewYamlError(node, "EnvPath must be a scalar or mapping node")
}
