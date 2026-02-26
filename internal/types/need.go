package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Need struct {
	Id       string `yaml:"id,omitempty" json:"id,omitempty"`
	Parallel bool   `yaml:"parallel,omitempty" json:"parallel,omitempty"`
}

type Needs []Need

func NewNeeds() *Needs {
	return &Needs{}
}

func (n *Needs) FindByName(id string) (*Need, bool) {
	for _, need := range *n {
		if need.Id == id {
			return &need, true
		}
	}
	return nil, false
}

func (n *Needs) Names() []string {
	names := make([]string, 0, len(*n))
	for _, need := range *n {
		names = append(names, need.Id)
	}
	return names
}

func (n *Needs) UnmarshalYAML(node *yaml.Node) error {
	if n == nil {
		n = &Needs{}
	}

	if node.Kind == yaml.ScalarNode {
		var singleNeed Need
		singleNeed.Id = node.Value
		singleNeed.Parallel = false
		*n = append(*n, singleNeed)
		return nil
	}

	if node.Kind == yaml.SequenceNode {
		for _, needNode := range node.Content {
			var need Need
			err := need.UnmarshalYAML(needNode)
			if err != nil {
				return err
			}
			*n = append(*n, need)
		}
		return nil
	}

	return errors.NewYamlError(node, "invalid yaml node for Needs")
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
