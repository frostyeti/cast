//go:build windows

package bash

import (
	"path/filepath"
	"strings"

	"github.com/frostyeti/go/exec"
)

func init() {
	exec.Register("bash", &exec.Executable{
		Name:     "bash",
		Variable: "RUN_WIN_BASH_EXE",
		Windows: []string{
			"${ProgramFiles}\\Git\\bin\\bash.exe",
			"%ProgramFiles(x86)%\\Git\\bin\\bash.exe",
			"${SystemRoot}\\System32\\bash.exe",
		},
	})
}

func resolveScriptFile(script string) string {
	if !filepath.IsAbs(script) {
		file, err := filepath.Abs(script)
		if err != nil {
			script = file
		}
	}

	// determine if bash is the WSL one.
	bash, _ := exec.Find("bash", nil)
	if !strings.Contains(strings.ToLower(bash), "system32") {
		return script
	}

	script = strings.ReplaceAll(script, "\\", "/")
	if script[1] == ':' {
		script = "/mnt/" + strings.ToLower(script[0:1]) + script[2:]
	}

	return script
}
