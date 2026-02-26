package projects_test

import (
	"testing"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"go.yaml.in/yaml/v4"
)

func TestJobExtends(t *testing.T) {
	yamlData := `
jobs:
  base_job:
    desc: "A base job"
    cwd: "/tmp/base"
    timeout: "10s"
    steps:
      - run: "echo hello"
    env:
      FOO: "bar"
      BASE_VAR: "true"
    dotenv:
      - .env.base

  child_job:
    extends: "base_job"
    desc: "A child job"
    steps:
      - run: "echo child"
    env:
      FOO: "baz"
    dotenv:
      - .env.child
`

	var schema types.Project
	err := yaml.Unmarshal([]byte(yamlData), &schema)
	if err != nil {
		t.Fatalf("Failed to unmarshal yaml: %v", err)
	}

	p := projects.Project{
		Schema: schema,
	}

	err = p.Init()
	if err != nil {
		t.Fatalf("Failed to init project: %v", err)
	}

	childJob, ok := p.Schema.Jobs.Get("child_job")
	if !ok {
		t.Fatalf("Expected child_job to exist")
	}

	if childJob.Desc != "A child job" {
		t.Errorf("Expected child_job desc to be 'A child job', got '%s'", childJob.Desc)
	}

	if childJob.Cwd == nil || *childJob.Cwd != "/tmp/base" {
		t.Errorf("Expected child_job cwd to be '/tmp/base', got '%v'", childJob.Cwd)
	}

	if childJob.Timeout == nil || *childJob.Timeout != "10s" {
		t.Errorf("Expected child_job timeout to be '10s', got '%v'", childJob.Timeout)
	}

	if len(childJob.Steps) != 1 || childJob.Steps[0].Run != "echo child" {
		t.Errorf("Expected child_job to have its own steps")
	}

	fooVar := childJob.Env.Get("FOO")
	if fooVar != "baz" {
		t.Errorf("Expected FOO to be 'baz', got '%s'", fooVar)
	}

	baseVar := childJob.Env.Get("BASE_VAR")
	if baseVar != "true" {
		t.Errorf("Expected BASE_VAR to be 'true', got '%s'", baseVar)
	}

	if childJob.DotEnv == nil || len(*childJob.DotEnv) != 2 {
		t.Errorf("Expected child_job to have 2 dotenvs, got '%v'", childJob.DotEnv)
	} else {
		if (*childJob.DotEnv)[0].Path != ".env.base" {
			t.Errorf("Expected first dotenv to be .env.base")
		}
		if (*childJob.DotEnv)[1].Path != ".env.child" {
			t.Errorf("Expected second dotenv to be .env.child")
		}
	}
}
