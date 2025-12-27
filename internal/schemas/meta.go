package schemas

import (
	"strconv"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Meta struct {
	values map[string]interface{}
	keys   []string
}

type jsonMeta struct {
	Values map[string]interface{} `json:"values"`
	Keys   []string               `json:"keys"`
}

func (m *Meta) MarshalJSON() ([]byte, error) {
	if m == nil {
		m = &Meta{}
	}

	jmeta := jsonMeta{
		Values: m.values,
		Keys:   m.keys,
	}
	return yaml.Marshal(jmeta)
}

func (m *Meta) UnmarshalJSON(data []byte) error {
	if m == nil {
		m = &Meta{}
	}
	if m.values == nil {
		m.values = make(map[string]interface{})
	}

	jmeta := jsonMeta{}
	err := yaml.Unmarshal(data, &jmeta)
	if err != nil {
		return err
	}

	m.values = jmeta.Values
	m.keys = jmeta.Keys

	return nil
}

func (m *Meta) Get(key string) (interface{}, bool) {
	if m == nil || m.values == nil {
		return nil, false
	}
	val, ok := m.values[key]
	return val, ok
}

func (m *Meta) GetString(key string) (string, bool) {
	if m == nil || m.values == nil {
		return "", false
	}
	val, ok := m.values[key]
	if !ok {
		return "", false
	}
	strVal, ok := val.(string)
	return strVal, ok
}

func (m *Meta) GetInt(key string) (int, bool) {
	if m == nil || m.values == nil {
		return 0, false
	}
	val, ok := m.values[key]
	if !ok {
		return 0, false
	}
	intVal, ok := val.(int)
	if ok {
		return intVal, true
	}

	floatVal, ok := val.(float64)
	if ok {
		return int(floatVal), true
	}

	strVal, ok := val.(string)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(strVal)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (m *Meta) GetBool(key string) (bool, bool) {
	if m == nil || m.values == nil {
		return false, false
	}
	val, ok := m.values[key]
	if !ok {
		return false, false
	}
	boolVal, ok := val.(bool)
	if ok {
		return boolVal, true
	}

	strVal, ok := val.(string)
	if !ok {
		return false, false
	}
	parsed, err := strconv.ParseBool(strVal)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func (m *Meta) GetStringSlice(key string) ([]string, bool) {
	if m == nil || m.values == nil {
		return nil, false
	}
	val, ok := m.values[key]
	if !ok {
		return nil, false
	}
	sliceVal, ok := val.([]string)
	if ok {
		return sliceVal, true
	}

	interfaceSlice, ok := val.([]interface{})
	if !ok {
		return nil, false
	}

	strSlice := make([]string, 0, len(interfaceSlice))
	for _, item := range interfaceSlice {
		strItem, ok := item.(string)
		if !ok {
			return nil, false
		}
		strSlice = append(strSlice, strItem)
	}

	return strSlice, true
}

func (m *Meta) Set(key string, value interface{}) {
	if m == nil {
		m = &Meta{}
	}
	if m.values == nil {
		m.values = make(map[string]interface{})
	}
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *Meta) Keys() []string {
	if m == nil {
		return []string{}
	}
	return m.keys
}

func (m *Meta) Values() []interface{} {
	if m == nil {
		return []interface{}{}
	}

	values := make([]interface{}, 0, len(m.keys))
	for _, key := range m.keys {
		values = append(values, m.values[key])
	}
	return values
}

func (m *Meta) Len() int {
	if m == nil {
		return 0
	}
	return len(m.keys)
}

func (m *Meta) IsEmpty() bool {
	return m == nil || len(m.keys) == 0
}

func (m *Meta) UnmarshalYAML(node *yaml.Node) error {
	if m == nil {
		m = &Meta{}
	}
	if m.values == nil {
		m.values = make(map[string]interface{})
	}

	if node.Kind != yaml.MappingNode {
		return errors.YamlErrorf(*node, "expected yaml mapping for meta")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		var value interface{}
		err := valueNode.Decode(&value)
		if err != nil {
			return err
		}

		m.Set(key, value)
	}

	return nil
}
