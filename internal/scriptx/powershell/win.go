//go:build windows

package powershell

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("powershell", &exec.Executable{
		Name:     "powershell",
		Variable: "POWERSHELL_EXE",
		Windows: []string{
			"${SystemRoot}\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
		},
	})
}
