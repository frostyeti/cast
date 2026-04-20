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

var taskRunCmd = &cobra.Command{
	Use:               "run [task name]",
	Aliases:           []string{"r"},
	Short:             "Run a specific task in the project",
	Long:              `Run a specific task defined in the project's configuration.`,
	ValidArgsFunction: provideProjectCompletion,
	RunE:              tasksRunCmd.RunE,
}

var taskListCmdAlias = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all available tasks in the project",
	Long:    `List all available tasks defined in the project's configuration.`,
	RunE:    taskListCmd.RunE,
}

var taskExecCmdAlias = &cobra.Command{
	Use:     "exec VAR=NAME [command] [-- args...]",
	Aliases: []string{"x"},
	Short:   "Executes a shell commmand using the environment variables from the local castfile",
	Long:    `Executes a shell command using the environment variables for the local castfile.`,
	Args:    cobra.ArbitraryArgs,
	RunE:    tasksExecCmd.RunE,
}

func init() {
	rootCmd.AddCommand(taskCmd)
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")
	taskCmd.PersistentFlags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	taskCmd.PersistentFlags().StringP("context", "c", context, "Context name to use from the project")
	taskCmd.PersistentFlags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	taskCmd.PersistentFlags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
	_ = taskCmd.RegisterFlagCompletionFunc("project", provideProjectFlagCompletion)
	_ = taskCmd.RegisterFlagCompletionFunc("context", provideContextFlagCompletion)

	taskRunCmd.Flags().StringP("job", "j", "", "Job name to run (executes job and downstream jobs if any)")
	taskCmd.AddCommand(taskRunCmd)
	taskCmd.AddCommand(taskListCmdAlias)
	taskCmd.AddCommand(taskExecCmdAlias)
}
