package types

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

type Project struct {
	Id        string          `yaml:"id,omitempty" json:"id,omitempty"`
	Name      string          `yaml:"name,omitempty" json:"name,omitempty"`
	Version   string          `yaml:"version,omitempty" json:"version,omitempty"`
	Desc      string          `yaml:"description,omitempty" json:"description,omitempty"`
	Imports   Imports         `yaml:"imports,omitempty" json:"imports,omitempty"`
	Env       Env             `yaml:"env,omitempty" json:"env,omitempty"`
	DotEnv    DotEnvs         `yaml:"dotenv,omitempty" json:"dotenv,omitempty"`
	Paths     Paths           `yaml:"paths,omitempty" json:"paths,omitempty"`
	Defaults  ProjectDefaults `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Config    ProjectConfig
	Meta      Meta       `yaml:"meta,omitempty" json:"meta,omitempty"`
	Workspace *Workspace `yaml:"workspace,omitempty" json:"workspace,omitempty"`
	Tasks     TaskMap    `yaml:"tasks,omitempty" json:"tasks,omitempty"`
	Inventory Inventory  `yaml:"inventory,omitempty" json:"inventory,omitempty"`
	Modules   []Module   `yaml:"-" json:"-"`
	File      string     `yaml:"-" json:"-"`
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
