package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Schedule struct {
	Crons    []string `yaml:"crons,omitempty" json:"crons,omitempty"`
	Timezone *string  `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

func (s *Schedule) UnmarshalYAML(value *yaml.Node) error {
	if s == nil {
		s = &Schedule{}
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		switch keyNode.Value {
		case "crons":
			if valueNode.Kind == yaml.ScalarNode {
				s.Crons = append(s.Crons, valueNode.Value)
			} else if valueNode.Kind == yaml.SequenceNode {
				for _, item := range valueNode.Content {
					s.Crons = append(s.Crons, item.Value)
				}
			} else {
				return errors.NewYamlError(valueNode, "crons must be a string or sequence of strings")
			}
		case "cron": // backward compatibility
			s.Crons = append(s.Crons, valueNode.Value)
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
