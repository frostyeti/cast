package deno

import (
	"reflect"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	cmd := New("arg1", "arg2")
	expectedArgs := []string{"arg1", "arg2"}
	if !strings.HasSuffix(cmd.Path, "deno") || !reflect.DeepEqual(cmd.Args[1:], expectedArgs) {
		t.Errorf("expected path ending with 'deno' and args %v, got path '%s' and args %v", expectedArgs, cmd.Path, cmd.Args[1:])
	}
}

func TestFile(t *testing.T) {
	cmd := File("test.js", "arg1")
	expectedArgs := []string{"run", "-A", "test.js", "arg1"}
	if !strings.HasSuffix(cmd.Path, "deno") || !reflect.DeepEqual(cmd.Args[1:], expectedArgs) {
		t.Errorf("expected path ending with 'deno' and args %v, got path '%s' and args %v", expectedArgs, cmd.Path, cmd.Args[1:])
	}
}

func TestInline(t *testing.T) {
	cmd := Inline("console.log('hello')", "arg1")
	expectedArgs := []string{"eval", "console.log('hello')", "arg1"}
	if !strings.HasSuffix(cmd.Path, "deno") || !reflect.DeepEqual(cmd.Args[1:], expectedArgs) {
		t.Errorf("expected path ending with 'deno' and args %v, got path '%s' and args %v", expectedArgs, cmd.Path, cmd.Args[1:])
	}
}

func TestScript_File(t *testing.T) {
	cmd := Script("test.js", "arg1")
	expectedArgs := []string{"run", "-A", "test.js", "arg1"}
	if !strings.HasSuffix(cmd.Path, "deno") || !reflect.DeepEqual(cmd.Args[1:], expectedArgs) {
		t.Errorf("expected path ending with 'deno' and args %v, got path '%s' and args %v", expectedArgs, cmd.Path, cmd.Args[1:])
	}
}

func TestScript_Inline(t *testing.T) {
	cmd := Script("console.log('hello')", "arg1")
	expectedArgs := []string{"eval", "console.log('hello')", "arg1"}
	if !strings.HasSuffix(cmd.Path, "deno") || !reflect.DeepEqual(cmd.Args[1:], expectedArgs) {
		t.Errorf("expected path ending with 'deno' and args %v, got path '%s' and args %v", expectedArgs, cmd.Path, cmd.Args[1:])
	}
}