package types

import (
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type OsInfo struct {
	Platform     string `json:"platform,omitempty"`
	Arch         string `json:"arch,omitempty"`
	Variant      string `json:"variant,omitempty"`
	Family       string `json:"family,omitempty"`
	Codename     string `json:"codename,omitempty"`
	Version      string `json:"version,omitempty"`
	BuildVersion string `json:"buildVersion,omitempty"`
}

func (o *OsInfo) UnmarshalYAML(node *yaml.Node) error {
	if o == nil {
		o = &OsInfo{}
	}

	if node.Kind == yaml.ScalarNode {
		o.Platform = node.Value
		return nil
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			key := keyNode.Value
			switch key {
			case "platform":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'platform' field")
				}
				o.Platform = valueNode.Value
			case "arch":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'arch' field")
				}
				o.Arch = valueNode.Value
			case "variant":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'variant' field")
				}
				o.Variant = valueNode.Value
			case "family":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'family' field")
				}
				o.Family = valueNode.Value
			case "codename":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'codename' field")
				}
				o.Codename = valueNode.Value
			case "version":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'version' field")
				}
				o.Version = valueNode.Value
			case "build_version", "build-version", "buildVersion":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'build_version' field")
				}
				o.BuildVersion = valueNode.Value
			default:
				return errors.YamlErrorf(keyNode, "unexpected field '%s' in OsInfo", key)
			}
		}
	}

	switch strings.ToLower(o.Platform) {
	case "windows", "win", "win32", "win64":
		o.Platform = "windows"
	case "linux":
		o.Platform = "linux"
	case "darwin", "osx", "mac", "macos":
		o.Platform = "darwin"
	default:
		return errors.YamlErrorf(node, "unsupported platform '%s'", o.Platform)
	}

	return errors.NewYamlError(node, "expected yaml scalar or mapping for 'OsInfo' node")
}
