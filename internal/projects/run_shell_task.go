package projects

import (
	"bufio"
	"strconv"
	"strings"

	"github.com/frostyeti/cast/internal/scriptx/bash"
	"github.com/frostyeti/cast/internal/scriptx/bun"
	"github.com/frostyeti/cast/internal/scriptx/deno"
	"github.com/frostyeti/cast/internal/scriptx/dotnet"
	"github.com/frostyeti/cast/internal/scriptx/golang"
	"github.com/frostyeti/cast/internal/scriptx/node"
	"github.com/frostyeti/go/cmdargs"
	"github.com/frostyeti/go/env"
	"github.com/frostyeti/go/exec"

	//"github.com/frostyeti/cast/internal/scriptx/nushell"
	"github.com/frostyeti/cast/internal/scriptx/powershell"
	"github.com/frostyeti/cast/internal/scriptx/pwsh"
	"github.com/frostyeti/cast/internal/scriptx/python"
	"github.com/frostyeti/cast/internal/scriptx/ruby"
	"github.com/frostyeti/cast/internal/scriptx/sh"

	"github.com/frostyeti/cast/internal/errors"
)

func runShell(ctx TaskContext) *TaskResult {
	res := NewTaskResult()
	if ctx.Task.Uses == "" {
		shell, ok := ctx.Task.Env["RUN_DEFAULT_SHELL"]
		if ok && shell != "" {
			ctx.Task.Uses = shell
		} else {
			shell := "shell"
			ctx.Task.Uses = shell
		}
	}

	var cmd *exec.Cmd

	run := ctx.Task.Run
	splat := ctx.Task.Args

	switch ctx.Task.Uses {
	case "runshell":
		fallthrough
	case "shell":
		return runXPlatShell(run, ctx)
	case "bash":
		cmd = bash.ScriptContext(ctx.Context, run, splat...)

	case "pwsh":
		cmd = pwsh.ScriptContext(ctx.Context, run, splat...)
	case "powershell":
		cmd = powershell.ScriptContext(ctx.Context, run, splat...)

	case "sh":
		cmd = sh.ScriptContext(ctx.Context, run, splat...)

	case "go":
		fallthrough
	case "golang":
		cmd = golang.ScriptContext(ctx.Context, run, splat...)

	case "dotnet":
		fallthrough
	case "csharp":
		cmd = dotnet.ScriptContext(ctx.Context, run, splat...)

	case "deno":
		cmd = deno.ScriptContext(ctx.Context, run, splat...)

	case "node":
		cmd = node.ScriptContext(ctx.Context, run, splat...)

	case "bun":
		cmd = bun.ScriptContext(ctx.Context, run, splat...)

	case "python":
		cmd = python.ScriptContext(ctx.Context, run, splat...)

	case "ruby":
		cmd = ruby.ScriptContext(ctx.Context, run, splat...)

	default:
		err := errors.New("Unsupported shell: " + ctx.Task.Uses)
		return res.Fail(err)
	}

	if ctx.Task.Cwd != "" {
		cmd.Dir = ctx.Task.Cwd
	}

	if len(ctx.Task.Env) > 0 {
		cmd.WithEnvMap(ctx.Task.Env)
	}

	res.Start()
	o, err := cmd.Run()
	if err != nil {
		return res.Fail(err)
	}

	if o.Code != 0 {
		err := errors.New("Task " + ctx.Task.Id + " failed with exit code " + strconv.Itoa(o.Code))
		return res.Fail(err)
	}

	// Placeholder for running a shell command
	// This would typically involve executing the command in the shell
	return res.Ok()
}

func runXPlatShell(script string, ctx TaskContext) *TaskResult {

	keys := []string{}
	for k := range ctx.Task.Env {
		keys = append(keys, k)
	}

	opts := &env.ExpandOptions{
		Get: func(key string) string {
			s, ok := ctx.Task.Env[key]
			if ok {
				return s
			}

			return ""
		},
		Set: func(key, value string) error {
			ctx.Task.Env[key] = value
			return nil
		},
		Keys:                keys,
		ExpandUnixArgs:      true,
		ExpandWindowsVars:   false,
		CommandSubstitution: true,
	}

	script, err := env.ExpandWithOptions(script, opts)
	if err != nil {
		res := NewTaskResult()
		return res.Fail(err)
	}

	commands := []string{}
	sb := strings.Builder{}

	res := NewTaskResult()
	res.Start()

	scanner := bufio.NewScanner(strings.NewReader(script))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasSuffix(trimmed, "\\") || strings.HasSuffix(trimmed, "`") {
			sb.WriteString(trimmed)
			continue
		}

		sb.WriteString(trimmed)
		commands = append(commands, sb.String())
		sb.Reset()
	}

	if sb.Len() > 0 {
		commands = append(commands, sb.String())
	}

	for _, command := range commands {
		args := cmdargs.Split(command)
		hasOps := false
		for _, arg := range args.ToArray() {

			if arg == "&&" || arg == "||" || arg == "|" || arg == ";" {
				hasOps = true
				break
			}
		}

		if !hasOps && args.Len() > 0 {
			exe := args.Shift()
			cmd := exec.New(exe, args.ToArray()...)
			cmd.WithEnvMap(ctx.Task.Env)
			cmd.WithCwd(ctx.Task.Cwd)

			o, err := cmd.Run()
			if err != nil {
				return res.Fail(err)
			}

			if o.Code != 0 {
				err := errors.New("Task " + ctx.Task.Id + " failed with exit code " + strconv.Itoa(o.Code))
				return res.Fail(err)
			}

			continue
		}

		ops := []*commandOperation{}
		currentOp := &commandOperation{}
		for _, part := range args.ToArray() {
			if part == "&&" || part == "||" || part == "|" || part == ";" {
				currentOp.Operation = part
				next := currentOp
				ops = append(ops, next)

				currentOp = &commandOperation{}
				continue
			}

			if part == "" {
				continue
			}

			currentOp.Command.Append(part)
		}

		if currentOp.Command.Len() > 0 {
			ops = append(ops, currentOp)
		}

		lastOperation := ""
		for i := 0; i < len(ops); i++ {
			op := *ops[i]
			if op.IsPipe() {
				exe := op.Command.Shift()
				cmd0 := exec.New(exe, op.Command.ToArray()...)
				cmd0.WithEnvMap(ctx.Task.Env)
				cmd0.WithCwd(ctx.Task.Cwd)

				j := 1
				var pipe *exec.Pipeline
				l := len(ops)
				nextOp := &commandOperation{}
				j = i + 1
				for j < l {

					nextOp := ops[j]
					lastOperation = nextOp.Operation

					if pipe == nil {
						exe := nextOp.Command.Shift()
						nextCmd := exec.New(exe, nextOp.Command.ToArray()...)
						nextCmd.WithEnvMap(ctx.Task.Env)
						nextCmd.WithCwd(ctx.Task.Cwd)
						pipe = cmd0.Pipe(nextCmd)
					} else {
						exe := nextOp.Command.Shift()
						nextCmd := exec.New(exe, nextOp.Command.ToArray()...)
						nextCmd.WithEnvMap(ctx.Task.Env)
						nextCmd.WithCwd(ctx.Task.Cwd)
						pipe = pipe.Pipe(nextCmd)
					}

					if !nextOp.IsPipe() {
						break
					}

					j++
					if j >= l {
						break
					}
				}

				nextOp = ops[j]
				i = j
				o, err := pipe.Run()
				if o.Code == 0 {
					if nextOp.IsOr() {
						return res.Ok()
					}

					continue
				}

				if nextOp.IsOr() {
					continue
				}

				if err != nil {
					return res.Fail(err)
				}

				err = errors.New("Task " + ctx.Task.Id + " failed with exit code " + strconv.Itoa(o.Code))
				return res.Fail(err)
			}

			exe3 := op.Command.Shift()
			cmd3 := exec.New(exe3, op.Command.ToArray()...)
			cmd3.WithEnvMap(ctx.Task.Env)
			cmd3.WithCwd(ctx.Task.Cwd)

			o, err := cmd3.Run()
			if o.Code == 0 {
				if lastOperation == "||" || op.IsOr() {
					return res.Ok()
				}

				lastOperation = op.Operation
				continue
			}

			if lastOperation == "||" || op.IsOr() {
				lastOperation = op.Operation
				continue
			}

			if err != nil {
				return res.Fail(err)
			}

			err = errors.New("Task " + ctx.Task.Id + " failed with exit code " + strconv.Itoa(o.Code))
			return res.Fail(err)
		}
	}

	return res.Ok()
}

type commandOperation struct {
	Command   cmdargs.Args
	Operation string // "pipe", "and", "or", "sequence"
}

func (s *commandOperation) IsPipe() bool {
	return s.Operation == "|"
}

func (s *commandOperation) IsAnd() bool {
	return s.Operation == "&&"
}

func (s *commandOperation) IsOr() bool {
	return s.Operation == "||"
}

func (s *commandOperation) IsSequence() bool {
	return s.Operation == ";"
}

func (s *commandOperation) IsNoop() bool {
	return s.Operation == ""
}
