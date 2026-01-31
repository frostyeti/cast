package types

import (
	"os"
	"path/filepath"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Module struct {
	Id        string     `yaml:"id,omitempty" json:"id,omitempty"`
	Name      string     `yaml:"name,omitempty" json:"name,omitempty"`
	Version   string     `yaml:"version,omitempty" json:"version,omitempty"`
	Desc      string     `yaml:"description,omitempty" json:"description,omitempty"`
	Imports   *Imports   `yaml:"imports,omitempty" json:"imports,omitempty"`
	Env       *Env       `yaml:"env,omitempty" json:"env,omitempty"`
	DotEnv    *DotEnvs   `yaml:"dotenv,omitempty" json:"dotenv,omitempty"`
	Paths     *Paths     `yaml:"paths,omitempty" json:"paths,omitempty"`
	Meta      *Meta      `yaml:"meta,omitempty" json:"meta,omitempty"`
	Tasks     *TaskMap   `yaml:"tasks,omitempty" json:"tasks,omitempty"`
	Inventory *Inventory `yaml:"inventory,omitempty" json:"inventory,omitempty"`
	File      string     `yaml:"-" json:"-"`
	Dir       string     `yaml:"-" json:"-"`
	TaskNames []string   `yaml:"-" json:"-"`
	Namespace string     `yaml:"-" json:"-"`
}

func (m *Module) UnmarshalYAML(node *yaml.Node) error {
	if m == nil {
		m = &Module{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "module must be a mapping node.")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "module id must be a scalar.")
			}
			m.Id = valueNode.Value
		case "name":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "module name must be a scalar.")
			}
			m.Name = valueNode.Value

		case "version":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "module version must be a scalar.")
			}
			m.Version = valueNode.Value
		case "description", "desc":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "module description must be a scalar.")
			}
			m.Desc = valueNode.Value
		case "imports":
			imports := &Imports{}
			err := imports.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.Imports = imports
		case "env":
			env := &Env{}
			err := env.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.Env = env
		case "dotenv":
			dotenv := &DotEnvs{}
			err := dotenv.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.DotEnv = dotenv
		case "paths":
			paths := &Paths{}
			err := paths.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.Paths = paths
		case "meta":
			meta := NewMeta()
			err := meta.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.Meta = meta
		case "tasks":
			taskMap := NewTaskMap()
			err := taskMap.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.Tasks = taskMap
		case "inventory":
			inventory := &Inventory{}
			err := inventory.UnmarshalYAML(valueNode)
			if err != nil {
				return err
			}
			m.Inventory = inventory
		}
	}

	return nil
}

func (m *Module) ReadFromYaml(file string) error {
	if !filepath.IsAbs(file) {
		absFile, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		file = absFile
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, m)
	if err != nil {
		return err
	}

	m.File = file
	m.Dir = filepath.Dir(file)

	return nil
}
