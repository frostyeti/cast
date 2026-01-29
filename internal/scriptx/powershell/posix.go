//go:build !windows

package powershell

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("powershell", &exec.Executable{
		Name:     "powershell",
		Variable: "POWERSHELL_EXE",
		Linux: []string{
			"/bin/pwsh",
			"/usr/bin/pwsh",
		},
	})
}
