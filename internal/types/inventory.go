package types

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type InventoryDefaults map[string]HostDefaults

type Inventory struct {
	Defaults  InventoryDefaults
	Hosts     map[string]HostInfo
	HostOrder []string
}

func (iv *Inventory) UnmarshalYAML(node *yaml.Node) error {
	if iv == nil {
		iv = &Inventory{
			Hosts:    make(map[string]HostInfo),
			Defaults: make(InventoryDefaults),
		}
	}

	if iv.Hosts == nil {
		iv.Hosts = make(map[string]HostInfo)
	}

	if iv.Defaults == nil {
		iv.Defaults = make(InventoryDefaults)
	}

	if iv.HostOrder == nil {
		iv.HostOrder = []string{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.NewYamlError(node, "invalid yaml node for Inventory")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		switch key {
		case "defaults":
			if valueNode.Kind != yaml.MappingNode {
				return errors.NewYamlError(valueNode, "expected yaml mapping for 'defaults' field")
			}
			for j := 0; j < len(valueNode.Content); j += 2 {
				defaultKeyNode := valueNode.Content[j]
				defaultValueNode := valueNode.Content[j+1]

				defaultKey := defaultKeyNode.Value
				var hostDefaults HostDefaults
				if err := defaultValueNode.Decode(&hostDefaults); err != nil {
					return err
				}
				iv.Defaults[defaultKey] = hostDefaults
			}
		case "hosts":
			if valueNode.Kind == yaml.SequenceNode {
				for j := 0; j < len(valueNode.Content); j++ {
					hostNode := valueNode.Content[j]
					var hostInfo *HostInfo
					if err := hostNode.Decode(&hostInfo); err != nil {
						return err
					}
					host := (*hostInfo).Host

					iv.Hosts[host] = *hostInfo
					iv.HostOrder = append(iv.HostOrder, host)
				}
				continue
			}
			if valueNode.Kind != yaml.MappingNode {
				return errors.NewYamlError(valueNode, "expected yaml mapping or sequence for 'hosts' field")
			}

			for j := 0; j < len(valueNode.Content); j += 2 {
				hostKeyNode := valueNode.Content[j]
				hostValueNode := valueNode.Content[j+1]

				hostKey := hostKeyNode.Value
				var hostInfo HostInfo
				if err := hostValueNode.Decode(&hostInfo); err != nil {
					return err
				}
				iv.Hosts[hostKey] = hostInfo
				iv.HostOrder = append(iv.HostOrder, hostKey)
			}
		}
	}

	return nil
}
