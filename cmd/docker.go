package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/paths"
	"github.com/spf13/cobra"
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Manage docker tasks",
}

var dockerPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge downloaded docker images used by cast tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, err := paths.UserDataDir()
		if err != nil {
			return err
		}

		trackingFile := filepath.Join(dataDir, "cast", "docker_images.txt")
		if _, err := os.Stat(trackingFile); os.IsNotExist(err) {
			fmt.Println("No cast docker images tracking file found.")
			return nil
		}

		content, err := os.ReadFile(trackingFile)
		if err != nil {
			return err
		}

		images := strings.Split(string(content), "\n")
		var remaining []string

		for _, image := range images {
			img := strings.TrimSpace(image)
			if img == "" {
				continue
			}

			fmt.Printf("Purging docker image: %s\n", img)
			dockerCmd := exec.Command("docker", "rmi", img)
			dockerCmd.Stdout = os.Stdout
			dockerCmd.Stderr = os.Stderr
			if err := dockerCmd.Run(); err != nil {
				fmt.Printf("Failed to purge %s, keeping in list.\n", img)
				remaining = append(remaining, img)
			}
		}

		if len(remaining) == 0 {
			os.Remove(trackingFile)
		} else {
			os.WriteFile(trackingFile, []byte(strings.Join(remaining, "\n")+"\n"), 0644)
		}

		return nil
	},
}

func init() {
	toolCmd.AddCommand(dockerCmd)
	dockerCmd.AddCommand(dockerPurgeCmd)
}
