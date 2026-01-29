package ruby

import (
	"context"
	"strings"

	"github.com/frostyeti/go/exec"
)

const NAME = "ruby"

var ScriptArgs = []string{"-e"}

func New(args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "ruby"
	}

	return exec.New(exe, args...)
}

func NewContext(ctx context.Context, args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "ruby"
	}

	return exec.NewContext(ctx, exe, args...)
}

func File(file string, args ...string) *exec.Cmd {
	allArgs := append([]string{file}, args...)
	return New(allArgs...)
}

func FileContext(ctx context.Context, file string, args ...string) *exec.Cmd {
	allArgs := append([]string{file}, args...)
	return NewContext(ctx, allArgs...)
}

func Inline(script string, args ...string) *exec.Cmd {
	allArgs := append(append(ScriptArgs, script), args...)
	return New(allArgs...)
}

func InlineContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	allArgs := append(append(ScriptArgs, script), args...)
	return NewContext(ctx, allArgs...)
}

func Script(script string, args ...string) *exec.Cmd {
	if !strings.ContainsAny(script, "\n\r") {
		trimmed := strings.TrimSpace(script)
		if strings.HasSuffix(trimmed, ".rb") {
			return File(trimmed, args...)
		}
	}

	return Inline(script, args...)
}

func ScriptContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	if !strings.ContainsAny(script, "\n\r") {
		trimmed := strings.TrimSpace(script)
		if strings.HasSuffix(trimmed, ".rb") {
			return FileContext(ctx, trimmed, args...)
		}
	}

	return InlineContext(ctx, script, args...)
}
