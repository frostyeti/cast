package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type On struct {
	Schedule *Schedule `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Webhooks Webhooks  `yaml:"webhooks,omitempty" json:"webhooks,omitempty"`
}

func (o *On) UnmarshalYAML(value *yaml.Node) error {
	if o == nil {
		o = &On{}
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		switch keyNode.Value {
		case "schedule":
			var schedule Schedule
			err := valueNode.Decode(&schedule)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode on schedule: "+err.Error())
			}
			o.Schedule = &schedule
		case "webhooks":
			var webhooks Webhooks
			err := valueNode.Decode(&webhooks)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode on webhooks: "+err.Error())
			}
			for k, v := range webhooks {
				v.Id = k
				webhooks[k] = v
			}
			o.Webhooks = webhooks
		default:
			return errors.NewYamlError(keyNode, "unknown field for On: "+keyNode.Value)
		}
	}

	return nil
}
