//go:build !windows

package pwsh

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("pwsh", &exec.Executable{
		Name:     "pwsh",
		Variable: "RUN_PWSH_EXE",
		Linux: []string{
			"/bin/pwsh",
			"/usr/bin/pwsh",
		},
	})
}
