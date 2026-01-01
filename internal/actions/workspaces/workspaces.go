package workspaces

import (
	"os"
	"path/filepath"

	"github.com/frostyeti/cast/internal/schemas"
)

func New() {

}

func SetupGlobal() error {
	config, err := schemas.NewGlobalWorkspaceConfig()
	if err != nil {
		return err
	}

	dir := config.Dir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file := config.File
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err := config.SaveFile(file); err != nil {
			return err
		}
	}

	inventoryDir := os.Getenv("CAST_GLOBAL_INVENTORY_DIR")
	vendorDir := os.Getenv("CAST_GLOBAL_VENDOR_DIR")
	modulesDir := os.Getenv("CAST_GLOBAL_MODULES_DIR")
	cacheDir := os.Getenv("CAST_GLOBAL_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = filepath.Join(dir, "cache")
	}
	if vendorDir == "" {
		vendorDir = filepath.Join(dir, "vendor")
	}

	if inventoryDir == "" {
		inventoryDir = filepath.Join(dir, "inventory")
	}
	if modulesDir == "" {
		modulesDir = filepath.Join(dir, "modules")
	}

	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// You might want to do something with the config here

	return nil
}

func Setup(path string) error {
	config, err := schemas.NewWorkspaceConfigFromPath(path)
	if err != nil {
		return err
	}

	dir := config.Dir
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file := config.File
	if _, err := os.Stat(file); os.IsNotExist(err) {
		if err := config.SaveFile(file); err != nil {
			return err
		}
	}

	cacheDir := filepath.Join(dir, "cache")
	vendorDir := filepath.Join(dir, "vendor")
	modulesDir := filepath.Join(dir, "modules")
	inventoryDir := filepath.Join(dir, "inventory")

	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	return nil
}

func GetGlobalConfig() schemas.WorkspaceConfig {
	return schemas.WorkspaceConfig{}
}
