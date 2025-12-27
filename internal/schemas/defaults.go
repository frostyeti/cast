package schemas

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type WorkspaceDefaults struct {
	Context string        `yaml:"context,omitempty"`
	Shell   string        `yaml:"shell,omitempty"`
	Cache   CacheDefaults `yaml:"cache,omitempty"`
	Remote  bool          `yaml:"remote,omitempty"`
}

func (wd *WorkspaceDefaults) MarshalYAML() (interface{}, error) {
	mapping := make(map[string]interface{})
	mapping["context"] = wd.Context
	mapping["shell"] = wd.Shell

	cache, err := wd.Cache.MarshalYAML()
	if err != nil {
		return nil, err
	}
	mapping["cache"] = cache

	mapping["remote"] = wd.Remote

	return mapping, nil
}

type CacheDefaults struct {
	Enabled *bool `yaml:"enabled,omitempty"`
}

func (cd *CacheDefaults) MarshalYAML() (interface{}, error) {
	mapping := make(map[string]interface{})
	if cd.Enabled != nil {
		mapping["enabled"] = *cd.Enabled
	}
	return mapping, nil
}

func (cd *CacheDefaults) UnmarshalYAML(node *yaml.Node) error {
	if cd == nil {
		cd = &CacheDefaults{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "cache defaults must be mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "enabled":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'enabled' field")
			}
			if valueNode.Value == "true" || valueNode.Value == "1" {
				trueVal := true
				cd.Enabled = &trueVal
			} else if valueNode.Value == "false" || valueNode.Value == "0" {
				falseVal := false
				cd.Enabled = &falseVal
			} else {
				return errors.NewYamlError(valueNode, "expected 'true' or 'false' for 'enabled' field")
			}
		}
	}

	return nil
}

func (wd *WorkspaceDefaults) UnmarshalYAML(node *yaml.Node) error {
	if wd == nil {
		wd = &WorkspaceDefaults{}
	}

	if wd.Context == "" {
		wd.Context = "default"
	}

	if wd.Shell == "" {
		wd.Shell = "shell"
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "defaults must be mapping node")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "context":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'context' field")
			}
			wd.Context = valueNode.Value
		case "shell":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'shell' field")
			}
			wd.Shell = valueNode.Value
		case "cache":
			if err := wd.Cache.UnmarshalYAML(valueNode); err != nil {
				return err
			}
		case "remote":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.NewYamlError(valueNode, "expected yaml scalar for 'remote' field")
			}
			if valueNode.Value == "true" || valueNode.Value == "1" {
				wd.Remote = true
			} else if valueNode.Value == "false" || valueNode.Value == "0" {
				wd.Remote = false
			} else {
				return errors.NewYamlError(valueNode, "expected 'true' or 'false' for 'remote' field")
			}
		}
	}

	return nil
}
