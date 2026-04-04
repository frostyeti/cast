package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/paths"
	castdeno "github.com/frostyeti/cast/internal/scriptx/deno"
	"github.com/frostyeti/go/exec"
)

const denoLingeringResourceExitCode = 124
const defaultDenoDiagnosticDelayMs = 1500

func createDenoTaskCmd(ctx TaskContext, script string, args []string) (*exec.Cmd, func(), error) {
	wrappedPath, cleanup, err := writeDenoTaskWrapper(script, ctx.Task.Cwd, ctx.Task.Id, ctx.Task.Name)
	if err != nil {
		return nil, nil, err
	}

	cmd := castdeno.FileContext(ctx.Context, wrappedPath, args...)
	return cmd, cleanup, nil
}

func writeDenoTaskWrapper(script, cwd, taskID, taskName string) (string, func(), error) {
	if strings.TrimSpace(script) == "" {
		return "", nil, errors.New("No script provided for Deno task")
	}

	wrapperDir := os.TempDir()
	wrapperExt := ".ts"
	sourceLabel := "<inline>"
	contents := script

	if denoScriptPath(script) {
		resolvedPath := strings.TrimSpace(script)
		if !filepath.IsAbs(resolvedPath) {
			var err error
			resolvedPath, err = paths.ResolvePath(cwd, resolvedPath)
			if err != nil {
				return "", nil, errors.New("Failed to resolve Deno script path: " + err.Error())
			}
		}

		bytes, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", nil, errors.New("Failed to read Deno script file: " + err.Error())
		}

		contents = string(bytes)
		sourceLabel = resolvedPath
		wrapperDir = filepath.Dir(resolvedPath)
		wrapperExt = filepath.Ext(resolvedPath)
		if wrapperExt == "" {
			wrapperExt = ".ts"
		}
	} else if pathIsDir(cwd) {
		wrapperDir = cwd
	}

	taskSlug := taskID
	if strings.TrimSpace(taskSlug) == "" {
		taskSlug = "task"
	}
	taskSlug = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, taskSlug)

	pattern := fmt.Sprintf(".cast-deno-%s-*%s", taskSlug, wrapperExt)
	f, err := os.CreateTemp(wrapperDir, pattern)
	if err != nil {
		return "", nil, errors.New("Failed to create Deno wrapper file: " + err.Error())
	}

	wrapperPath := f.Name()
	wrapperContent := contents
	if !strings.HasSuffix(wrapperContent, "\n") {
		wrapperContent += "\n"
	}
	wrapperContent += buildDenoLingeringResourceHelper(taskName, sourceLabel)
	wrapperContent += "\n__castScheduleLingeringResourceDiagnostic();\n"

	if _, err := f.WriteString(wrapperContent); err != nil {
		_ = f.Close()
		_ = os.Remove(wrapperPath)
		return "", nil, errors.New("Failed to write Deno wrapper file: " + err.Error())
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(wrapperPath)
		return "", nil, errors.New("Failed to finalize Deno wrapper file: " + err.Error())
	}

	cleanup := func() {
		_ = os.Remove(wrapperPath)
	}

	return wrapperPath, cleanup, nil
}

func denoScriptPath(script string) bool {
	if strings.ContainsAny(script, "\n\r") {
		return false
	}

	trimmed := strings.TrimSpace(script)
	for _, ext := range castdeno.Extensions {
		if strings.HasSuffix(trimmed, ext) {
			return true
		}
	}

	return false
}

func pathIsDir(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func buildDenoLingeringResourceHelper(taskName, sourceLabel string) string {
	if strings.TrimSpace(taskName) == "" {
		taskName = "deno-task"
	}

	return fmt.Sprintf(`
const __castDenoTaskName = %s;
const __castDenoSource = %s;
const __castDenoDiagnosticDelayMs = Number(Deno.env.get("CAST_DENO_DIAGNOSTIC_DELAY_MS") ?? %q);

function __castScheduleLingeringResourceDiagnostic() {
	if (!Number.isFinite(__castDenoDiagnosticDelayMs) || __castDenoDiagnosticDelayMs < 0) {
		return;
	}

	const __castTimer = setTimeout(() => {
		const __castResources = Deno.resources();
		const __castOpenResources = Object.entries(__castResources);
		if (__castOpenResources.length === 0) {
			return;
		}

		console.error("[cast] Deno task " + __castDenoTaskName + " finished running user code but did not exit after " + __castDenoDiagnosticDelayMs + "ms.");
		console.error("[cast] Source: " + __castDenoSource);
		console.error("[cast] Active Deno resources still keeping the process alive:");
		console.error(JSON.stringify(__castResources, null, 2));
		console.error("[cast] Close the resources in your script, add explicit teardown, or call Deno.exit(0) when completion is intentional.");
		Deno.exit(%d);
	}, __castDenoDiagnosticDelayMs);

	if (typeof Deno.unrefTimer === "function") {
		Deno.unrefTimer(__castTimer);
	}
}
`, strconv.Quote(taskName), strconv.Quote(sourceLabel), strconv.Itoa(defaultDenoDiagnosticDelayMs), denoLingeringResourceExitCode)
}

func newDenoLingeringResourceError(taskID string) error {
	if strings.TrimSpace(taskID) == "" {
		return errors.New("Deno task left resources open; see diagnostics above")
	}

	return errors.Newf("Task %s left Deno resources open; see diagnostics above", taskID)
}

func buildDenoModuleWrapper(modulePath, withJSON, taskName string) string {
	return fmt.Sprintf(`
import process from "node:process";
import * as mod from %s;

globalThis["inputs"] = %s;

%s

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

main()
	.then(() => {
		__castScheduleLingeringResourceDiagnostic();
	})
	.catch(err => {
		console.error(err);
		process.exit(1);
	});
`, strconv.Quote(modulePath), withJSON, buildDenoLingeringResourceHelper(taskName, modulePath))
}
