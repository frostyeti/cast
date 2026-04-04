package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

// Schedule defines cron-based triggers for a project.
// Crons accepts a string or list of strings.
type Schedule struct {
	Crons    []string `yaml:"crons,omitempty" json:"crons,omitempty"`
	Timezone *string  `yaml:"timezone,omitempty" json:"timezone,omitempty"`
}

func (s *Schedule) UnmarshalYAML(value *yaml.Node) error {
	if s == nil {
		s = &Schedule{}
	}

	if value.Kind == yaml.DocumentNode && len(value.Content) > 0 {
		value = value.Content[0]
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		switch keyNode.Value {
		case "crons":
			switch valueNode.Kind {
			case yaml.ScalarNode:
				s.Crons = append(s.Crons, valueNode.Value)
			case yaml.SequenceNode:
				for _, item := range valueNode.Content {
					s.Crons = append(s.Crons, item.Value)
				}
			default:
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
