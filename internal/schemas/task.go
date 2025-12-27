package schemas

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Task struct {
	Id        string
	Name      string
	Slug      string
	Desc      *string
	Help      *string
	Env       *EnvVars
	DotEnv    []string
	Cwd       *string
	Timeout   *string
	Run       *string
	Uses      *string
	Args      []string
	Needs     []Need
	With      *With
	Hosts     []string
	Predicate *string
	Hooks     *Hooks
	Force     *string
	Extends   *string
}

func (t *Task) UnmarshalYAML(value *yaml.Node) error {
	if t == nil {
		t = &Task{}
	}

	if value.Kind == yaml.ScalarNode {
		t.Run = &value.Value
		return nil
	}

	if value.Kind != yaml.MappingNode {
		return errors.NewYamlError(value, "expected yaml scalar or mapping for task")
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		key := keyNode.Value
		switch key {
		case "id":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'id' field")
			}
			t.Id = valueNode.Value
		case "desc":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'desc' field")
			}
			t.Desc = &valueNode.Value
		case "help":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'help' field")
			}
			t.Help = &valueNode.Value
		case "name":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'name' field")
			}
			t.Name = valueNode.Value
		case "env":
			var env EnvVars
			if err := valueNode.Decode(&env); err != nil {
				return err
			}
			t.Env = &env
		case "dotenv", "envfile", "env-file":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'dotenv' field")
			}
			t.DotEnv = make([]string, 0)
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.NewYamlError(item, "expected yaml scalar in 'dotenv' list")
				}
				t.DotEnv = append(t.DotEnv, item.Value)
			}
		case "cwd":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'cwd' field")
			}
			t.Cwd = &valueNode.Value
		case "timeout":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'timeout' field")
			}
			t.Timeout = &valueNode.Value
		case "run":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'run' field")
			}
			t.Run = &valueNode.Value
		case "uses":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'uses' field")
			}
			t.Uses = &valueNode.Value
		case "args":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'args' field")
			}
			t.Args = make([]string, 0)
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.NewYamlError(item, "expected yaml scalar in 'args' list")
				}
				t.Args = append(t.Args, item.Value)
			}
		case "needs", "deps", "dependencies":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'needs' field")
			}
			t.Needs = []Need{}
			for _, item := range valueNode.Content {
				var need Need
				if err := item.Decode(&need); err != nil {
					return err
				}
				t.Needs = append(t.Needs, need)
			}
		case "with", "input", "inputs":
			with := &With{}
			if err := valueNode.Decode(with); err != nil {
				return err
			}
			t.With = with
		case "hosts":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'hosts' field")
			}
			t.Hosts = make([]string, 0)
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.NewYamlError(item, "expected yaml scalar in 'hosts' list")
				}
				t.Hosts = append(t.Hosts, item.Value)
			}
		case "if", "predicate":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'if' field")
			}
			t.Predicate = &valueNode.Value
		default:
			return errors.YamlErrorf(keyNode, "unexpected field '%s' in task", key)
		}
	}

	return nil
}
