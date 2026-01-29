package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Import struct {
	From         string   `json:"from,omitempty"`
	Namespace    string   `json:"namespace,omitempty"`
	Tasks        []string `json:"tasks,omitempty"`
	resolvedFrom string   `json:"-"`
}

type Imports []Import

func (i *Imports) UnmarshalYAML(node *yaml.Node) error {
	if i == nil {
		i = &Imports{}
	}

	if node.Kind == yaml.ScalarNode {
		var singleImport Import
		singleImport.From = node.Value
		*i = append(*i, singleImport)
		return nil
	}

	if node.Kind == yaml.SequenceNode {
		for _, importNode := range node.Content {
			var imp Import
			err := imp.UnmarshalYAML(importNode)
			if err != nil {
				return err
			}
			*i = append(*i, imp)
		}
		return nil
	}

	return errors.NewYamlError(node, "invalid yaml node for Imports")
}

func (i *Import) UnmarshalYAML(node *yaml.Node) error {
	if i == nil {
		i = &Import{}
	}

	if node.Kind == yaml.ScalarNode {
		i.From = node.Value
		return nil
	}

	if node.Kind == yaml.MappingNode {
		for j := 0; j < len(node.Content); j += 2 {
			keyNode := node.Content[j]
			valueNode := node.Content[j+1]

			key := keyNode.Value
			switch key {
			case "from":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(node, "expected yaml scalar for 'from' field")
				}
				i.From = valueNode.Value
			case "namespace", "ns":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.NewYamlError(node, "expected yaml scalar for 'namespace' field")
				}
				i.Namespace = valueNode.Value
			case "tasks":
				if valueNode.Kind != yaml.SequenceNode {
					return errors.NewYamlError(node, "expected yaml sequence for 'tasks' field")
				}
				var tasks []string
				for _, taskNode := range valueNode.Content {
					if taskNode.Kind != yaml.ScalarNode {
						return errors.NewYamlError(node, "expected yaml scalar for task in 'tasks' field")
					}
					tasks = append(tasks, taskNode.Value)
				}
			}
		}
		return nil
	}

	return errors.NewYamlError(node, "invalid yaml node for Import")
}
