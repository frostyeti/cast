package projects

import "time"

type Task struct {
	Id       string
	Name     string
	Desc     string
	Uses     string
	Run      string
	Env      map[string]string
	With     map[string]any
	If       bool
	Cwd      string
	Timeout  time.Duration
	Force    bool
	Hosts    []HostInfo
	Args     []string
	Template string
}
