package projects

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/go/exec"
)

func IsCastTaskDefinitionFile(path string) bool {
	if path == "" {
		return false
	}

	base := filepath.Base(path)
	switch base {
	case "cast", "spell", "cast.task", "cast.yaml", "cast.yml", "spell.yaml", "spell.yml":
		return true
	}

	return strings.HasSuffix(base, ".task") || strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")
}

func runRemoteTask(ctx TaskContext) *TaskResult {
	res := NewTaskResult()
	res.Start()

	uses := ctx.Task.Uses

	var modulePath string
	var err error

	if IsRemoteTask(uses) {
		trustedSources := ctx.Project.Schema.TrustedSources
		modulePath, err = FetchRemoteTask(ctx.Project, uses, trustedSources, ctx.Stdout)
		if err != nil {
			return res.Fail(err)
		}
	} else {
		// If it's a local module or file
		modulePath = uses
	}

	if IsCastTaskDefinitionFile(modulePath) {
		return runCastTask(ctx, modulePath)
	}

	// TODO: get default js runtime
	return runDenoWrapper(ctx, modulePath, "deno")
}

func runDenoWrapper(ctx TaskContext, modulePath string, jsRuntime string) *TaskResult {
	res := NewTaskResult()
	res.Start()

	cwd := ctx.Task.Cwd

	// Generate wrapper script
	tmpDir := os.TempDir()
	wrapperPath := filepath.Join(tmpDir, fmt.Sprintf("cast_wrapper_%s.ts", ctx.Task.Id))

	e := ctx.Task.Env

	// Convert `With` arguments to JSON to inject them into the script or Deno.env
	withJSON, _ := json.Marshal(ctx.Task.With)
	if string(withJSON) == "null" {
		withJSON = []byte("{}")
	}

	// We pass args as stringified JSON to process.env or just in the wrapper script.
	wrapperContent := buildDenoModuleWrapper(modulePath, string(withJSON), ctx.Task.Name)

	err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644)
	if err != nil {
		return res.Fail(errors.Newf("Failed to write deno wrapper script: %v", err))
	}
	defer func() {
		_ = os.Remove(wrapperPath)
	}()

	args := []string{"run", "-A"}
	exeName := "deno"

	if jsRuntime == "bun" {
		exeName = "bun"
		args = []string{}
	}

	// Ensure deno is available via mise/exec
	denoExe, _ := exec.Find(exeName, nil)
	if denoExe == "" {
		denoExe = exeName
	}

	args = append(args, wrapperPath)
	cmd := exec.New(denoExe, args...)
	cmd.WithCwd(cwd)
	cmd.WithEnvMap(e)

	o, err := runCmdWithContext(ctx, cmd)
	if err != nil {
		if o != nil && o.Code == denoLingeringResourceExitCode {
			return res.Fail(newDenoLingeringResourceError(ctx.Task.Id))
		}
		return res.Fail(err)
	}

	if o.Code != 0 {
		if o.Code == denoLingeringResourceExitCode {
			return res.Fail(newDenoLingeringResourceError(ctx.Task.Id))
		}
		return res.Fail(errors.Newf("Deno task failed with exit code %d", o.Code))
	}

	return res.Ok()
}
