package golang

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/go/exec"
)

const NAME = "go"

var ScriptArgs = []string{"run"}

var Extensions = []string{".go"}

func New(args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "go"
	}

	return exec.New(exe, args...)
}

func NewContext(ctx context.Context, args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "go"
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
	tmpDir := os.TempDir()

	data := []byte(script)

	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash[:])
	tmpFile := filepath.Join(tmpDir, "run-"+hashStr+".go")

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		if err := os.WriteFile(tmpFile, data, 0744); err != nil {
			cmd := New(args...)
			cmd.Err = err
			return cmd
		}
	}

	cmd := File(tmpFile, args...)
	cmd.TempFile = &tmpFile
	return cmd
}

func InlineContext(ctx context.Context, script string, args ...string) *exec.Cmd {
	tmpDir := os.TempDir()

	data := []byte(script)

	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash[:])
	tmpFile := filepath.Join(tmpDir, "run-"+hashStr+".go")

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		if err := os.WriteFile(tmpFile, data, 0744); err != nil {
			cmd := NewContext(ctx, args...)
			cmd.Err = err
			return cmd
		}
	}

	cmd := FileContext(ctx, tmpFile, args...)
	cmd.TempFile = &tmpFile
	return cmd
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
