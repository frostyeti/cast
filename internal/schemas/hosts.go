package schemas

import (
	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Hosts struct {
	values map[string]HostElement
	keys   []string
}

type jsonHosts struct {
	Values map[string]HostElement `json:"values"`
	Keys   []string               `json:"keys"`
}

func (h *Hosts) MarshalJSON() ([]byte, error) {
	if h == nil {
		h = &Hosts{}
	}

	jhosts := jsonHosts{
		Values: h.values,
		Keys:   h.keys,
	}
	return yaml.Marshal(jhosts)
}

func (h *Hosts) UnmarshalJSON(data []byte) error {
	if h == nil {
		h = &Hosts{}
	}
	if h.values == nil {
		h.values = make(map[string]HostElement)
	}

	jhosts := jsonHosts{}
	err := yaml.Unmarshal(data, &jhosts)
	if err != nil {
		return err
	}

	h.values = jhosts.Values
	h.keys = jhosts.Keys

	return nil
}

func NewHosts() *Hosts {
	return &Hosts{
		values: make(map[string]HostElement),
		keys:   []string{},
	}
}

func (h *Hosts) Get(host string) (HostElement, bool) {
	if h == nil || h.values == nil {
		return HostElement{}, false
	}
	val, ok := h.values[host]
	return val, ok
}

func (h *Hosts) Keys() []string {
	if h == nil {
		return []string{}
	}
	return h.keys
}

func (h *Hosts) Set(host string, entry HostElement) {
	if h == nil {
		h = &Hosts{}
	}
	if h.values == nil {
		h.values = make(map[string]HostElement)
		h.keys = []string{}
	}
	if _, exists := h.values[host]; !exists {
		h.keys = append(h.keys, host)
	}
	h.values[host] = entry
}

func (h *Hosts) Len() int {
	if h == nil {
		return 0
	}
	return len(h.keys)
}

func (h *Hosts) Values() []HostElement {
	if h == nil {
		return []HostElement{}
	}

	values := make([]HostElement, 0, len(h.keys))
	for _, key := range h.keys {
		values = append(values, h.values[key])
	}
	return values
}

func (h *Hosts) UnmarshalYAML(node *yaml.Node) error {
	if h == nil {
		h = &Hosts{}
	}

	if node.Kind == yaml.SequenceNode {
		for _, itemNode := range node.Content {
			var he HostElement
			err := itemNode.Decode(&he)
			if err != nil {
				return errors.YamlErrorf(*itemNode, "failed to decode host entry: %v", err)
			}
			h.Set(he.Host, he)
		}
		return nil
	}

	if node.Kind != yaml.MappingNode {
		return errors.YamlErrorf(*node, "expected yaml mapping for hosts")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		var he HostElement
		err := valueNode.Decode(&he)
		if err != nil {
			return errors.YamlErrorf(*valueNode, "failed to decode host entry for '%s': %v", key, err)
		}
		h.Set(key, he)
	}

	return nil
}
