//go:build !windows

package dotnet

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("dotnet", &exec.Executable{
		Name:     "dotnet",
		Variable: "RUN_DOTNET_EXE",
		Linux: []string{
			"$HOME/.dotnet/dotnet",
			"$HOME/.local/share/dotnet/dotnet",
			"/usr/local/bin/dotnet",
			"/usr/bin/dotnet",
			"/bin/dotnet",
		},
	})
}
