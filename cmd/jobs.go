package cmd

import (
	"fmt"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/spf13/cobra"
)

var jobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage and run jobs",
}

var jobListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List jobs in the project",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		project, _, err := loadProjectForJobCommand(cmd)
		if err != nil {
			return err
		}

		if project.Schema.Jobs == nil || project.Schema.Jobs.Len() == 0 {
			return nil
		}

		max := 7
		for _, jobName := range project.Schema.Jobs.Keys() {
			if len(jobName) > max {
				max = len(jobName) + 5
			}
		}

		for _, jobName := range project.Schema.Jobs.Keys() {
			job, _ := project.Schema.Jobs.Get(jobName)
			desc := job.Desc
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%*s  %s\n", -max, jobName, desc)
		}

		return nil
	},
}

var jobRunCmd = &cobra.Command{
	Use:   "run <job> [-- args...]",
	Short: "Run a job in the project",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, contextName, err := loadProjectForJobCommand(cmd)
		if err != nil {
			return err
		}

		runDownstream, _ := cmd.Flags().GetBool("downstream")
		runParams := projects.RunJobParams{
			JobID:         args[0],
			Context:       cmd.Context(),
			ContextName:   contextName,
			Args:          args[1:],
			RunDownstream: runDownstream,
		}

		if err := project.RunJob(runParams); err != nil {
			return errors.Newf("failure running job %s: %w", args[0], err)
		}

		return nil
	},
}

func loadProjectForJobCommand(cmd *cobra.Command) (*projects.Project, string, error) {
	projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
	if err != nil {
		return nil, "", err
	}

	contextName := resolveDefaultContextName(cmd, projectFile)

	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return nil, "", errors.Newf("failed to load project file %s: %w", projectFile, err)
	}
	project.ContextName = contextName
	if err := project.Init(); err != nil {
		return nil, "", errors.Newf("failed to initialize project %s: %w", projectFile, err)
	}

	return project, contextName, nil
}

func init() {
	rootCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(jobRunCmd)
	jobCmd.AddCommand(jobListCmd)

	jobRunCmd.Flags().Bool("downstream", true, "Run downstream dependent jobs")
}
