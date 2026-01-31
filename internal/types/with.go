package types

import (
	"maps"
	"strconv"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type With struct {
	values map[string]interface{}
	keys   []string
}

func (w *With) UnmarshalYAML(value *yaml.Node) error {
	if w == nil {
		w = NewWith()
	}

	if value.Kind != yaml.MappingNode {
		return errors.NewYamlError(value, "expected yaml mapping for 'with' field")
	}

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valueNode := value.Content[i+1]

		key := keyNode.Value
		var val interface{}
		if err := valueNode.Decode(&val); err != nil {
			return errors.NewYamlError(valueNode, "failed to decode 'with' field value: "+err.Error())
		}

		w.Add(key, val)
	}

	return nil
}

func NewWith() *With {
	return &With{
		values: map[string]interface{}{},
		keys:   []string{},
	}
}

func (w *With) ToMap() map[string]interface{} {
	if w == nil {
		w = NewWith()
	}
	m := make(map[string]interface{})
	maps.Copy(m, w.values)
	return m
}

func (w *With) init() {
	if w == nil {
		w = NewWith()
	}

	if w.values == nil {
		w.values = map[string]interface{}{}
	}

	if w.keys == nil {
		w.keys = []string{}
	}
}

func (w *With) Keys() []string {
	if w == nil {
		return []string{}
	}
	keys := make([]string, 0, len(w.keys))
	keys = append(keys, w.keys...)
	return keys
}

func (w *With) Values() []interface{} {
	if w == nil {
		return []interface{}{}
	}
	values := make([]interface{}, 0, len(w.values))
	for _, k := range w.keys {
		values = append(values, w.values[k])
	}
	return values
}

func (w *With) Len() int {
	if w == nil {
		return 0
	}

	w.init()
	return len(w.keys)
}

func (w *With) Add(key string, value interface{}) {
	if w == nil {
		w = NewWith()
	}

	if _, exists := w.values[key]; !exists {
		w.keys = append(w.keys, key)
		w.values[key] = value
	}
}

func (w *With) Set(key string, value interface{}) {
	if w == nil {
		w = NewWith()
	}

	if _, exists := w.values[key]; !exists {
		w.keys = append(w.keys, key)
	}
	w.values[key] = value
}

func (w *With) Get(key string) (interface{}, bool) {
	if w == nil {
		w = NewWith()
	}
	val, exists := w.values[key]
	return val, exists
}

func (w *With) Has(key string) bool {
	if w == nil {
		w = NewWith()
	}
	_, exists := w.values[key]
	return exists
}

func (w *With) GetString(key string) (string, bool) {
	if w == nil {
		w = NewWith()
	}
	val, exists := w.values[key]
	if !exists {
		return "", false
	}
	strVal, ok := val.(string)
	return strVal, ok
}

func (w *With) GetInt(key string) (int, bool) {
	if w == nil {
		w = NewWith()
	}
	val, exists := w.values[key]
	if !exists {
		return 0, false
	}
	intVal, ok := val.(int)
	if ok {
		return intVal, true
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

func (w *With) GetFloat64(key string) (float64, bool) {
	if w == nil {
		w = NewWith()
	}
	val, exists := w.values[key]
	if !exists {
		return 0, false
	}
	floatVal, ok := val.(float64)
	if ok {
		return floatVal, true
	}

	strVal, ok := val.(string)
	if !ok {
		return 0, false
	}

	parsed, err := strconv.ParseFloat(strVal, 64)
	if err != nil {
		return 0, false
	}

	return parsed, true
}

func (w *With) GetBool(key string) (bool, bool) {
	if w == nil {
		w = NewWith()
	}
	val, exists := w.values[key]
	if !exists {
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

func (w *With) GetStringSlice(key string) ([]string, bool) {
	if w == nil {
		w = NewWith()
	}
	val, exists := w.values[key]
	if !exists {
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

func (w *With) Delete(key string) {
	if w == nil {
		w = NewWith()
	}
	if _, exists := w.values[key]; exists {
		delete(w.values, key)
		// Remove from keys slice
		for i, k := range w.keys {
			if k == key {
				w.keys = append(w.keys[:i], w.keys[i+1:]...)
				break
			}
		}
	}
}

func (w *With) Merge(other *With) {
	if w == nil {
		w = NewWith()
	}

	other.init()

	for _, k := range other.keys {
		if _, exists := w.values[k]; !exists {
			w.keys = append(w.keys, k)
		}
		w.values[k] = other.values[k]
	}
}

func (w *With) Clone() *With {

	clone := NewWith()
	if w == nil {
		return clone
	}

	for k, v := range w.values {
		clone.values[k] = v
	}
	clone.keys = append(clone.keys, w.keys...)
	return clone
}
