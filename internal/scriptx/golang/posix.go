//go:build !windows

package golang

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("go", &exec.Executable{
		Name:     "go",
		Variable: "RUN_GO_EXE",
		Linux: []string{
			"${HOME}/.local/shared/go/bin/go",
			"/usr/local/go/bin/go",
			"/usr/local/bin/go",
			"/usr/bin/go",
		},
	})
}
