//go:build !windows

package python

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("python", &exec.Executable{
		Name:     "python",
		Variable: "RUN_PYTHON_EXE",
		Linux: []string{
			"/usr/bin/python",
			"/usr/bin/python3",
			"/usr/local/bin/python",
			"/usr/local/bin/python3",
		},
	})
}
