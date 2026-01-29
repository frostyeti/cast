package types

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

type Module struct {
	Id        string    `yaml:"id,omitempty" json:"id,omitempty"`
	Name      string    `yaml:"name,omitempty" json:"name,omitempty"`
	Version   string    `yaml:"version,omitempty" json:"version,omitempty"`
	Desc      string    `yaml:"description,omitempty" json:"description,omitempty"`
	Imports   Imports   `yaml:"imports,omitempty" json:"imports,omitempty"`
	Env       Env       `yaml:"env,omitempty" json:"env,omitempty"`
	DotEnv    DotEnvs   `yaml:"dotenv,omitempty" json:"dotenv,omitempty"`
	Paths     Paths     `yaml:"paths,omitempty" json:"paths,omitempty"`
	Meta      Meta      `yaml:"meta,omitempty" json:"meta,omitempty"`
	Tasks     TaskMap   `yaml:"tasks,omitempty" json:"tasks,omitempty"`
	Inventory Inventory `yaml:"inventory,omitempty" json:"inventory,omitempty"`
	File      string    `yaml:"-" json:"-"`
	Dir       string    `yaml:"-" json:"-"`
	TaskNames []string  `yaml:"-" json:"-"`
	Namespace string    `yaml:"-" json:"-"`
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
