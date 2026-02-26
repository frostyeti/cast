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

func runRemoteTask(ctx TaskContext) *TaskResult {
	res := NewTaskResult()
	res.Start()

	uses := ctx.Task.Uses

	var modulePath string
	var err error

	if IsRemoteTask(uses) {
		trustedSources := ctx.Project.Schema.TrustedSources
		modulePath, err = FetchRemoteTask(ctx.Project, uses, trustedSources)
		if err != nil {
			return res.Fail(err)
		}
	} else {
		// If it's a local module or file
		modulePath = uses
	}

	if strings.HasSuffix(modulePath, "casttask.yaml") || strings.HasSuffix(modulePath, "casttask.yml") {
		return runCastTask(ctx, modulePath)
	}

	return runDenoWrapper(ctx, modulePath)
}

func runDenoWrapper(ctx TaskContext, modulePath string) *TaskResult {
	res := NewTaskResult()
	res.Start()

	cwd := ctx.Task.Cwd

	// Generate wrapper script
	tmpDir := os.TempDir()
	wrapperPath := filepath.Join(tmpDir, fmt.Sprintf("cast_deno_wrapper_%s.ts", ctx.Task.Id))

	// Convert `With` arguments to JSON to inject them into the script or Deno.env
	withJSON, _ := json.Marshal(ctx.Task.With)
	if string(withJSON) == "null" {
		withJSON = []byte("{}")
	}

	// We pass args as stringified JSON to process.env or just in the wrapper script.
	wrapperContent := fmt.Sprintf(`
import * as mod from "%s";

const withArgs = %s;

// Inject into Deno.env
for (const [key, value] of Object.entries(withArgs)) {
	if (value !== null && value !== undefined) {
		Deno.env.set(key, typeof value === 'string' ? value : JSON.stringify(value));
	}
}

async function main() {
	try {
		if (typeof mod.setup === 'function') {
			await mod.setup();
		}
		if (typeof mod.run === 'function') {
			await mod.run();
		} else if (typeof mod.default === 'function') {
			await mod.default();
		}
	} finally {
		if (typeof mod.teardown === 'function') {
			await mod.teardown();
		}
	}
}

main().catch(err => {
	console.error(err);
	Deno.exit(1);
});
`, modulePath, string(withJSON))

	err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644)
	if err != nil {
		return res.Fail(errors.Newf("Failed to write deno wrapper script: %v", err))
	}
	defer os.Remove(wrapperPath)

	// Ensure deno is available via mise/exec
	denoExe, _ := exec.Find("deno", nil)
	if denoExe == "" {
		denoExe = "deno"
	}

	args := []string{"run", "-A"}

	// Add environment variables
	for k, v := range ctx.Task.Env {
		os.Setenv(k, v) // we can set it in current process or cmd.Env
	}

	args = append(args, wrapperPath)
	cmd := exec.New(denoExe, args...)
	cmd.WithCwd(cwd)

	// Copy Env variables to cmd.Env
	mergedEnv := os.Environ()
	for k, v := range ctx.Task.Env {
		mergedEnv = append(mergedEnv, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.WithEnv(mergedEnv...)

	o, err := runCmdWithContext(ctx, cmd)
	if err != nil {
		return res.Fail(err)
	}

	if o.Code != 0 {
		return res.Fail(errors.Newf("Deno task failed with exit code %d", o.Code))
	}

	return res.Ok()
}
