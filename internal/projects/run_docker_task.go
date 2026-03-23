package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/paths"
	"github.com/frostyeti/go/cmdargs"
	"github.com/frostyeti/go/exec"
)

func runDockerTask(ctx TaskContext) *TaskResult {
	res := NewTaskResult()

	cwd := ctx.Task.Cwd

	imageValue, ok := ctx.Task.With["image"]
	if !ok {
		return res.Fail(errors.New("docker task requires an 'image' defined in 'with'"))
	}
	image, ok := imageValue.(string)
	if !ok || image == "" {
		return res.Fail(errors.New("docker task requires 'image' to be a valid string"))
	}

	args := []string{"run", "--rm"}

	if cwd != "" {
		// Set workspace / app directory
		args = append(args, "-w", "/app")
		args = append(args, "-v", fmt.Sprintf("%s:/app", cwd))
	}

	// Mount additional volumes if requested
	volumesValue, ok := ctx.Task.With["volumes"]
	if ok {
		if volumesList, ok := volumesValue.([]any); ok {
			for _, v := range volumesList {
				if volStr, ok := v.(string); ok {
					args = append(args, "-v", volStr)
				}
			}
		}
	}

	// only set environment variables explicitly set on the task
	if ctx.Schema.Env != nil {
		for _, k := range ctx.Schema.Env.Keys() {
			v := ctx.Task.Env[k]
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Add the image
	args = append(args, image)

	// Command + Arguments
	cmdValue, ok := ctx.Task.With["command"]
	commandSet := false
	if ok {
		commandSet = true
		if cmdStr, ok := cmdValue.(string); ok && cmdStr != "" {
			args = append(args, cmdStr)
		}
	}

	argsValue, ok := ctx.Task.With["args"]
	argsSet := false
	if ok {
		argsSet = true
		if argsList, ok := argsValue.([]any); ok {
			for _, arg := range argsList {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		} else if argStr, ok := argsValue.(string); ok {
			// Try to parse string into args
			parsed := cmdargs.Split(argStr)
			args = append(args, parsed.ToArray()...)
		}
	}

	if !commandSet && !argsSet {
		// Add run property logic if command isn't present
		if ctx.Task.Run != "" && cmdValue == nil {
			runArgs := cmdargs.Split(ctx.Task.Run)
			args = append(args, runArgs.ToArray()...)
		}
	}

	if len(ctx.Task.Args) > 0 {
		args = append(args, ctx.Task.Args...)
	}

	// Log image usage
	trackDockerImage(image)

	res.Start()

	// TODO: print command, but remove any secrets
	// cliArgs := cmdargs.New(args).String()

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

func trackDockerImage(image string) {
	dataDir, err := paths.UserDataDir()
	if err != nil {
		return
	}
	castDir := filepath.Join(dataDir, "cast")
	os.MkdirAll(castDir, 0755)

	trackingFile := filepath.Join(castDir, "docker_images.txt")

	var images []string
	if content, err := os.ReadFile(trackingFile); err == nil {
		images = strings.Split(string(content), "\n")
	}

	found := false
	for _, img := range images {
		if strings.TrimSpace(img) == image {
			found = true
			break
		}
	}

	if !found {
		f, err := os.OpenFile(trackingFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(image + "\n")
			f.Close()
		}
	}
}
