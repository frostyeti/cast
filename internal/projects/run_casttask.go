package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/exec"
)

func runCastTask(ctx TaskContext, casttaskPath string) *TaskResult {
	res := NewTaskResult()
	res.Start()

	var def types.CastTask
	if err := def.ReadFromYaml(casttaskPath); err != nil {
		return res.Fail(err)
	}

	// Prepare environment for injection
	envUpdates := make(map[string]string)

	// Copy current context env
	for k, v := range ctx.Task.Env {
		envUpdates[k] = v
	}

	// Validate inputs and create INPUT_ vars
	for inputName, inputDef := range def.Inputs {
		// Try exact match first
		val, ok := ctx.Task.With[inputName]

		// Try lowercase match
		if !ok {
			for k, v := range ctx.Task.With {
				if strings.ToLower(k) == strings.ToLower(inputName) {
					val = v
					ok = true
					break
				}
			}
		}

		if !ok && inputDef.Required {
			return res.Fail(errors.Newf("remote task '%s' requires input '%s'", def.Name, inputName))
		}

		valStr := ""
		if ok {
			valStr = fmt.Sprintf("%v", val)
		} else if inputDef.Default != "" {
			valStr = inputDef.Default
		}

		if valStr != "" {
			envKey := "INPUT_" + strings.ToUpper(strings.ReplaceAll(inputName, "-", "_"))
			envUpdates[envKey] = valStr
		}
	}

	// Update the Task's environment
	if ctx.Task.Env == nil {
		ctx.Task.Env = make(map[string]string)
	}
	for k, v := range envUpdates {
		ctx.Task.Env[k] = v
	}

	// Dispatch based on Runs.Using
	switch def.Runs.Using {
	case "docker":
		return runCastTaskDocker(ctx, &def)
	case "deno":
		// Override Uses and With, run as Deno
		mainPath := def.Runs.Main
		if mainPath == "" {
			mainPath = "mod.ts" // fallback
		}

		absMainPath := filepath.Join(filepath.Dir(casttaskPath), mainPath)
		// Clear With because inputs are now in ENV
		ctx.Task.With = make(map[string]any)
		return runDenoWrapper(ctx, absMainPath)
	case "composite":
		return res.Fail(errors.New("composite remote tasks are not yet implemented"))
	default:
		return res.Fail(errors.Newf("unknown execution engine '%s' in cast.task", def.Runs.Using))
	}
}

func runCastTaskDocker(ctx TaskContext, def *types.CastTask) *TaskResult {
	res := NewTaskResult()
	res.Start()

	cwd := ctx.Task.Cwd
	image := def.Runs.Image
	if image == "" {
		return res.Fail(errors.New("docker task requires an 'image' defined in 'runs'"))
	}

	args := []string{"run", "--rm"}

	if cwd != "" {
		args = append(args, "-w", "/app")
		args = append(args, "-v", fmt.Sprintf("%s:/app", cwd))
	}

	for k, v := range ctx.Task.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, image)

	// Expand env vars in args before appending
	for _, arg := range def.Runs.Args {
		expanded := os.Expand(arg, func(key string) string {
			return ctx.Task.Env[key]
		})
		args = append(args, expanded)
	}

	trackDockerImage(image)

	cmd := exec.New("docker", args...)
	cmd.WithCwd(cwd)

	o, err := runCmdWithContext(ctx, cmd)
	if err != nil {
		return res.Fail(err)
	}

	if o.Code != 0 {
		return res.Fail(errors.Newf("Docker task failed with exit code %d", o.Code))
	}

	return res.Ok()
}
