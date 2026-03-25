package types_test

import (
	"testing"

	"github.com/frostyeti/cast/internal/types"
	"go.yaml.in/yaml/v4"
)

func TestProjectConfigUnmarshal_AllowsUnknownValues(t *testing.T) {
	data := []byte(`
context: prod
substitution: true
feature_flags:
  lint: true
  retries: 3
`)

	var cfg types.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal project config failed: %v", err)
	}

	if cfg.Context == nil || *cfg.Context != "prod" {
		t.Fatalf("expected context=prod, got %+v", cfg.Context)
	}

	if cfg.Substitution == nil || !*cfg.Substitution {
		t.Fatalf("expected substitution=true, got %+v", cfg.Substitution)
	}

	if cfg.Values == nil {
		t.Fatalf("expected values map to be initialized")
	}

	v, ok := cfg.Values["feature_flags"]
	if !ok || v == nil {
		t.Fatalf("expected feature_flags in values map, got %#v", cfg.Values)
	}
}
