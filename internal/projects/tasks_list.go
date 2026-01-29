package projects

import "github.com/frostyeti/cast/internal/types"

func (p *Project) ListTasks() (types.TaskMap, error) {
	err := p.Init()
	if err != nil {
		return types.TaskMap{}, err
	}

	return p.Tasks, nil
}
