package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Schedule struct {
	Cron     string  `yaml:"cron,omitempty" json:"cron,omitempty"`
	Timezone *string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

func (s *Schedule) UnmarshalYAML(value *yaml.Node) error {
	if s == nil {
		s = &Schedule{}
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		switch keyNode.Value {
		case "cron":
			s.Cron = valueNode.Value
		case "timezone":
			var timezone string
			err := valueNode.Decode(&timezone)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode schedule timezone: "+err.Error())
			}
			s.Timezone = &timezone
		default:
			return errors.NewYamlError(keyNode, "unknown field for Schedule: "+keyNode.Value)
		}
	}

	return nil
}
