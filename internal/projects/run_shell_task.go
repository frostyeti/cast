package projects

import (
	"bufio"
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/frostyeti/cast/internal/paths"
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

	cwd := ctx.Task.Cwd
	scriptValue, ok := ctx.Task.With["script"]
	script, isString := scriptValue.(string)
	if ok && isString && script != "" {
		if !filepath.IsAbs(script) {
			script1, err := paths.ResolvePath(cwd, script)
			if err != nil {
				return res.Fail(errors.New("Failed to resolve script path for shell task: " + err.Error()))
			}
			script = script1
		}

		if _, err := os.Stat(script); os.IsNotExist(err) {
			return res.Fail(errors.New("Script file not found for shell task: " + script))
		}

		bytes, err := os.ReadFile(script)
		if err != nil {
			return res.Fail(errors.New("Failed to read script file for shell task: " + err.Error()))
		}
		ctx.Task.Run = string(bytes)
	}

	var cmd *exec.Cmd

	run := ctx.Task.Run

	if run == "" {
		return res.Fail(errors.New("No script provided for shell task"))
	}

	if ctx.Task.Template == "gotmpl" {
		tmp, err := template.New(ctx.Task.Id).Funcs(sprig.FuncMap()).Parse(run)
		if err != nil {
			return res.Fail(errors.New("failed to parse template file: " + err.Error()))
		}

		data := map[string]any{
			"env":  ctx.Task.Env,
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
		}

		var buf bytes.Buffer
		err = tmp.Execute(&buf, data)
		if err != nil {
			return res.Fail(errors.New("failed to parse template file: " + err.Error()))
		}

		run = buf.String()
	}

	splat := ctx.Task.Args

	switch ctx.Task.Uses {
	case "runshell":
		fallthrough
	case "shell":
		if canAppendShellArgs(run, ctx.Task.Cwd) {
			mergedArgs := append(cmdargs.Split(run).ToArray(), splat...)
			if len(mergedArgs) == 0 {
				return res.Fail(errors.New("No script provided for shell task"))
			}

			exe := mergedArgs[0]
			exeArgs := []string{}
			if len(mergedArgs) > 1 {
				exeArgs = mergedArgs[1:]
			}

			cmd = exec.New(exe, exeArgs...)
			break
		}

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
	o, err := runCmdWithContext(ctx, cmd)
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

func canAppendShellArgs(run, cwd string) bool {
	trimmed := strings.TrimSpace(run)
	if trimmed == "" {
		return false
	}

	if strings.ContainsAny(trimmed, "\n\r") {
		return false
	}

	if strings.ContainsAny(trimmed, "|;&") {
		return false
	}

	if hasNamedShellVars(trimmed) {
		return false
	}

	parts := cmdargs.Split(trimmed).ToArray()
	if len(parts) == 0 {
		return false
	}

	if len(parts) == 1 {
		if strings.HasPrefix(parts[0], "./") || strings.HasPrefix(parts[0], "../") || strings.HasPrefix(parts[0], "/") {
			candidate := parts[0]
			if !filepath.IsAbs(candidate) && cwd != "" {
				if resolved, err := paths.ResolvePath(cwd, candidate); err == nil {
					candidate = resolved
				}
			}
			if paths.IsFile(candidate) {
				return true
			}
		}

		if strings.EqualFold(parts[0], "sh") || strings.EqualFold(parts[0], "bash") || strings.EqualFold(parts[0], "pwsh") || strings.EqualFold(parts[0], "powershell") {
			return false
		}

		lower := strings.ToLower(parts[0])
		for _, ext := range []string{".sh", ".bash", ".zsh", ".ps1", ".py", ".rb", ".js", ".ts", ".mjs", ".cjs", ".go", ".cs"} {
			if strings.HasSuffix(lower, ext) {
				return true
			}
		}
	}

	return true
}

func hasNamedShellVars(run string) bool {
	runes := []rune(run)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '$' {
			continue
		}

		if i > 0 && runes[i-1] == '\\' {
			continue
		}

		if i+1 >= len(runes) {
			return true
		}

		next := runes[i+1]
		if next == '{' || next == '_' || (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
			return true
		}
	}

	return false
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

			o, err := runCmdWithContext(ctx, cmd)
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

				var pipe *exec.Pipeline
				l := len(ops)
				nextOp := &commandOperation{}
				j := i + 1
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
				o, err := pipe.Output()
				if len(o.Stdout) > 0 {
					_, _ = ctx.Stdout.Write(o.Stdout)
				}
				if len(o.Stderr) > 0 {
					_, _ = ctx.Stderr.Write(o.Stderr)
				}
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

			o, err := runCmdWithContext(ctx, cmd3)
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
