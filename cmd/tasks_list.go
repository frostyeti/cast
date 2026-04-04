package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var taskListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all available tasks in the project",
	Long:    `List all available tasks defined in the project's configuration.`,
	RunE: func(cmd *cobra.Command, a []string) error {
		args := os.Args
		invokedFromTaskNamespace := false
		invokedViaListShortcut := false
		if len(args) > 1 {
			switch args[1] {
			case "task", "tasks":
				invokedFromTaskNamespace = true
				if len(args) > 2 && (args[2] == "list" || args[2] == "ls") {
					invokedViaListShortcut = true
				}
			case "list", "ls":
				invokedViaListShortcut = true
			}
		}

		if len(args) > 0 {
			// always will be the cli command
			args = args[1:]

			if len(args) > 0 && (args[0] == "task" || args[0] == "tasks") {
				args = args[1:]
			}

			if len(args) > 0 && (args[0] == "list" || args[0] == "ls") {
				args = args[1:]
			} else if len(args) > 0 {
				index := -1
				for i, arg := range args {
					if arg == "list" || arg == "ls" {
						index = i
						break
					}
				}

				if index != -1 {
					args = append(args[:index], args[index+1:]...)
				}
			}
		}

		flags := pflag.NewFlagSet("", pflag.ContinueOnError)
		projectFile := env.Get("CAST_PROJECT")
		contextName := env.Get("CAST_CONTEXT")

		flags.StringP("project", "p", projectFile, "Path to the project file (castfile.yaml)")
		flags.StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
		flags.StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
		flags.StringP("context", "c", contextName, "Context to use.")

		targets := []string{}
		cmdArgs := []string{}
		remainingArgs := []string{}
		size := len(args)
		inRemaining := false
		for i := 0; i < size; i++ {
			n := args[i]
			if n == "--" {
				inRemaining = true
				continue
			}

			if inRemaining {
				remainingArgs = append(remainingArgs, args[i])
				continue
			}

			if len(n) > 0 && n[0] == '-' {
				cmdArgs = append(cmdArgs, n)
				j := i + 1
				if j < size && len(args[j]) > 0 && args[j][0] != '-' {
					cmdArgs = append(cmdArgs, args[j])
					i++ // Skip the next argument as it's a value for the flag
				}

				continue
			}

			targets = append(targets, n)
			inRemaining = true
		}

		targetProvided := len(targets) > 0

		err := flags.Parse(cmdArgs)
		if err != nil {
			cmd.PrintErrf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		projectFile, _ = flags.GetString("project")
		contextName, _ = flags.GetString("context")

		projectName := ""

		if projectFile != "" {
			info, err := os.Stat(projectFile)

			if err != nil {
				if os.IsNotExist(err) {
					projectName = projectFile
					projectFile = ""
				} else {
					return errors.Newf("failed to access project file %s: %w", projectFile, err)
				}
			} else {
				if info != nil && info.IsDir() {
					projectName = projectFile
					projectFile = ""
					tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
					for _, f := range tryFiles {
						fullPath := filepath.Join(projectName, f)
						if _, err := os.Stat(fullPath); err == nil {
							projectFile = fullPath
							projectName = ""
							break
						}
					}
				}
			}

		}

		if projectFile == "" {
			tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
			for _, f := range tryFiles {
				if _, err := os.Stat(f); err == nil {
					projectFile = f
					break
				}
			}
		}

		if projectFile == "" {
			return errors.New("no castfile found in current directory")
		}

		project := &projects.Project{}
		err = project.LoadFromYaml(projectFile)
		if err != nil {
			return errors.Newf("failed to load project file %s: %w", projectFile, err)
		}

		if projectName != "" {
			err := project.InitWorkspace()
			if err != nil {
				return errors.Newf("failed to initialize workspace for project %s: %w", projectName, err)
			}

			workspaceProject, ok := project.Workspace[projectName]
			if !ok {
				return errors.Newf("project %s not found in workspace", projectName)
			}

			if workspaceProject.Project == nil {
				workspaceProject.Project = &projects.Project{}
			}
			project = workspaceProject.Project
			projectFile = workspaceProject.Path
			if err := project.LoadFromYaml(workspaceProject.Path); err != nil {
				return errors.Newf("failed to load project file %s: %w", workspaceProject.Path, err)
			}
		}

		if strings.TrimSpace(contextName) == "" {
			contextName = resolveDefaultContextName(cmd, projectFile)
		}

		if !invokedFromTaskNamespace && invokedViaListShortcut && !targetProvided {
			taskNameToRun := "list"
			if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "ls") {
				taskNameToRun = "ls"
			}

			if task, ok := project.Schema.Tasks.Get(taskNameToRun); ok {
				runParams := projects.RunTasksParams{
					Targets:     []string{task.Name},
					Args:        remainingArgs,
					Context:     cmd.Context(),
					ContextName: contextName,
					Stdout:      cmd.OutOrStdout(),
					Stderr:      cmd.ErrOrStderr(),
				}

				results, runErr := project.RunTask(runParams)
				if runErr != nil {
					return errors.Newf("failure with project %s: %w", projectFile, runErr)
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

				return nil
			}
		}

		results, err := project.ListTasks()
		if err != nil {
			return errors.Newf("failure with project %s:%w", projectFile, err)
		}

		max := 7
		for _, taskName := range results.Keys() {
			if len(taskName) > max {
				max = len(taskName) + 5
			}
		}

		for _, taskName := range results.Keys() {
			task, _ := results.Get(taskName)
			desc := ""
			if task.Desc != nil {
				desc = *task.Desc
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-*s  %s\n", max, taskName, desc)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(taskListCmd)
	project := env.Get("CAST_PROJECT")

	taskListCmd.Flags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
}
