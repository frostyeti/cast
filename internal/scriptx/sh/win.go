//go:build windows

package sh

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("sh", &exec.Executable{
		Name:     "sh",
		Variable: "RUN_WIN_SH_EXE",
		Windows: []string{
			"${ProgramFiles}\\Git\\bin\\sh.exe",
			"${ProgramFiles(x86)}\\Git\\bin\\sh.exe",
		},
	})
}
