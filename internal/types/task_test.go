package types

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestTaskNeedsAcceptsStringAndObject(t *testing.T) {
	var task Task
	require.NoError(t, yaml.Unmarshal([]byte("needs: build\n"), &task))
	require.Len(t, task.Needs, 1)
	require.Equal(t, "build", task.Needs[0].Id)

	var taskList Task
	require.NoError(t, yaml.Unmarshal([]byte("needs:\n  - build\n  - id: test\n    parallel: true\n"), &taskList))
	require.Len(t, taskList.Needs, 2)
	require.Equal(t, "build", taskList.Needs[0].Id)
	require.Equal(t, "test", taskList.Needs[1].Id)
	require.True(t, taskList.Needs[1].Parallel)
}
