package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Webhooks map[string]Webhook

type Webhook struct {
	Id     string `yaml:"id,omitempty" json:"id,omitempty"`
	Job    string `yaml:"job,omitempty" json:"job,omitempty"`
	Task   string `yaml:"task,omitempty" json:"task,omitempty"`
	Secret string `yaml:"secret,omitempty" json:"secret,omitempty"`
	Token  string `yaml:"token,omitempty" json:"token,omitempty"`
}

func (w *Webhook) UnmarshalYAML(node *yaml.Node) error {
	if w == nil {
		w = &Webhook{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "webhook must be a mapping node.")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			w.Id = valueNode.Value
		case "job":
			w.Job = valueNode.Value
		case "task":
			w.Task = valueNode.Value
		case "secret":
			w.Secret = valueNode.Value
		case "token":
			w.Token = valueNode.Value
		default:
			return errors.NewYamlError(keyNode, "unknown field for Webhook: "+keyNode.Value)
		}
	}

	return nil
}
