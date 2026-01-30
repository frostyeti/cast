package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/go/env"
	"github.com/frostyeti/go/exec"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var tasksExecCmd = &cobra.Command{
	Use:     "exec VAR=NAME [command] [-- args...]",
	Aliases: []string{"x"},
	Short:   "Run a specific task in the project",
	Long:    `Run a specific task defined in the project's configuration.`,
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, a []string) error {
		args := os.Args

		if len(args) > 0 {
			// always will be the cli command
			args = args[1:]
			commandName := args[0]

			if len(args) > 0 && (commandName == "exec" || commandName == "x") {
				args = args[1:]
			} else if len(args) > 0 {
				index := -1
				for i, arg := range args {
					if arg == "exec" || arg == "x" {
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

		cmdArgs := []string{}
		remainingArgs := []string{}
		size := len(args)
		inRemaining := false
		vars := map[string]string{}
		for i := 0; i < size; i++ {
			n := args[i]

			if inRemaining {
				remainingArgs = append(remainingArgs, n)
				continue
			}

			if strings.ContainsRune(n, '=') && !strings.HasPrefix(n, "-") {
				parts := strings.SplitN(n, "=", 2)
				vars[parts[0]] = parts[1]
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

			inRemaining = true
			remainingArgs = append(remainingArgs, n)
		}

		err := flags.Parse(cmdArgs)
		if err != nil {
			cmd.PrintErrf("Error parsing flags: %v\n", err)
			os.Exit(1)
		}

		projectFile, _ = flags.GetString("project")
		contextName, _ = flags.GetString("context")
		envs, _ := flags.GetStringToString("env")
		//dotenvs, _ := flags.GetStringArray("dotenv")

		if contextName == "" {
			contextName = "default"
		}

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

		for k, v := range vars {
			project.Env.Set(k, v)
		}

		for k, v := range envs {
			project.Env.Set(k, v)
		}

		if len(remainingArgs) == 0 {
			return errors.New("no command specified to exec")
		}

		next := []any{}
		for _, r := range remainingArgs {
			next = append(next, r)
		}
		cmd.Println(next...)

		args = []string{}
		if len(remainingArgs) > 1 {
			args = remainingArgs[1:]
		}

		cmd0 := exec.New(remainingArgs[0], args...)
		cmd0.WithEnvMap(project.Env.ToMap())
		o, err := cmd0.Run()
		if err != nil {
			return errors.Newf("failed to execute command %s: %w", strings.Join(remainingArgs, " "), err)
		}

		os.Exit(o.Code)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tasksExecCmd)
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")

	tasksExecCmd.Flags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	tasksExecCmd.Flags().StringP("context", "c", context, "Context name to use from the project")
	tasksExecCmd.Flags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	tasksExecCmd.Flags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
}
