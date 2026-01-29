package sh

import (
	"context"
	"strings"

	"github.com/frostyeti/go/exec"
)

const NAME = "sh"

var ScriptArgs = []string{"-e"}

func New(args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "sh"
	}

	return exec.New(exe, args...)
}

func NewContext(ctx context.Context, args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "sh"
	}

	return exec.NewContext(ctx, exe, args...)
}

func File(path string, args ...string) *exec.Cmd {
	allArgs := append(append(ScriptArgs, path), args...)
	return New(allArgs...)
}

func FileContext(ctx context.Context, path string, args ...string) *exec.Cmd {
	allArgs := append(append(ScriptArgs, path), args...)
	return NewContext(ctx, allArgs...)
}

func Inline(script string, args ...string) *exec.Cmd {
	splat := append(ScriptArgs, script)
	splat = append(splat, args...)
	return New(splat...)
}

func InlineContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	splat := append(ScriptArgs, script)
	splat = append(splat, args...)
	return NewContext(ctx, splat...)
}

func Script(script string, args ...string) *exec.Cmd {
	if !strings.ContainsAny(script, "\n\r") {
		trimmed := strings.TrimSpace(script)
		if strings.HasSuffix(trimmed, ".sh") {
			return File(trimmed, args...)
		}
	}

	return Inline(script, args...)
}

func ScriptContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	if !strings.ContainsAny(script, "\n\r") {
		trimmed := strings.TrimSpace(script)
		if strings.HasSuffix(trimmed, ".sh") {
			return FileContext(ctx, trimmed, args...)
		}
	}

	return InlineContext(ctx, script, args...)
}
