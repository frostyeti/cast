//go:build windows

package bun

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("bun", &exec.Executable{
		Name:     "bun",
		Variable: "RUN_WIN_BUN_EXE",
		Windows: []string{
			"${USERPROFILE}\\.bun\\bin\\bun.exe",
			"${LOCALAPPDATA}\\Programs\\bin\\bun.exe",
			"${LOCALAPPDATA}\\Microsoft\\WinGet\\Links\\bin.exe",
			"${ProgramFiles}\\bun\\bin\\bun.exe",
			"${ProgramFiles(x86)}\\bun\\bin\\bun.exe",
		},
	})
}
