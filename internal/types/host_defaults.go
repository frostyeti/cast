package types

import (
	"fmt"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type HostDefaults struct {
	Port         *uint    `yaml:"port,omitempty" json:"port,omitempty"`
	User         *string  `yaml:"user,omitempty" json:"user,omitempty"`
	IdentityFile *string  `yaml:"identity,omitempty" json:"identity,omitempty"`
	Password     *string  `yaml:"password,omitempty" json:"password,omitempty"`
	Tags         []string `yaml:"groups,omitempty" json:"tags,omitempty"`
	Meta         *Meta    `yaml:"meta,omitempty" json:"meta,omitempty"`
	OS           *OsInfo  `yaml:"os,omitempty" json:"os,omitempty"`
}

func (hd *HostDefaults) UnmarshalYAML(value *yaml.Node) error {
	if hd == nil {
		hd = &HostDefaults{}
	}

	if value.Kind != yaml.MappingNode {
		return errors.YamlErrorf(value, "expected yaml scalar or mapping for host entry")
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		key := keyNode.Value
		switch key {
		case "port":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'port' field")
			}
			var port uint
			_, err := fmt.Sscanf(valueNode.Value, "%d", &port)
			if err != nil {
				return errors.YamlErrorf(valueNode, "invalid port number: %v", err)
			}
			hd.Port = &port
		case "user":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'user' field")
			}
			user := valueNode.Value
			hd.User = &user
		case "identityFile", "identity_file", "identity-file", "identity":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'identity-file' field")
			}
			identityFile := valueNode.Value
			hd.IdentityFile = &identityFile
		case "password-variable", "passwordVariable", "password_variable", "password":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'password' field")
			}
			password := valueNode.Value
			hd.Password = &password
		case "tags":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.YamlErrorf(valueNode, "expected yaml sequence for 'tags' field")
			}
			tags := make([]string, 0)
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(item, "expected yaml scalar for tag item")
				}
				tags = append(tags, item.Value)
			}
			hd.Tags = tags
		case "meta":
			if valueNode.Kind != yaml.MappingNode {
				return errors.YamlErrorf(valueNode, "expected yaml mapping for 'meta' field")
			}
			meta := &Meta{}
			meta.keys = make([]string, 0)
			meta.values = make(map[string]interface{})
			for j := 0; j < len(valueNode.Content); j += 2 {
				metaKeyNode := valueNode.Content[j]
				metaValueNode := valueNode.Content[j+1]

				if metaKeyNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(metaKeyNode, "expected yaml scalar for meta key")
				}
				metaKey := metaKeyNode.Value

				var metaValue interface{}
				if err := metaValueNode.Decode(&metaValue); err != nil {
					return errors.YamlErrorf(metaValueNode, "failed to decode meta value: %v", err)
				}

				meta.Set(metaKey, metaValue)
			}
			hd.Meta = meta
		case "os":
			var os OsInfo
			if err := valueNode.Decode(&os); err != nil {
				return errors.YamlErrorf(valueNode, "failed to decode 'os' field: %v", err)
			}
			hd.OS = &os
		default:
			return errors.YamlErrorf(keyNode, "unexpected field '%s' in host entry", key)
		}
	}

	return nil
}
