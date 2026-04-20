package cmd

import (
	"fmt"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:                "config",
	Short:              "Manage castfile config values",
	Args:               cobra.ArbitraryArgs,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileForConfigCommand(cmd, args)
		if err != nil {
			return err
		}

		cleanArgs, wantsHelp := sanitizeDynamicArgs(args)
		handled, err := tryRunConfigTaskOverride(cmd, projectFile, cleanArgs, wantsHelp)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}

		if wantsHelp || len(cleanArgs) == 0 {
			return cmd.Help()
		}

		switch cleanArgs[0] {
		case "set":
			if len(cleanArgs) < 3 {
				return errors.New("usage: cast config set <key> <value>")
			}
			return setProjectConfigValue(projectFile, cleanArgs[1], cleanArgs[2])
		case "get":
			if len(cleanArgs) < 2 {
				return errors.New("usage: cast config get <key>")
			}
			value, found, err := getProjectConfigValue(projectFile, cleanArgs[1])
			if err != nil {
				return err
			}
			if !found {
				return errors.Newf("config key not found: %s", cleanArgs[1])
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		case "rm", "remove", "delete", "del":
			if len(cleanArgs) < 2 {
				return errors.New("usage: cast config rm <key>")
			}
			return removeProjectConfigValue(projectFile, cleanArgs[1])
		default:
			return cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")
	configCmd.PersistentFlags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	configCmd.PersistentFlags().StringP("context", "c", context, "Context name to use from the project")
	_ = configCmd.RegisterFlagCompletionFunc("project", provideProjectFlagCompletion)
	_ = configCmd.RegisterFlagCompletionFunc("context", provideContextFlagCompletion)
}

func resolveProjectFileForConfigCommand(cmd *cobra.Command, args []string) (string, error) {
	tmp := &cobra.Command{}
	tmp.Flags().StringP("project", "p", env.Get("CAST_PROJECT"), "")
	tmp.Flags().StringP("context", "c", env.Get("CAST_CONTEXT"), "")
	tmp.Flags().StringArrayP("dotenv", "E", []string{}, "")
	tmp.Flags().StringToStringP("env", "e", map[string]string{}, "")
	tmp.FParseErrWhitelist.UnknownFlags = true
	_ = tmp.Flags().Parse(args)

	projectFile, err := resolveProjectFileFromFlagOrCwd(tmp)
	if err == nil && strings.TrimSpace(projectFile) != "" {
		return projectFile, nil
	}

	return resolveProjectFileFromFlagOrCwd(cmd)
}

func tryRunConfigTaskOverride(cmd *cobra.Command, projectFile string, args []string, wantsHelp bool) (bool, error) {
	schema, err := loadProjectSchema(projectFile)
	if err != nil {
		return false, err
	}
	if schema == nil || schema.Tasks == nil {
		return false, nil
	}

	if hasConfigSubcmdOverride(schema) {
		if len(args) == 0 {
			if _, ok := schema.Tasks.Get("config:help"); ok {
				return true, runTaskHelpBlockOrTask(cmd, projectFile, "config:help", nil)
			}
			return true, cmd.Help()
		}

		for i := len(args); i >= 1; i-- {
			taskID := "config:" + strings.Join(args[:i], ":")
			if _, ok := schema.Tasks.Get(taskID); !ok {
				continue
			}
			if wantsHelp {
				return true, runTaskHelpBlockOrTask(cmd, projectFile, taskID, nil)
			}
			return true, runTaskByID(cmd, projectFile, taskID, args[i:])
		}

		if wantsHelp {
			if _, ok := schema.Tasks.Get("config:help"); ok {
				return true, runTaskHelpBlockOrTask(cmd, projectFile, "config:help", nil)
			}
		}

		return true, cmd.Help()
	}

	if _, ok := schema.Tasks.Get("config"); ok {
		if wantsHelp {
			return true, runTaskHelpBlockOrTask(cmd, projectFile, "config", nil)
		}
		return true, runTaskByID(cmd, projectFile, "config", args)
	}

	return false, nil
}

func hasConfigSubcmdOverride(schema *types.Project) bool {
	return hasRootSubcmdOverride(schema, "config")
}
