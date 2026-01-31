package cmd

import (
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

var tasksRunCmd = &cobra.Command{
	Use:     "run [task name]",
	Aliases: []string{"r"},
	Short:   "Run a specific task in the project",
	Long:    `Run a specific task defined in the project's configuration.`,
	RunE: func(cmd *cobra.Command, a []string) error {
		args := os.Args

		if len(args) > 0 {
			// always will be the cli command
			args = args[1:]

			if len(args) > 0 && args[0] == "run" {
				args = args[1:]
			} else if len(args) > 0 {
				index := -1
				for i, arg := range args {
					if arg == "run" {
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
		inTargets := false
		for i := 0; i < size; i++ {
			n := args[i]
			if n == "--" {
				inRemaining = true
				continue
			}

			if inTargets && n[0] == '-' {
				remainingArgs = append(remainingArgs, n)
				inRemaining = true
				continue
			}

			if inRemaining {
				remainingArgs = append(remainingArgs, n)
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

			inTargets = true

			targets = append(targets, n)
		}

		if len(targets) == 0 {
			targets = append(targets, "default")
		}

		err := flags.Parse(cmdArgs)
		if err != nil {
			cmd.PrintErrf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		projectFile, _ = flags.GetString("project")
		contextName, _ = flags.GetString("context")
		projectName := ""

		remove := []int{}
		for i, target := range targets {
			if strings.HasPrefix(target, "@") {
				projectFile = target[1:]
				hashIndex := strings.Index(projectFile, ":")
				if hashIndex != -1 {
					contextName = projectFile[hashIndex+1:]
					projectFile = projectFile[:hashIndex]
				}
				remove = append(remove, i)
				continue
			}
		}

		for i := len(remove) - 1; i >= 0; i-- {
			index := remove[i]
			if index >= 0 && index < len(targets) {
				targets = append(targets[:index], targets[index+1:]...)
			}
		}

		if contextName == "" {
			contextName = "default"
		}

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
				return errors.Newf("project '%s' not found in workspace", projectName)
			}

			if workspaceProject.Project == nil {
				workspaceProject.Project = &projects.Project{}
			}
			project = workspaceProject.Project
			projectFile = workspaceProject.Path

			err = project.LoadFromYaml(workspaceProject.Path)
			if err != nil {
				return errors.Newf("failed to load project file %s: %w", workspaceProject.Path, err)
			}
		}

		project.ContextName = contextName
		err = project.Init()
		if err != nil {
			return errors.Newf("failed to initialize project %s: %w", projectFile, err)
		}

		params := projects.RunTasksParams{
			Targets:     targets,
			Args:        remainingArgs,
			Context:     cmd.Context(),
			ContextName: contextName,
		}

		results, err := project.RunTask(params)
		if err != nil {
			return errors.Newf("failure with project %s: %w", projectFile, err)
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
	},
}

func init() {
	rootCmd.AddCommand(tasksRunCmd)
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")

	tasksRunCmd.Flags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	tasksRunCmd.Flags().StringP("context", "c", context, "Context name to use from the project")
	tasksRunCmd.Flags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	tasksRunCmd.Flags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
}
