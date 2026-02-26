package types

import (
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Task struct {
	Id       string   `yaml:"id,omitempty" json:"id,omitempty"`
	Name     string   `yaml:"name,omitempty" json:"name,omitempty"`
	Slug     string   `yaml:"slug,omitempty" json:"slug,omitempty"`
	Desc     *string  `yaml:"desc,omitempty" json:"desc,omitempty"`
	Help     *string  `yaml:"help,omitempty" json:"help,omitempty"`
	Env      *Env     `yaml:"env,omitempty" json:"env,omitempty"`
	DotEnv   []string `yaml:"dotenv,omitempty" json:"dotenv,omitempty"`
	Cwd      *string  `yaml:"cwd,omitempty" json:"cwd,omitempty"`
	Timeout  *string  `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Run      *string  `yaml:"run,omitempty" json:"run,omitempty"`
	Uses     *string  `yaml:"uses,omitempty" json:"uses,omitempty"`
	Args     []string `yaml:"args,omitempty" json:"args,omitempty"`
	Needs    Needs    `yaml:"needs,omitempty" json:"needs,omitempty"`
	With     *With    `yaml:"with,omitempty" json:"with,omitempty"`
	Hosts    []string `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	If       *string  `yaml:"if,omitempty" json:"if,omitempty"`
	Hooks    *Hooks   `yaml:"hooks,omitempty" json:"hooks,omitempty"`
	Force    *string  `yaml:"force,omitempty" json:"force,omitempty"`
	Extends  *string  `yaml:"extends,omitempty" json:"extends,omitempty"`
	Template *string  `yaml:"template,omitempty" json:"template,omitempty"`
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
		case "desc", "description":
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
		case "hooks":
			if valueNode.Kind == yaml.ScalarNode {
				hooks := &Hooks{}
				v := strings.TrimSpace(valueNode.Value)
				if strings.EqualFold(v, "true") || v == "1" {
					hooks.After = []string{"after"}
					hooks.Before = []string{"before"}
					t.Hooks = hooks
				}

				if strings.EqualFold(v, "false") || v == "0" {
					t.Hooks = nil
				}
				continue
			}

			if valueNode.Kind != yaml.MappingNode {
				return errors.NewYamlError(valueNode, "expected yaml mapping for 'hooks' field")
			}
			var hooks Hooks
			if err := valueNode.Decode(&hooks); err != nil {
				return err
			}
			t.Hooks = &hooks
		case "force":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'force' field")
			}
			t.Force = &valueNode.Value
		case "extends":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'extends' field")
			}
			t.Extends = &valueNode.Value
		case "template":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'template' field")
			}

			v := strings.TrimSpace(valueNode.Value)
			if strings.EqualFold(v, "true") || v == "1" {
				v = "gotmpl"
				t.Template = &v
				continue
			}
			if strings.EqualFold(v, "false") || v == "0" {
				t.Template = nil
				continue
			}
			t.Template = &valueNode.Value

		case "env":
			var env Env
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
		case "uses", "use":
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
			t.Needs = Needs{}
			for _, item := range valueNode.Content {
				var need Need
				if err := item.Decode(&need); err != nil {
					return err
				}
				t.Needs = append(t.Needs, need)
			}
		case "with", "input", "inputs":
			with := NewWith()
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
			t.If = &valueNode.Value
		default:
			return errors.YamlErrorf(keyNode, "unexpected field '%s' in task", key)
		}
	}

	return nil
}
