package models

import (
	"encoding/json"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/id"
)

type Tasks struct {
	Items map[string]Task

	keys []string
}

type jsonTasks struct {
	Items map[string]Task `json:"values"`
	Keys  []string        `json:"keys"`
}

func (t *Tasks) MarshalJSON() ([]byte, error) {
	if t == nil {
		t = &Tasks{}
	}

	jTasks := jsonTasks{
		Items: t.Items,
		Keys:  t.keys,
	}
	return json.Marshal(jTasks)
}

func (t *Tasks) UnmarshalJSON(data []byte) error {
	if t == nil {
		t = &Tasks{}
	}
	if t.Items == nil {
		t.Items = make(map[string]Task)
	}

	jTasks := jsonTasks{}
	err := json.Unmarshal(data, &jTasks)
	if err != nil {
		return err
	}

	t.Items = jTasks.Items
	t.keys = jTasks.Keys

	return nil
}

func NewTasks() *Tasks {
	return &Tasks{
		Items: make(map[string]Task),
		keys:  []string{},
	}
}

func (t *Tasks) init() {
	if t == nil {
		t = &Tasks{}
	}

	if t.Items == nil {
		t.Items = map[string]Task{}
	}

	if t.keys == nil {
		t.keys = []string{}
	}
}

func (t *Tasks) ToMap() map[string]Task {
	t.init()
	m := make(map[string]Task, len(t.Items))
	for k, v := range t.Items {
		m[k] = v
	}
	return m
}

func (t *Tasks) Len() int {
	if t == nil {
		return 0
	}

	t.init()
	return len(t.Items)
}

func (t *Tasks) IsEmpty() bool {
	return t.Len() == 0
}

func (t *Tasks) Keys() []string {
	t.init()
	keys := make([]string, 0, len(t.keys))
	keys = append(keys, t.keys...)
	return keys
}

func (t *Tasks) Values() []Task {
	t.init()
	values := make([]Task, 0, len(t.Items))
	for _, k := range t.keys {
		values = append(values, t.Items[k])
	}
	return values
}

func (t *Tasks) Add(entry *Task) bool {
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
	t.Items[entry.Id] = *entry

	return true
}

func (t *Tasks) Get(name string) (Task, bool) {
	if t == nil || t.Items == nil {
		return Task{}, false
	}

	entry, ok := t.Items[name]
	if ok {
		return entry, ok
	}

	for _, k := range t.keys {
		if strings.EqualFold(k, name) {
			entry, ok := t.Items[k]
			return entry, ok
		}
	}

	return Task{}, false
}

func (t *Tasks) GetById(idValue string) (Task, bool) {
	if t == nil || t.Items == nil {
		return Task{}, false
	}

	for _, k := range t.keys {
		if strings.EqualFold(id.Convert(k), idValue) {
			entry, ok := t.Items[k]
			return entry, ok
		}
	}

	return Task{}, false
}

func (t *Tasks) Set(entry *Task) bool {
	t.init()

	if entry == nil || entry.Id == "" {
		return false
	}

	_, exists := t.Items[entry.Id]
	if !exists {
		t.keys = append(t.keys, entry.Id)
	}

	t.Items[entry.Id] = *entry

	return true
}

func (t *Tasks) TryGetSlice(key ...string) ([]Task, bool) {
	if t == nil || t.Items == nil {
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

func (t *Tasks) FlattenTasks(targets []string, context string) ([]Task, error) {
	return FlattenTasks(targets, *t, []Task{}, context)
}

func FlattenTasks(targets []string, tasks Tasks, set []Task, context string) ([]Task, error) {

	for _, target := range targets {
		t := target

		var task Task
		found := false

		// prefer context-specific task if context is provided and it is found.
		if context != "" {
			t = target + ":" + context
			task2, ok := tasks.Get(t)
			if ok {
				task = task2
				found = true
			}
		}

		if !found {
			t = target
			task2, ok := tasks.Get(t)
			if !ok {
				return nil, errors.New("Task not found: " + target + " or " + t)
			}

			task = task2
		}

		// ensure dependencies are added first
		if len(task.Needs) > 0 {
			neededTasks, err := FlattenTasks(task.Needs.Ids(), tasks, set, context)
			if err != nil {
				return nil, err
			}
			set = neededTasks
		}

		// Treat hooks as something that always must be added around the main task
		// even if they were already added as part of dependencies.

		// only add before hooks if they task is setup for hooks
		if len(task.Hooks.Before) > 0 {
			for _, beforeHookSuffix := range task.Hooks.Before {
				// use task.Id to ensure that context-specific hooks are resolved
				// if the main task is context-specific, otherwise use the base task.
				hookTaskName := task.Id + ":" + beforeHookSuffix
				beforeTask, ok := tasks.Get(hookTaskName)
				if ok {
					set = append(set, beforeTask)
				}
			}
		}

		added := false
		for _, task2 := range set {
			if task.Id == task2.Id {
				added = true
				break
			}
		}

		if !added {
			set = append(set, task)
		}

		// only add after hooks if they task is setup for hooks
		if len(task.Hooks.After) > 0 {
			for _, afterHookSuffix := range task.Hooks.After {
				// use task.Id to ensure that context-specific hooks are resolved
				// if the main task is context-specific, otherwise use the base task.
				hookTaskName := task.Id + ":" + afterHookSuffix
				afterTask, ok := tasks.Get(hookTaskName)
				if ok {
					set = append(set, afterTask)
				}
			}
		}
	}

	return set, nil
}

func FindCyclicalReferences(tasks []Task) []Task {
	stack := []Task{}
	cycles := []Task{}

	var resolve func(task Task) bool
	resolve = func(task Task) bool {
		for _, t := range stack {
			if task.Id == t.Id {
				return false
			}
		}

		stack = append(stack, task)
		if len(task.Needs) > 0 {
			for _, need := range task.Needs {
				for _, nextTask := range tasks {
					if nextTask.Id == need.Id {
						if !resolve(nextTask) {
							return false
						}
					}
				}
			}
		}

		stack = stack[:len(stack)-1]
		return true
	}

	for _, task := range tasks {
		if !resolve(task) {
			cycles = append(cycles, task)
		}
	}

	return cycles
}
