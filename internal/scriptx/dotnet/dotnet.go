package dotnet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/go/exec"
)

const NAME = "dotnet"

var extensions = []string{".cs", ".csx", ".dll", ".exe", ".csproj", ".fsproj", ".vbproj"}

var ScriptArgs = []string{}

func New(args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "dotnet"
	}

	return exec.New(exe, args...)
}

func NewContext(ctx context.Context, args ...string) *exec.Cmd {
	exe, _ := exec.Find(NAME, nil)
	if exe == "" {
		exe = "dotnet"
	}

	return exec.NewContext(ctx, exe, args...)
}

func Script(path string, args ...string) *exec.Cmd {

	if !strings.ContainsAny(path, "\r\n") {
		var trimmed = strings.TrimSpace(path)
		for _, ext := range extensions {
			if strings.HasSuffix(trimmed, ext) {
				return File(trimmed, args...)
			}
		}
	}

	return Inline(path, args...)
}

func ScriptContext(ctx context.Context, path string, args ...string) *exec.Cmd {

	if !strings.ContainsAny(path, "\r\n") {
		var trimmed = strings.TrimSpace(path)
		for _, ext := range extensions {
			if strings.HasSuffix(trimmed, ext) {
				return FileContext(ctx, trimmed, args...)
			}
		}
	}

	return InlineContext(ctx, path, args...)
}

func File(path string, args ...string) *exec.Cmd {

	ext := filepath.Ext(path)

	splat := []string{"run"}
	if len(ScriptArgs) > 0 {
		splat = append(splat, ScriptArgs...)
	}

	switch ext {
	case ".cs", ".csx", "":
		splat := append(splat, "--file", path)
		if len(args) > 0 {
			splat = append(splat, "--")
			splat = append(splat, args...)
		}
		return New(splat...)
	case ".dll", ".exe":
		splat := append([]string{path}, args...)
		return New(splat...)
	default:
		splat = append(splat, "--project", path)
		if len(args) > 0 {
			splat = append(splat, "--")
			splat = append(splat, args...)
		}
		return New(splat...)
	}
}

func FileContext(ctx context.Context, path string, args ...string) *exec.Cmd {
	ext := filepath.Ext(path)

	splat := []string{"run"}
	if len(ScriptArgs) > 0 {
		splat = append(splat, ScriptArgs...)
	}

	switch ext {
	case ".cs", ".csx", "":
		splat := append(splat, "--file", path)
		if len(args) > 0 {
			splat = append(splat, "--")
			splat = append(splat, args...)
		}
		return NewContext(ctx, splat...)
	case ".dll", ".exe":
		splat := append([]string{path}, args...)
		return NewContext(ctx, splat...)
	default:
		splat = append(splat, "--project", path)
		if len(args) > 0 {
			splat = append(splat, "--")
			splat = append(splat, args...)
		}
		return NewContext(ctx, splat...)
	}
}

func Inline(script string, args ...string) *exec.Cmd {
	tmpDir := os.TempDir()
	data := []byte(script)

	hash := sha256.Sum256(data)
	hexHash := hex.EncodeToString(hash[:])
	tmpFile := filepath.Join(tmpDir, "run-"+hexHash+".cs")

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		err := os.WriteFile(tmpFile, data, 0744)
		if err != nil {
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
	hexHash := hex.EncodeToString(hash[:])
	tmpFile := filepath.Join(tmpDir, "run-"+hexHash+".cs")

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		err := os.WriteFile(tmpFile, data, 0744)
		if err != nil {
			cmd := NewContext(ctx, args...)
			cmd.Err = err
			return cmd
		}
	}

	cmd := FileContext(ctx, tmpFile, args...)
	cmd.TempFile = &tmpFile
	return cmd
}
