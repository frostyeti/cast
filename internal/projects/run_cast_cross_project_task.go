package projects

import (
	"context"
	"os"
	"path/filepath"

	"github.com/frostyeti/cast/internal/errors"
)

func runCastCrossProjectTask(ctx TaskContext) *TaskResult {
	res := NewTaskResult()
	res.Start()

	// Find target castfile or directory
	fileValue, hasFile := ctx.Task.With["file"]
	dirValue, hasDir := ctx.Task.With["dir"]

	if !hasFile && !hasDir {
		return res.Fail(errors.New("cast handler requires 'file' or 'dir' defined in 'with'"))
	}

	targetPath := ""
	if hasFile {
		fileStr, ok := fileValue.(string)
		if !ok || fileStr == "" {
			return res.Fail(errors.New("'file' in 'with' must be a non-empty string"))
		}
		targetPath = fileStr
	} else {
		dirStr, ok := dirValue.(string)
		if !ok || dirStr == "" {
			return res.Fail(errors.New("'dir' in 'with' must be a non-empty string"))
		}
		targetPath = dirStr
	}

	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(ctx.Task.Cwd, targetPath)
	}

	// Find what to run: task or job
	taskValue, hasTask := ctx.Task.With["task"]
	jobValue, hasJob := ctx.Task.With["job"]

	if !hasTask && !hasJob {
		return res.Fail(errors.New("cast handler requires 'task' or 'job' defined in 'with'"))
	}

	if hasTask && hasJob {
		return res.Fail(errors.New("cast handler cannot accept both 'task' and 'job' simultaneously"))
	}

	targetName := ""
	isJob := false
	if hasTask {
		taskStr, ok := taskValue.(string)
		if !ok || taskStr == "" {
			return res.Fail(errors.New("'task' in 'with' must be a non-empty string"))
		}
		targetName = taskStr
	} else {
		jobStr, ok := jobValue.(string)
		if !ok || jobStr == "" {
			return res.Fail(errors.New("'job' in 'with' must be a non-empty string"))
		}
		targetName = jobStr
		isJob = true
	}

	// Initialize the target project
	targetProj := &Project{}

	// Check if targetPath is a directory or a file
	// LoadFromYaml expects a path to a file or it finds one in dir.
	stat, err := os.Stat(targetPath)
	if err != nil {
		return res.Fail(errors.Newf("failed to stat target path '%s': %v", targetPath, err))
	}

	actualPath := targetPath
	if stat.IsDir() {
		found := false
		tryFiles := []string{"castfile", "castfile.yaml", "castfile.yml", ".castfile"}
		for _, f := range tryFiles {
			p := filepath.Join(targetPath, f)
			if _, err := os.Stat(p); err == nil {
				actualPath = p
				found = true
				break
			}
		}
		if !found {
			return res.Fail(errors.Newf("no castfile found in directory '%s'", targetPath))
		}
	}

	err = targetProj.LoadFromYaml(actualPath)
	if err != nil {
		return res.Fail(errors.Newf("failed to load target castfile '%s': %v", actualPath, err))
	}

	// Inherit environment from current context
	for k, v := range ctx.Task.Env {
		targetProj.Env.Set(k, v)
	}

	if isJob {
		// Run the job
		jobParams := RunJobParams{
			JobID:         targetName,
			Context:       context.Background(),
			ContextName:   "default",
			Stdout:        ctx.Stdout,
			Stderr:        ctx.Stderr,
			RunDownstream: false,
		}
		err = targetProj.RunJob(jobParams)
		if err != nil {
			return res.Fail(errors.Newf("cross-project job '%s' failed: %v", targetName, err))
		}
	} else {
		// Run the task
		params := RunTasksParams{
			Targets:     []string{targetName},
			Context:     context.Background(),
			ContextName: "default",
			Stdout:      ctx.Stdout,
			Stderr:      ctx.Stderr,
			Env:         ctx.Task.Env,
		}
		_, err = targetProj.RunTask(params)
		if err != nil {
			return res.Fail(errors.Newf("cross-project task '%s' failed: %v", targetName, err))
		}
	}

	return res.Ok()
}
