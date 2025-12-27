package schemas

import (
	"encoding/json"
	"strings"

	"github.com/frostyeti/cast/internal/id"
)

type TaskMap struct {
	values map[string]Task

	keys []string
}

type jsonTaskMap struct {
	Values map[string]Task `json:"values"`
	Keys   []string        `json:"keys"`
}

func (t *TaskMap) MarshalJSON() ([]byte, error) {
	if t == nil {
		t = &TaskMap{}
	}

	jtaskMap := jsonTaskMap{
		Values: t.values,
		Keys:   t.keys,
	}
	return json.Marshal(jtaskMap)
}

func (t *TaskMap) UnmarshalJSON(data []byte) error {
	if t == nil {
		t = &TaskMap{}
	}
	if t.values == nil {
		t.values = make(map[string]Task)
	}

	jtaskMap := jsonTaskMap{}
	err := json.Unmarshal(data, &jtaskMap)
	if err != nil {
		return err
	}

	t.values = jtaskMap.Values
	t.keys = jtaskMap.Keys

	return nil
}

func NewTaskMap() *TaskMap {
	return &TaskMap{
		values: make(map[string]Task),
		keys:   []string{},
	}
}

func (t *TaskMap) init() {
	if t == nil {
		t = &TaskMap{}
	}

	if t.values == nil {
		t.values = map[string]Task{}
	}

	if t.keys == nil {
		t.keys = []string{}
	}
}

func (t *TaskMap) ToMap() map[string]Task {
	t.init()
	m := make(map[string]Task, len(t.values))
	for k, v := range t.values {
		m[k] = v
	}
	return m
}

func (t *TaskMap) Len() int {
	if t == nil {
		return 0
	}

	t.init()
	return len(t.values)
}

func (t *TaskMap) IsEmpty() bool {
	return t.Len() == 0
}

func (t *TaskMap) Keys() []string {
	t.init()
	keys := make([]string, 0, len(t.keys))
	keys = append(keys, t.keys...)
	return keys
}

func (t *TaskMap) Values() []Task {
	t.init()
	values := make([]Task, 0, len(t.values))
	for _, k := range t.keys {
		values = append(values, t.values[k])
	}
	return values
}

func (t *TaskMap) Add(entry *Task) bool {
	t.init()

	if entry == nil || entry.Id == "" {
		return false
	}

	for _, k := range t.keys {
		if strings.EqualFold(k, entry.Id) {
			return false
		}
	}

	t.keys = append(t.keys, entry.Id)
	t.values[entry.Id] = *entry

	return true
}

func (t *TaskMap) Get(name string) (Task, bool) {
	if t == nil || t.values == nil {
		return Task{}, false
	}

	entry, ok := t.values[name]
	if ok {
		return entry, ok
	}

	for _, k := range t.keys {
		if strings.EqualFold(k, name) {
			entry, ok := t.values[k]
			return entry, ok
		}
	}

	return Task{}, false
}

func (t *TaskMap) GetById(idValue string) (Task, bool) {
	if t == nil || t.values == nil {
		return Task{}, false
	}

	for _, k := range t.keys {
		if strings.EqualFold(id.Convert(k), idValue) {
			entry, ok := t.values[k]
			return entry, ok
		}
	}

	return Task{}, false
}

func (t *TaskMap) Set(entry *Task) bool {
	t.init()

	if entry == nil || entry.Id == "" {
		return false
	}

	_, exists := t.values[entry.Id]
	if !exists {
		t.keys = append(t.keys, entry.Id)
	}

	t.values[entry.Id] = *entry

	return true
}

func (t *TaskMap) TryGetSlice(key ...string) ([]Task, bool) {
	if t == nil || t.values == nil {
		return nil, false
	}

	results := make([]Task, 0, len(key))
	for _, k := range key {
		s, ok := t.Get(k)
		if !ok {
			continue
		}
		results = append(results, s)
	}
	return results, len(results) > 0
}
