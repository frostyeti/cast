//go:build !windows

package ruby

import "github.com/frostyeti/go/exec"

func init() {
	exec.Register("ruby", &exec.Executable{
		Name:     "ruby",
		Variable: "RUN_RUBY_EXE",
		Linux: []string{
			"/usr/bin/ruby",
			"/usr/local/bin/ruby",
		},
	})
}
