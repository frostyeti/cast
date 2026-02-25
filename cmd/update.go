package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update and refresh local task and module caches",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, _ := cmd.Flags().GetString("project")
		if projectFile == "" {
			projectFile = env.Get("CAST_PROJECT")
		}

		if projectFile == "" {
			cwd, err := os.Getwd()
			if err == nil {
				currentDir := cwd
				for currentDir != "/" && currentDir != "" {
					tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
					for _, f := range tryFiles {
						fullPath := filepath.Join(currentDir, f)
						if _, err := os.Stat(fullPath); err == nil {
							projectFile = fullPath
							break
						}
					}
					if projectFile != "" {
						break
					}
					nextDir := filepath.Dir(currentDir)
					if nextDir == currentDir {
						break
					}
					currentDir = nextDir
				}
			}
		}

		if projectFile == "" {
			return fmt.Errorf("no castfile found in current or parent directories")
		}

		if !filepath.IsAbs(projectFile) {
			abs, err := filepath.Abs(projectFile)
			if err != nil {
				return err
			}
			projectFile = abs
		}

		tasksCache := filepath.Join(filepath.Dir(projectFile), ".cast", "tasks")
		fmt.Printf("Clearing tasks cache at %s\n", tasksCache)

		if err := os.RemoveAll(tasksCache); err != nil {
			return err
		}

		fmt.Println("Successfully refreshed caches.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
