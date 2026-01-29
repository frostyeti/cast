//go:build !windows

package sh

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("sh", &exec.Executable{
		Name:     "sh",
		Variable: "RUN_SH_EXE",
		Linux: []string{
			"/bin/sh",
			"/usr/bin/sh",
		},
	})
}
