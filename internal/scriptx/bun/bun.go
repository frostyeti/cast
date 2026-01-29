package bun

import (
	"context"
	"strings"

	"github.com/frostyeti/go/exec"
)

const NAME = "bun"

var ScriptArgs = []string{}

var Extensions = []string{".js", ".mjs", ".cjs", ".ts"}

func New(args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "bun"
	}

	return exec.New(exe, args...)
}

func NewContext(ctx context.Context, args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "bun"
	}

	return exec.NewContext(ctx, exe, args...)
}

func File(path string, args ...string) *exec.Cmd {
	splat := append(ScriptArgs, path)
	allArgs := append(splat, args...)
	return New(allArgs...)
}

func FileContext(ctx context.Context, path string, args ...string) *exec.Cmd {
	splat := append(ScriptArgs, path)
	allArgs := append(splat, args...)
	return NewContext(ctx, allArgs...)
}

func Inline(script string, args ...string) *exec.Cmd {
	splat := append(ScriptArgs, "-e", script)
	allArgs := append(splat, args...)
	return New(allArgs...)
}

func InlineContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	splat := append(ScriptArgs, "-e", script)
	allArgs := append(splat, args...)
	return NewContext(ctx, allArgs...)
}

func Script(script string, args ...string) *exec.Cmd {
	if !strings.ContainsAny(script, "\n\r") {
		trimmed := strings.TrimSpace(script)
		for _, ext := range Extensions {
			if strings.HasSuffix(trimmed, ext) {
				return File(trimmed, args...)
			}
		}
	}

	return Inline(script, args...)
}

func ScriptContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	if !strings.ContainsAny(script, "\n\r") {
		trimmed := strings.TrimSpace(script)
		for _, ext := range Extensions {
			if strings.HasSuffix(trimmed, ext) {
				return FileContext(ctx, trimmed, args...)
			}
		}
	}

	return InlineContext(ctx, script, args...)
}
