package types

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/frostyeti/cast/internal/id"
	"go.yaml.in/yaml/v4"
)

type JobMap struct {
	values map[string]Job

	keys []string
}

type jsonJobMap struct {
	Values map[string]Job `json:"values"`
	Keys   []string       `json:"keys"`
}

func (t *JobMap) MarshalJSON() ([]byte, error) {
	if t == nil {
		t = &JobMap{}
	}

	jJobMap := jsonJobMap{
		Values: t.values,
		Keys:   t.keys,
	}
	return json.Marshal(jJobMap)
}

func (t *JobMap) UnmarshalJSON(data []byte) error {
	if t == nil {
		t = &JobMap{}
	}
	if t.values == nil {
		t.values = make(map[string]Job)
	}

	jJobMap := jsonJobMap{}
	err := json.Unmarshal(data, &jJobMap)
	if err != nil {
		return err
	}

	t.values = jJobMap.Values
	t.keys = jJobMap.Keys

	return nil
}

func (t *JobMap) UnmarshalYAML(node *yaml.Node) error {
	if t == nil {
		t = &JobMap{
			values: make(map[string]Job),
			keys:   []string{},
		}
	}

	if t.values == nil {
		t.values = make(map[string]Job)
	}

	if t.keys == nil {
		t.keys = []string{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.New("invalid yaml node for JobMap")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		key := keyNode.Value
		var job Job
		if err := valueNode.Decode(&job); err != nil {
			return err
		}

		job.Name = key
		if job.Id == "" {
			job.Id = id.Convert(job.Name)
		}

		t.Add(&job)
	}

	return nil
}

func NewJobMap() *JobMap {
	return &JobMap{
		values: make(map[string]Job),
		keys:   []string{},
	}
}

func (t *JobMap) init() {
	if t == nil {
		t = &JobMap{}
	}

	if t.values == nil {
		t.values = map[string]Job{}
	}

	if t.keys == nil {
		t.keys = []string{}
	}
}

func (t *JobMap) ToMap() map[string]Job {
	if t == nil {
		t = NewJobMap()
	}
	m := make(map[string]Job, len(t.values))
	for k, v := range t.values {
		m[k] = v
	}
	return m
}

func (t *JobMap) Len() int {
	if t == nil {
		return 0
	}

	t.init()
	return len(t.values)
}

func (t *JobMap) IsEmpty() bool {
	if t == nil {
		return true
	}
	return t.Len() == 0
}

func (t *JobMap) Keys() []string {
	if t == nil {
		t = NewJobMap()
	}
	keys := []string{}
	keys = append(keys, t.keys...)
	return keys
}

func (t *JobMap) Values() []Job {
	if t == nil {
		t = NewJobMap()
	}

	values := []Job{}
	for _, k := range t.keys {
		values = append(values, t.values[k])
	}
	return values
}

func (t *JobMap) Add(entry *Job) bool {
	if t == nil {
		t = NewJobMap()
	}

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

func (t *JobMap) Get(name string) (Job, bool) {
	if t == nil {
		t = NewJobMap()
	}

	if t == nil || t.values == nil {
		return Job{}, false
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

	return Job{}, false
}

func (t *JobMap) GetById(idValue string) (Job, bool) {
	if t == nil {
		t = NewJobMap()
	}

	if t == nil || t.values == nil {
		return Job{}, false
	}

	for _, k := range t.keys {
		if strings.EqualFold(id.Convert(k), idValue) {
			entry, ok := t.values[k]
			return entry, ok
		}
	}

	return Job{}, false
}

func (t *JobMap) Set(entry *Job) bool {
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

func (t *JobMap) TryGetSlice(key ...string) ([]Job, bool) {
	if t == nil || t.values == nil {
		return nil, false
	}

	results := make([]Job, 0, len(key))
	for _, k := range key {
		s, ok := t.Get(k)
		if !ok {
			continue
		}
		results = append(results, s)
	}
	return results, len(results) > 0
}

func (t *JobMap) FlattenJobs(targets []string, context string) ([]Job, error) {
	return FlattenJobs(targets, *t, []Job{}, context)
}

func FlattenJobs(targets []string, jobs JobMap, set []Job, context string) ([]Job, error) {

	for _, target := range targets {
		t := target

		var job Job
		found := false

		// prefer context-specific Job if context is provided and it is found.
		if context != "" {
			t = target + ":" + context
			job2, ok := jobs.Get(t)
			if ok {
				job = job2
				found = true
			}
		}

		if !found {
			t = target
			job2, ok := jobs.Get(t)
			if !ok {
				return nil, errors.New("Job not found: " + target + " or " + t)
			}

			job = job2
		}

		// ensure dependencies are added first
		if job.Needs != nil && len(*job.Needs) > 0 {
			neededJobs, err := FlattenJobs(job.Needs.Names(), jobs, set, context)
			if err != nil {
				return nil, err
			}
			set = neededJobs
		}

		added := false
		for _, Job2 := range set {
			if job.Id == Job2.Id {
				added = true
				break
			}
		}

		if !added {
			set = append(set, job)
		}
	}

	return set, nil
}

func FindJobCyclicalReferences(jobs []Job) []Job {
	stack := []Job{}
	cycles := []Job{}

	var resolve func(job Job) bool
	resolve = func(job Job) bool {
		for _, t := range stack {
			if job.Id == t.Id {
				return false
			}
		}

		stack = append(stack, job)

		if job.Needs != nil && len(*job.Needs) > 0 {
			for _, need := range *job.Needs {
				for _, nextJob := range jobs {
					if nextJob.Id == need.Id {
						if !resolve(nextJob) {
							return false
						}
					}
				}
			}
		}

		stack = stack[:len(stack)-1]
		return true
	}

	for _, job := range jobs {
		if !resolve(job) {
			cycles = append(cycles, job)
		}
	}

	return cycles
}
