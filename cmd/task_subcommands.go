package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

var taskInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install remote tasks from a castfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		project := &projects.Project{}
		if err := project.LoadFromYaml(projectFile); err != nil {
			return errors.Newf("failed to load project file %s: %w", projectFile, err)
		}

		if err := project.Init(); err != nil {
			return errors.Newf("failed to initialize project %s: %w", projectFile, err)
		}

		seen := map[string]struct{}{}
		for _, task := range project.Tasks.Values() {
			if task.Uses == nil {
				continue
			}
			uses := strings.TrimSpace(*task.Uses)
			if uses == "" || !projects.IsRemoteTask(uses) {
				continue
			}
			if _, ok := seen[uses]; ok {
				continue
			}
			seen[uses] = struct{}{}

			_, err := projects.FetchRemoteTaskWithOptions(project, uses, project.Schema.TrustedSources, projects.FetchRemoteTaskOptions{
				Stdout: cmd.OutOrStdout(),
			})
			if err != nil {
				return err
			}
		}

		return nil
	},
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update branch/head remote tasks from a castfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		project := &projects.Project{}
		if err := project.LoadFromYaml(projectFile); err != nil {
			return errors.Newf("failed to load project file %s: %w", projectFile, err)
		}

		if err := project.Init(); err != nil {
			return errors.Newf("failed to initialize project %s: %w", projectFile, err)
		}

		seen := map[string]struct{}{}
		for _, task := range project.Tasks.Values() {
			if task.Uses == nil {
				continue
			}
			uses := strings.TrimSpace(*task.Uses)
			if uses == "" || !projects.IsRemoteTask(uses) || !projects.IsVolatileRemoteTaskRef(uses) {
				continue
			}
			if _, ok := seen[uses]; ok {
				continue
			}
			seen[uses] = struct{}{}

			_, err := projects.FetchRemoteTaskWithOptions(project, uses, project.Schema.TrustedSources, projects.FetchRemoteTaskOptions{
				Stdout:       cmd.OutOrStdout(),
				ForceRefresh: true,
			})
			if err != nil {
				return err
			}
		}

		return nil
	},
}

var taskClearCacheCmd = &cobra.Command{
	Use:   "clear-cache",
	Short: "Clear cached remote tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		projectDir := filepath.Dir(projectFile)
		cacheDir := projects.ResolveVolatileRemoteTasksDir(projectDir)
		if !global {
			if err := os.RemoveAll(cacheDir); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Cleared local task cache: %s\n", cacheDir)
			return nil
		}

		globalDir := projects.ResolveRemoteTasksDir(projectDir)
		if err := os.RemoveAll(globalDir); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Cleared global task cache: %s\n", globalDir)
		return nil
	},
}

var taskAddCmd = &cobra.Command{
	Use:   "add [remote-task-url]",
	Short: "Add a task to the current castfile",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		project := &projects.Project{}
		if err := project.LoadFromYaml(projectFile); err != nil {
			return errors.Newf("failed to load project file %s: %w", projectFile, err)
		}
		if err := project.Init(); err != nil {
			return errors.Newf("failed to initialize project %s: %w", projectFile, err)
		}

		name, _ := cmd.Flags().GetString("name")
		usesFlag, _ := cmd.Flags().GetString("uses")
		runFlag, _ := cmd.Flags().GetString("run")

		reader := bufio.NewReader(cmd.InOrStdin())
		remoteUses := ""
		if len(args) > 0 {
			remoteUses = strings.TrimSpace(args[0])
		}

		if remoteUses == "" {
			usesFlag = strings.TrimSpace(usesFlag)
			runFlag = strings.TrimSpace(runFlag)
			name = strings.TrimSpace(name)

			if usesFlag == "" && runFlag == "" {
				if name == "" {
					name, err = promptValue(reader, cmd.OutOrStdout(), "Task name", "")
					if err != nil {
						return err
					}
				}
				runFlag, err = promptValue(reader, cmd.OutOrStdout(), "Run command", "")
				if err != nil {
					return err
				}
				usesFlag = "shell"
			}
		}

		if name == "" {
			defaultName := ""
			if remoteUses != "" {
				defaultName = defaultTaskNameFromUses(remoteUses)
			}
			name, err = promptValue(reader, cmd.OutOrStdout(), "Task name", defaultName)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(name) == "" {
			return errors.New("task name is required")
		}
		if strings.ContainsAny(name, "\n\r\t") {
			return errors.New("task name contains invalid whitespace")
		}

		newTask := map[string]any{}
		withValues := map[string]any{}

		if remoteUses != "" {
			if !projects.IsRemoteTask(remoteUses) {
				return errors.Newf("remote task url must be a valid remote uses value: %s", remoteUses)
			}

			entryPath, err := projects.FetchRemoteTaskWithOptions(project, remoteUses, project.Schema.TrustedSources, projects.FetchRemoteTaskOptions{
				Stdout: cmd.OutOrStdout(),
			})
			if err != nil {
				return err
			}

			newTask["uses"] = remoteUses
			if projects.IsCastTaskDefinitionFile(entryPath) {
				castTask := &types.CastTask{}
				if err := castTask.ReadFromYaml(entryPath); err == nil {
					keys := make([]string, 0, len(castTask.Inputs))
					for key := range castTask.Inputs {
						keys = append(keys, key)
					}
					sort.Strings(keys)
					for _, key := range keys {
						input := castTask.Inputs[key]
						if !input.Required {
							continue
						}
						promptLabel := key
						if input.Description != "" {
							promptLabel = fmt.Sprintf("%s (%s)", key, input.Description)
						}
						value, err := promptValue(reader, cmd.OutOrStdout(), promptLabel, input.Default)
						if err != nil {
							return err
						}
						if strings.TrimSpace(value) == "" {
							return errors.Newf("required input %s cannot be empty", key)
						}
						withValues[key] = value
					}
				}
			}
		} else {
			if strings.TrimSpace(usesFlag) != "" {
				newTask["uses"] = strings.TrimSpace(usesFlag)
			}
			if strings.TrimSpace(runFlag) != "" {
				newTask["run"] = strings.TrimSpace(runFlag)
			}

			if _, hasUses := newTask["uses"]; !hasUses {
				newTask["uses"] = "shell"
			}
		}

		if len(withValues) > 0 {
			newTask["with"] = withValues
		}

		if err := writeTaskToProjectFile(projectFile, name, newTask); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added task '%s' to %s\n", name, projectFile)
		return nil
	},
}

func promptValue(reader *bufio.Reader, out io.Writer, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	value := strings.TrimSpace(line)
	if value == "" {
		return defaultValue, nil
	}

	return value, nil
}

func defaultTaskNameFromUses(uses string) string {
	uses = strings.TrimSpace(uses)
	if uses == "" {
		return ""
	}

	if idx := strings.LastIndex(uses, "/"); idx >= 0 && idx < len(uses)-1 {
		uses = uses[idx+1:]
	}
	if idx := strings.Index(uses, "@"); idx >= 0 {
		uses = uses[:idx]
	}
	uses = strings.Trim(uses, ":")
	if uses == "" {
		return "task"
	}
	return uses
}

func resolveProjectFileFromFlagOrCwd(cmd *cobra.Command) (string, error) {
	projectFile, _ := cmd.Flags().GetString("project")
	if strings.TrimSpace(projectFile) == "" {
		projectFile, _ = cmd.InheritedFlags().GetString("project")
	}
	if strings.TrimSpace(projectFile) != "" {
		if filepath.IsAbs(projectFile) {
			return projectFile, nil
		}
		abs, err := filepath.Abs(projectFile)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	currentDir := cwd
	for currentDir != "/" && currentDir != "" {
		for _, f := range []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"} {
			fullPath := filepath.Join(currentDir, f)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath, nil
			}
		}

		nextDir := filepath.Dir(currentDir)
		if nextDir == currentDir {
			break
		}
		currentDir = nextDir
	}

	return "", errors.New("no castfile found in current or parent directories")
}

func writeTaskToProjectFile(projectFile, taskName string, taskDef map[string]any) error {
	data, err := os.ReadFile(projectFile)
	if err != nil {
		return err
	}

	root := map[string]any{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}

	tasks, ok := root["tasks"].(map[string]any)
	if !ok || tasks == nil {
		tasks = map[string]any{}
	}
	tasks[taskName] = taskDef
	root["tasks"] = tasks

	out, err := yaml.Marshal(root)
	if err != nil {
		return err
	}

	return os.WriteFile(projectFile, out, 0o644)
}

func init() {
	for _, c := range []*cobra.Command{taskInstallCmd, taskUpdateCmd, taskClearCacheCmd, taskAddCmd} {
		taskCmd.AddCommand(c)
	}

	taskClearCacheCmd.Flags().Bool("global", false, "Clear global stable task cache")
	taskAddCmd.Flags().StringP("uses", "u", "", "Task uses value")
	taskAddCmd.Flags().StringP("name", "n", "", "Task name")
	taskAddCmd.Flags().StringP("run", "r", "", "Task run command")
}
