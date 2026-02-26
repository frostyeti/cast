package projects_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/stretchr/testify/require"
)

func TestStandaloneInventory(t *testing.T) {
	tmpDir := t.TempDir()

	projectDir := filepath.Join(tmpDir, "project")
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// Create an inventory file
	invContent := `
hosts:
  prod-db:
    host: 10.0.0.1
    user: dbadmin
    tags: [db, prod]
`
	invPath := filepath.Join(projectDir, "prod-inv.yaml")
	err = os.WriteFile(invPath, []byte(invContent), 0644)
	require.NoError(t, err)

	// Create castfile.yaml that imports this inventory
	castfileContent := `
inventories:
  - ./prod-inv.yaml
`
	castfilePath := filepath.Join(projectDir, "castfile.yaml")
	err = os.WriteFile(castfilePath, []byte(castfileContent), 0644)
	require.NoError(t, err)

	proj := &projects.Project{}
	err = proj.LoadFromYaml(castfilePath)
	require.NoError(t, err)

	err = proj.Init()
	require.NoError(t, err)

	require.Contains(t, proj.Hosts, "prod-db")
	require.Equal(t, "10.0.0.1", proj.Hosts["prod-db"].Host)
	require.Equal(t, "dbadmin", proj.Hosts["prod-db"].User)
	require.Contains(t, proj.Hosts["prod-db"].Tags, "db")
}

func TestStandaloneInventoryImplicitPath(t *testing.T) {
	tmpDir := t.TempDir()

	projectDir := filepath.Join(tmpDir, "project")
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// Create .cast/inventory
	invDir := filepath.Join(projectDir, ".cast", "inventory")
	err = os.MkdirAll(invDir, 0755)
	require.NoError(t, err)

	// Create an inventory file
	invContent := `
hosts:
  dev-api:
    host: 192.168.1.5
    user: apiuser
`
	invPath := filepath.Join(invDir, "dev.yaml") // matches "dev" implicit extension
	err = os.WriteFile(invPath, []byte(invContent), 0644)
	require.NoError(t, err)

	// Create castfile.yaml
	castfileContent := `
inventories:
  - dev
`
	castfilePath := filepath.Join(projectDir, "castfile.yaml")
	err = os.WriteFile(castfilePath, []byte(castfileContent), 0644)
	require.NoError(t, err)

	proj := &projects.Project{}
	err = proj.LoadFromYaml(castfilePath)
	require.NoError(t, err)

	err = proj.Init()
	require.NoError(t, err)

	require.Contains(t, proj.Hosts, "dev-api")
	require.Equal(t, "192.168.1.5", proj.Hosts["dev-api"].Host)
	require.Equal(t, "apiuser", proj.Hosts["dev-api"].User)
}
