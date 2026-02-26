/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "cast",
	Version:           "0.0.0",
	Short:             "Cast is a task runner and automation tool",
	Long:              "Cast is a task runner and automation tool",
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: provideProjectCompletion,
	RunE:              tasksRunCmd.RunE,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")

	rootCmd.Flags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	rootCmd.Flags().StringP("context", "c", context, "Context name to use from the project")
	rootCmd.Flags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	rootCmd.Flags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
}
