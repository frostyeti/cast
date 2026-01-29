package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type DotEnv struct {
	Path     string   `yaml:"path"`
	OS       string   `yaml:"os,omitempty"`
	Contexts []string `yaml:"contexts,omitempty"`
}

type DotEnvs []DotEnv

func (d DotEnvs) UnmarshalYAML(node *yaml.Node) error {
	if d == nil {
		d = []DotEnv{}
	}

	if node.Kind == yaml.ScalarNode {
		var singleDotEnv DotEnv
		singleDotEnv.Path = node.Value
		d = append(d, singleDotEnv)
		return nil
	}

	if node.Kind != yaml.SequenceNode {
		return errors.NewYamlError(node, "invalid yaml node for DotEnvs")
	}

	for _, dotenvNode := range node.Content {
		var next DotEnv
		err := next.UnmarshalYAML(dotenvNode)
		if err != nil {
			return err
		}
		d = append(d, next)
	}

	return nil
}

func (de *DotEnv) UnmarshalYAML(node *yaml.Node) error {
	if de == nil {
		de = &DotEnv{}
	}

	if node.Kind == yaml.ScalarNode {
		var singleDotEnv DotEnv
		singleDotEnv.Path = node.Value
		*de = singleDotEnv
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "invalid yaml node for DotEnv")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		switch key {
		case "path":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'path' field")
			}
			de.Path = valueNode.Value
		case "os":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'os' field")
			}
			de.OS = valueNode.Value
		case "contexts":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'contexts' field")
			}
			var contexts []string
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.NewYamlError(item, "expected yaml scalar for context item")
				}
				contexts = append(contexts, item.Value)
			}
			de.Contexts = contexts
		default:
			return errors.YamlErrorf(keyNode, "unexpected field '%s' in DotEnv entry", key)
		}
	}

	return nil
}

func (df *DotEnv) HasContext(context string) bool {
	if len(df.Contexts) == 0 {
		return context == "*" || context == "" || context == "default"
	}

	for _, ctx := range df.Contexts {
		if ctx == "*" || ctx == context {
			return true
		}
	}

	return false
}

func (df *DotEnv) MarshalYAML() (interface{}, error) {
	l := len(df.Contexts)
	isDefaultContext := l == 0 || (l == 1 && df.Contexts[0] == "*")

	if df.OS == "" && isDefaultContext {
		return df.Path, nil
	}

	mapping := make(map[string]interface{})

	if df.OS != "" && df.OS != "*" && isDefaultContext {
		mapping[df.OS] = df.Path
		return mapping, nil
	}

	mapping["path"] = df.Path

	if df.OS != "" && df.OS != "*" {
		mapping["os"] = df.OS
	}

	if !isDefaultContext {
		mapping["contexts"] = df.Contexts
	}

	return mapping, nil
}
