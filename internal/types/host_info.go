package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type HostInfo struct {
	Host         string   `json:"host:omitempty"`
	Port         *uint    `json:"port:omitempty"`
	User         *string  `json:"user:omitempty"`
	IdentityFile *string  `json:"identityFile:omitempty"`
	Password     *string  `json:"password:omitempty"`
	Tags         []string `json:"tags:omitempty"`
	Meta         *Meta    `json:"meta:omitempty"`
	OS           *OsInfo  `json:"os:omitempty"`
	Defaults     string   `json:"defaults:omitempty"`
}

func (he *HostInfo) UnmarshalYAML(node *yaml.Node) error {
	if he == nil {
		he = &HostInfo{}
	}

	if node.Kind == yaml.ScalarNode {
		hostStr := node.Value
		if strings.ContainsRune(hostStr, '@') {
			parts := strings.SplitN(hostStr, "@", 2)
			user := parts[0]
			he.User = &user
			hostStr = parts[1]
		}

		if strings.ContainsRune(hostStr, ':') {
			parts := strings.SplitN(hostStr, ":", 2)
			hostStr = parts[0]
			port := uint(0)
			portV, err := strconv.Atoi(parts[1])
			if err != nil {
				return errors.YamlErrorf(node, "invalid port number: %v", err)
			}
			if portV < 0 || portV > 65535 {
				return errors.YamlErrorf(node, "port number out of range: %d", portV)
			}

			port = uint(portV)
			he.Port = &port
		}

		he.Host = hostStr

		return nil
	}

	if node.Kind != yaml.MappingNode {
		return errors.YamlErrorf(node, "expected yaml scalar or mapping for host entry")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		switch key {
		case "host":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'host' field")
			}
			he.Host = valueNode.Value
		case "port":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'port' field")
			}
			var port uint
			_, err := fmt.Sscanf(valueNode.Value, "%d", &port)
			if err != nil {
				return errors.YamlErrorf(valueNode, "invalid port number: %v", err)
			}
			he.Port = &port
		case "user":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'user' field")
			}
			user := valueNode.Value
			he.User = &user
		case "identityFile", "identity_file", "identity-file", "identity":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'identity-file' field")
			}
			identityFile := valueNode.Value
			he.IdentityFile = &identityFile
		case "password-variable", "passwordVariable", "password_variable", "password":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'password' field")
			}
			password := valueNode.Value
			he.Password = &password
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
			he.Tags = tags
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
			he.Meta = meta
		case "os":
			var os OsInfo
			if err := valueNode.Decode(&os); err != nil {
				return errors.YamlErrorf(valueNode, "failed to decode 'os' field: %v", err)
			}
			he.OS = &os
		case "defaults":
			if valueNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(valueNode, "expected yaml scalar for 'defaults' field")
			}
			he.Defaults = valueNode.Value
		default:
			return errors.YamlErrorf(keyNode, "unexpected field '%s' in host entry", key)
		}
	}

	return nil
}
