package projects

import (
	"testing"

	"github.com/frostyeti/cast/internal/types"
)

func TestRunSSHTarget_SendEnvDefaultsFalse(t *testing.T) {
	taskSchema := &types.Task{With: types.NewWith()}
	if v, ok := taskSchema.With.GetBool("send-env"); ok || v {
		t.Fatalf("expected send-env to default false when omitted")
	}
}

func TestRunSSHTarget_SendEnvParsesBool(t *testing.T) {
	taskSchema := &types.Task{With: types.NewWith()}
	taskSchema.With.Set("send-env", true)
	v, ok := taskSchema.With.GetBool("send-env")
	if !ok || !v {
		t.Fatalf("expected send-env bool to parse from with block")
	}
}
