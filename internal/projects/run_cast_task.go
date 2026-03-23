package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/exec"
)

func cloneTaskEnv(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}

	clone := make(map[string]string, len(src))
	for k, v := range src {
		clone[k] = v
	}
	return clone
}

func defaultCompositeShell(ctx TaskContext) string {
	if ctx.Project != nil && ctx.Project.Schema.Defaults != nil && ctx.Project.Schema.Defaults.Shell != nil {
		shell := strings.TrimSpace(*ctx.Project.Schema.Defaults.Shell)
		if shell != "" {
			return shell
		}
	}

	return "bash"
}

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
				if strings.EqualFold(k, inputName) {
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
	switch strings.TrimSpace(def.Runs.Using) {
	case "docker":

		return runCastTaskDocker(ctx, &def)

	case "deno", "bun":
		// Override Uses and With, run as Deno
		mainPath := def.Runs.Main
		if mainPath == "" {
			mainPath = "mod.ts" // fallback
		}

		absMainPath := filepath.Join(filepath.Dir(casttaskPath), mainPath)
		// Clear With because inputs are now in ENV
		ctx.Task.With = make(map[string]any)
		return runDenoWrapper(ctx, absMainPath, def.Runs.Using)
	case "composite":
		return runCastTaskComposite(ctx, &def, casttaskPath)
	default:
		return res.Fail(errors.Newf("unknown execution engine '%s' in cast.task", def.Runs.Using))
	}
}

func runCastTaskComposite(ctx TaskContext, def *types.CastTask, casttaskPath string) *TaskResult {
	res := NewTaskResult()
	res.Start()

	if def == nil {
		return res.Fail(errors.New("composite remote task definition is nil"))
	}

	if len(def.Runs.Steps) == 0 {
		return res.Fail(errors.New("composite remote task requires steps"))
	}

	baseDir := filepath.Dir(casttaskPath)
	defaultShell := defaultCompositeShell(ctx)

	for i, step := range def.Runs.Steps {
		uses := ""
		if step.Uses != nil {
			uses = strings.TrimSpace(*step.Uses)
		}

		run := ""
		if step.Run != nil {
			run = *step.Run
		}

		if uses == "" {
			if run != "" {
				uses = defaultShell
			} else {
				return res.Fail(errors.Newf("composite step %d requires uses or run", i+1))
			}
		}

		stepEnv := cloneTaskEnv(ctx.Task.Env)
		if step.Env != nil {
			for k, v := range step.Env.ToMap() {
				stepEnv[k] = v
			}
		}

		stepWith := map[string]any{}
		if step.With != nil {
			stepWith = step.With.ToMap()
		}

		stepCwd := baseDir
		if ctx.Task != nil && strings.TrimSpace(ctx.Task.Cwd) != "" {
			stepCwd = ctx.Task.Cwd
		}
		if step.Cwd != nil && strings.TrimSpace(*step.Cwd) != "" {
			stepCwd = *step.Cwd
			if !filepath.IsAbs(stepCwd) {
				stepCwd = filepath.Join(baseDir, stepCwd)
			}
		}

		stepName := step.Name
		if stepName == "" {
			stepName = fmt.Sprintf("%s-step-%d", def.Name, i+1)
		}

		stepID := step.Id
		if stepID == "" {
			stepID = fmt.Sprintf("%s-step-%d", ctx.Task.Id, i+1)
		}

		stepTemplate := ctx.Task.Template
		if step.Template != nil && strings.TrimSpace(*step.Template) != "" {
			stepTemplate = strings.TrimSpace(*step.Template)
		}

		stepArgs := append([]string(nil), ctx.Args...)
		if len(step.Args) > 0 {
			stepArgs = append([]string(nil), step.Args...)
		}

		stepHosts := append([]HostInfo(nil), ctx.Task.Hosts...)
		if len(step.Hosts) > 0 {
			resolvedHosts := resolveCompositeStepHosts(ctx.Project, step.Hosts)
			if len(resolvedHosts) == 0 {
				return res.Fail(errors.Newf("composite step %d references no known hosts", i+1))
			}
			stepHosts = resolvedHosts
		}

		stepTask := &Task{
			Id:       stepID,
			Name:     stepName,
			Uses:     uses,
			Run:      run,
			Env:      stepEnv,
			With:     stepWith,
			Cwd:      stepCwd,
			Args:     stepArgs,
			Template: stepTemplate,
			Hosts:    stepHosts,
		}

		stepCtx := ctx
		stepCtx.Schema = &step
		stepCtx.Task = stepTask

		handler, ok := GetTaskHandler(uses)
		if !ok {
			if IsRemoteTask(uses) {
				handler = runRemoteTask
			} else {
				return res.Fail(errors.Newf("composite step %d uses unsupported handler %q", i+1, uses))
			}
		}

		result := handler(stepCtx)
		if result == nil {
			return res.Fail(errors.Newf("composite step %d returned no result", i+1))
		}
		result.Task = stepTask

		if result.Status == runstatus.Skipped {
			continue
		}

		if result.Err != nil || result.Status == runstatus.Error || result.Status == runstatus.Cancelled {
			return res.Fail(errors.Newf("composite step %d (%s) failed: %w", i+1, stepName, result.Err))
		}
	}

	return res.Ok()
}

func resolveCompositeStepHosts(project *Project, refs []string) []HostInfo {
	if project == nil || len(refs) == 0 {
		return nil
	}

	hosts := []HostInfo{}
	seen := map[string]bool{}

	for _, hostId := range refs {
		if host, ok := project.Hosts[hostId]; ok {
			if !seen[host.Host] {
				hosts = append(hosts, host)
				seen[host.Host] = true
			}
			continue
		}

		for _, h := range project.Hosts {
			matched := false
			for _, tag := range h.Tags {
				if tag == hostId {
					matched = true
					break
				}
			}
			if matched && !seen[h.Host] {
				hosts = append(hosts, h)
				seen[h.Host] = true
			}
		}
	}

	return hosts
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

	// only allow environment variable explicitly set for the task
	if ctx.Schema.Env != nil {
		for _, k := range ctx.Schema.Env.Keys() {
			v := ctx.Task.Env[k]
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
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
