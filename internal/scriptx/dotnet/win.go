//go:build windows

package dotnet

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("dotnet", &exec.Executable{
		Name:     "dotnet",
		Variable: "RUN_WIN_DOTNET_EXE",
		Windows: []string{
			"${HOME}\\.dotnet\\dotnet.exe",
			"${LOCALAPPDATA}\\dotnet\\dotnet.exe",
			"${ProgramFiles}\\dotnet\\dotnet.exe",
			"%ProgramFiles(x86)%\\dotnet\\dotnet.exe",
		},
	})
}
