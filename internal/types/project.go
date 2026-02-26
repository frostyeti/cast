package types

import (
	"os"
	"path/filepath"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Project struct {
	Id             string           `yaml:"id,omitempty" json:"id,omitempty"`
	Name           string           `yaml:"name,omitempty" json:"name,omitempty"`
	Version        string           `yaml:"version,omitempty" json:"version,omitempty"`
	Desc           string           `yaml:"description,omitempty" json:"description,omitempty"`
	Imports        *Imports         `yaml:"imports,omitempty" json:"imports,omitempty"`
	Env            *Env             `yaml:"env,omitempty" json:"env,omitempty"`
	DotEnv         *DotEnvs         `yaml:"dotenv,omitempty" json:"dotenv,omitempty"`
	Paths          *Paths           `yaml:"paths,omitempty" json:"paths,omitempty"`
	Defaults       *ProjectDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Config         *ProjectConfig   `yaml:"config,omitempty" json:"config,omitempty"`
	On             *On              `yaml:"on,omitempty" json:"on,omitempty"`
	Meta           *Meta            `yaml:"meta,omitempty" json:"meta,omitempty"`
	Workspace      *Workspace       `yaml:"workspace,omitempty" json:"workspace,omitempty"`
	Tasks          *TaskMap         `yaml:"tasks,omitempty" json:"tasks,omitempty"`
	Jobs           *JobMap          `yaml:"jobs,omitempty" json:"jobs,omitempty"`
	Inventory      *Inventory       `yaml:"inventory,omitempty" json:"inventory,omitempty"`
	Inventories    []string         `yaml:"inventories,omitempty" json:"inventories,omitempty"`
	TrustedSources []string         `yaml:"trusted_sources,omitempty" json:"trusted_sources,omitempty"`
	Modules        []Module         `yaml:"-" json:"-"`
	File           string           `yaml:"-" json:"-"`
}

func NewProject() *Project {
	return &Project{}
}

func (p *Project) UnmarshalYAML(node *yaml.Node) error {
	if p == nil {
		p = NewProject()
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "project must be a mapping node.")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "project id must be a scalar.")
			}
			p.Id = valueNode.Value
		case "name":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "project name must be a scalar.")
			}
			p.Name = valueNode.Value
		case "version":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "project version must be a scalar.")
			}
			p.Version = valueNode.Value
		case "description", "desc":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "project description must be a scalar.")
			}
			p.Desc = valueNode.Value

		case "on":
			if valueNode.Kind != yaml.MappingNode {
				return errors.NewYamlError(valueNode, "project on must be a mapping node.")
			}
			p.On = &On{}
			err := valueNode.Decode(p.On)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project on: "+err.Error())
			}

		case "import", "imports", "modules":
			if p.Imports == nil {
				p.Imports = &Imports{}
			}
			tempImports := &Imports{}
			err := valueNode.Decode(tempImports)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project imports/modules: "+err.Error())
			}
			*p.Imports = append(*p.Imports, *tempImports...)
		case "env":
			p.Env = NewEnv()
			err := valueNode.Decode(p.Env)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project env: "+err.Error())
			}
		case "dotenv":
			p.DotEnv = &DotEnvs{}
			err := valueNode.Decode(p.DotEnv)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project dotenv: "+err.Error())
			}
		case "paths":
			p.Paths = &Paths{}
			err := valueNode.Decode(p.Paths)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project paths: "+err.Error())
			}
		case "defaults":
			p.Defaults = &ProjectDefaults{}
			err := valueNode.Decode(p.Defaults)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project defaults: "+err.Error())
			}
		case "meta":
			p.Meta = &Meta{}
			err := valueNode.Decode(p.Meta)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project meta: "+err.Error())
			}
		case "workspace":
			p.Workspace = &Workspace{}
			err := valueNode.Decode(p.Workspace)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project workspace: "+err.Error())
			}
		case "tasks":
			p.Tasks = NewTaskMap()
			err := valueNode.Decode(p.Tasks)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project tasks: "+err.Error())
			}
		case "jobs":
			p.Jobs = NewJobMap()
			err := valueNode.Decode(p.Jobs)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project jobs: "+err.Error())
			}
		case "inventory":
			p.Inventory = &Inventory{}
			err := valueNode.Decode(p.Inventory)
			if err != nil {
				return errors.NewYamlError(valueNode, "failed to decode project inventory: "+err.Error())
			}
		case "inventories":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "project inventories must be a sequence.")
			}
			for _, item := range valueNode.Content {
				p.Inventories = append(p.Inventories, item.Value)
			}
		case "trusted_sources", "trustedSources", "trusted-sources":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "project trusted_sources must be a sequence.")
			}
			for _, item := range valueNode.Content {
				p.TrustedSources = append(p.TrustedSources, item.Value)
			}
		default:
			continue
		}
	}

	return nil
}

func (p *Project) ReadFromYaml(file string) error {
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

	err = yaml.Unmarshal(data, p)
	if err != nil {
		return err
	}

	p.File = file

	return nil
}
