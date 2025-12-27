package schemas

import (
	"os"

	"go.yaml.in/yaml/v4"
)

type Castfile struct {
	Imports  []Import `yaml:"imports,omitempty"`
	Id       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Alias    string
	Desc     *string `yaml:"description,omitempty"`
	Version  *string `yaml:"version,omitempty"`
	Meta     map[string]any
	Tags     []string           `yaml:"tags,omitempty"`
	Defaults *WorkspaceDefaults `yaml:"defaults,omitempty"`
	Env      *Env               `yaml:"env,omitempty"`
	Tasks    TaskMap            `yaml:"tasks,omitempty"`
	Needs    []Need             `yaml:"needs,omitempty"`
}

func (c *Castfile) UnmarshalYAML(node *yaml.Node) error {
	if c == nil {
		c = &Castfile{}
	}

	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "imports":
			var imports []Import
			if err := valueNode.Decode(&imports); err != nil {
				return err
			}
			c.Imports = imports
		case "meta":
			var meta map[string]any
			if err := valueNode.Decode(&meta); err != nil {
				return err
			}
			c.Meta = meta
		case "id":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			c.Id = valueNode.Value
		case "name":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			c.Name = valueNode.Value
		case "description":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			desc := valueNode.Value
			c.Desc = &desc
		case "version":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			version := valueNode.Value
			c.Version = &version
		case "tags":
			if valueNode.Kind != yaml.SequenceNode {
				return nil
			}
			tags := []string{}
			for _, tagNode := range valueNode.Content {
				if tagNode.Kind != yaml.ScalarNode {
					continue
				}
				tags = append(tags, tagNode.Value)
			}
			c.Tags = tags
		case "defaults":
			var wd WorkspaceDefaults
			if err := valueNode.Decode(&wd); err != nil {
				return err
			}
			c.Defaults = &wd
		case "env":
			var env Env
			if err := valueNode.Decode(&env); err != nil {
				return err
			}
			c.Env = &env
		case "tasks":
			var tasks TaskMap
			if err := valueNode.Decode(&tasks); err != nil {
				return err
			}
			c.Tasks = tasks
		case "needs":
			var needs []Need
			if err := valueNode.Decode(&needs); err != nil {
				return err
			}
			c.Needs = needs
		}
	}

	return nil
}

func (c *Castfile) MarshalYAML() (any, error) {
	m := make(map[string]any)

	if len(c.Imports) > 0 {
		m["imports"] = c.Imports
	}
	m["id"] = c.Id
	m["name"] = c.Name
	if c.Desc != nil {
		m["description"] = *c.Desc
	}
	if c.Version != nil {
		m["version"] = *c.Version
	}
	if len(c.Meta) > 0 {
		m["meta"] = c.Meta
	}
	if len(c.Tags) > 0 {
		m["tags"] = c.Tags
	}
	if c.Defaults != nil {
		m["defaults"] = c.Defaults
	}
	if c.Env != nil {
		m["env"] = c.Env
	}
	if c.Tasks.Len() > 0 {
		m["tasks"] = c.Tasks
	}
	if len(c.Needs) > 0 {
		m["needs"] = c.Needs
	}

	return m, nil
}

type CastfileInfo struct {
	Path    string
	Id      string
	Name    string
	Alias   string
	Version string
	Desc    string
	Needs   []Need
	Tags    []string
}

func (cf *CastfileInfo) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	cf.Path = path
	return yaml.Unmarshal(data, cf)
}

func (cf *CastfileInfo) UnmarshalYAML(node *yaml.Node) error {
	if cf == nil {
		cf = &CastfileInfo{}
	}

	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			cf.Id = valueNode.Value
		case "name":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			cf.Name = valueNode.Value
		case "alias":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			cf.Alias = valueNode.Value
		case "version":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			cf.Version = valueNode.Value
		case "description":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			cf.Desc = valueNode.Value
		case "needs":
			var needs []Need
			if err := valueNode.Decode(&needs); err != nil {
				return err
			}
			cf.Needs = needs
		case "tags":
			if valueNode.Kind != yaml.SequenceNode {
				return nil
			}
			tags := []string{}
			for _, tagNode := range valueNode.Content {
				if tagNode.Kind != yaml.ScalarNode {
					continue
				}
				tags = append(tags, tagNode.Value)
			}
			cf.Tags = tags
		}
	}

	return nil
}

func (cf *CastfileInfo) MarshalYAML() (any, error) {
	m := make(map[string]any)

	m["id"] = cf.Id
	m["name"] = cf.Name
	if cf.Alias != "" {
		m["alias"] = cf.Alias
	}
	if cf.Version != "" {
		m["version"] = cf.Version
	}
	if cf.Desc != "" {
		m["description"] = cf.Desc
	}
	if len(cf.Needs) > 0 {
		m["needs"] = cf.Needs
	}
	if len(cf.Tags) > 0 {
		m["tags"] = cf.Tags
	}

	return m, nil
}
