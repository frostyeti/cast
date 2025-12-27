package schemas

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Need struct {
	Id       string `yaml:"id,omitempty"`
	Parallel bool   `yaml:"parallel,omitempty"`
}

func (n *Need) UnmarshalYAML(node *yaml.Node) error {
	if n == nil {
		n = &Need{}
	}

	if node.Kind == yaml.ScalarNode {
		n.Id = node.Value
		n.Parallel = false
		return nil
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			key := keyNode.Value
			switch key {
			case "id":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'id' field")
				}
				n.Id = valueNode.Value
			case "parallel":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'parallel' field")
				}
				if valueNode.Value == "true" {
					n.Parallel = true
				} else {
					n.Parallel = false
				}
			default:
				return errors.YamlErrorf(keyNode, "unexpected field '%s' in need", key)
			}
		}

		return nil
	}

	return errors.NewYamlError(node, "expected yaml scalar or mapping for 'need' node")
}
