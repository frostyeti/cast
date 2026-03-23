package cmd

import (
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"tasks"},
	Short:   "Manage and run tasks",
}

func init() {
	rootCmd.AddCommand(taskCmd)
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")
	taskCmd.PersistentFlags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	taskCmd.PersistentFlags().StringP("context", "c", context, "Context name to use from the project")
	taskCmd.PersistentFlags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	taskCmd.PersistentFlags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
}
