package projects

import (
	"time"

	"github.com/frostyeti/go/exec"
)

// runCmdWithContext executes the command redirecting output to the TaskContext writers.
// This allows capturing output for web mode without globally changing os.Stdout.
func runCmdWithContext(ctx TaskContext, cmd *exec.Cmd) (*exec.Result, error) {
	cmd.WithStdout(ctx.Stdout)
	cmd.WithStderr(ctx.Stderr)

	var out exec.Result
	out.FileName = cmd.Cmd.Path
	out.Args = cmd.Cmd.Args
	out.StartedAt = time.Now().UTC()

	err := cmd.Start()
	if err != nil {
		out.EndedAt = time.Now().UTC()
		out.Code = 1
		return &out, err
	}

	err = cmd.Wait()
	out.EndedAt = time.Now().UTC()
	out.Code = cmd.Cmd.ProcessState.ExitCode()

	if err != nil {
		return &out, err
	}

	return &out, nil
}
