package schemas

import (
	"time"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type ProjectDiscovery struct {
	Include      []string     `yaml:"include,omitempty"`
	Exclude      []string     `yaml:"exclude,omitempty"`
	AutoDiscover AutoDiscover `yaml:"cache,omitempty"`
	Cache        []CastfileInfo
}

type AutoDiscover struct {
	Enabled bool          `yaml:"enabled,omitempty"`
	Expires time.Duration `yaml:"expires,omitempty"`
}

func (c *AutoDiscover) MarshalYAML() (interface{}, error) {
	mapping := make(map[string]interface{})
	mapping["enabled"] = c.Enabled
	mapping["expires"] = c.Expires.String()
	return mapping, nil
}

func (c *AutoDiscover) UnmarshalYAML(node *yaml.Node) error {
	if c == nil {
		c = &AutoDiscover{}
	}

	c.Expires = 24 * time.Hour

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "project discovery cache must be mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "enabled":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'enabled' field")
			}
			switch valueNode.Value {
			case "true", "1":
				c.Enabled = true
			case "false", "0":
				c.Enabled = false
			default:
				return errors.NewYamlError(valueNode, "expected 'true' or 'false' for 'enabled' field")
			}
		case "expires":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'expires' field")
			}
			duration, err := time.ParseDuration(valueNode.Value)
			if err != nil {
				return errors.NewYamlError(valueNode, "invalid duration format for 'expires' field")
			}
			c.Expires = duration
		}
	}

	return nil
}

func (c *ProjectDiscovery) MarshalYAML() (interface{}, error) {
	m := make(map[string]interface{})
	if len(c.Include) > 0 {
		m["include"] = c.Include
	}
	if len(c.Exclude) > 0 {
		m["exclude"] = c.Exclude
	}
	if len(c.Cache) > 0 {
		m["cache"] = c.Cache
	}
	m["auto-discover"] = c.AutoDiscover
	return m, nil
}

func (p *ProjectDiscovery) UnmarshalYAML(node *yaml.Node) error {
	if p == nil {
		p = &ProjectDiscovery{}
	}

	p.Include = []string{"**"}
	p.Exclude = []string{"**/.*/**", "**/bin/**", "**/obj/**", "**/vendor/**", "**/node_modules/**"}
	p.Cache = []CastfileInfo{}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "projects must be mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "include":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'include' field")
			}
			if len(valueNode.Content) > 0 {
				p.Include = []string{}
				for _, item := range valueNode.Content {
					if item.Kind != yaml.ScalarNode {
						return errors.NewYamlError(item, "expected yaml scalar in 'include' sequence")
					}
					p.Include = append(p.Include, item.Value)
				}
			}
		case "exclude":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'exclude' field")
			}

			if len(valueNode.Content) > 0 {
				p.Exclude = []string{}
				for _, item := range valueNode.Content {
					if item.Kind != yaml.ScalarNode {
						return errors.NewYamlError(item, "expected yaml scalar in 'exclude' sequence")
					}
					p.Exclude = append(p.Exclude, item.Value)
				}
			}
		case "cache":
			items := []CastfileInfo{}
			if valueNode.Kind != yaml.SequenceNode {
				return errors.NewYamlError(valueNode, "expected yaml sequence for 'cache' field")
			}
			for _, item := range valueNode.Content {
				var ci CastfileInfo
				if err := item.Decode(&ci); err != nil {
					return err
				}
				items = append(items, ci)
			}
			p.Cache = items

		case "auto-discover", "discover", "auto_discover":
			if err := p.AutoDiscover.UnmarshalYAML(valueNode); err != nil {
				return err
			}
		default:
			return errors.YamlErrorf(keyNode, "unexpected field '%s' in projects", keyNode.Value)
		}
	}

	return nil
}
