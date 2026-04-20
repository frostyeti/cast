package cmd

import (
	"os"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Manage workspace discovery and aliases",
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		handled, err := tryRunWorkspaceTaskOverride(cmd, args)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}

		return cmd.Help()
	},
}

var workspaceListCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List workspace projects and aliases",
	Args:    cobra.NoArgs,
	RunE:    runWorkspaceListCommand,
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(workspaceListCmd)

	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")
	workspaceCmd.PersistentFlags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	workspaceCmd.PersistentFlags().StringP("context", "c", context, "Context name to use from the project")
	workspaceCmd.PersistentFlags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	workspaceCmd.PersistentFlags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
	_ = workspaceCmd.RegisterFlagCompletionFunc("project", provideProjectFlagCompletion)
	_ = workspaceCmd.RegisterFlagCompletionFunc("context", provideContextFlagCompletion)
}

func tryRunWorkspaceTaskOverride(cmd *cobra.Command, args []string) (bool, error) {
	projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
	if err != nil || strings.TrimSpace(projectFile) == "" {
		return false, nil
	}

	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return false, nil
	}

	contextName := resolveDefaultContextName(cmd, projectFile)
	project.ContextName = contextName
	if err := project.Init(); err != nil {
		return false, nil
	}

	if _, ok := lookupTaskForContext(project, "workspace", contextName); !ok {
		return false, nil
	}

	results, err := project.RunTask(projects.RunTasksParams{
		Targets:     []string{"workspace"},
		Args:        args,
		Context:     cmd.Context(),
		ContextName: contextName,
		Stdout:      cmd.OutOrStdout(),
		Stderr:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return true, errors.Newf("failure with project %s: %w", projectFile, err)
	}

	for _, res := range results {
		if res.Status == runstatus.Error {
			os.Exit(1)
		}
	}

	for _, res := range results {
		if res.Status == runstatus.Cancelled {
			os.Exit(2)
		}
	}

	return true, nil
}
