package cmd

import (
	"fmt"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:                "context",
	Short:              "Manage castfile context values",
	Args:               cobra.ArbitraryArgs,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileForConfigCommand(cmd, args)
		if err != nil {
			return err
		}

		cleanArgs, wantsHelp := sanitizeDynamicArgs(args)
		handled, err := tryRunContextTaskOverride(cmd, projectFile, cleanArgs, wantsHelp)
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
		case "use", "set":
			if len(cleanArgs) < 2 {
				return errors.New("usage: cast context use <name>")
			}
			return setProjectConfigValue(projectFile, "context", cleanArgs[1])
		case "get", "show":
			value, found, err := getProjectConfigValue(projectFile, "context")
			if err != nil {
				return err
			}
			if !found {
				return errors.New("config key not found: context")
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		case "rm", "remove", "delete", "del":
			return removeProjectConfigValue(projectFile, "context")
		default:
			return cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")
	contextCmd.PersistentFlags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	contextCmd.PersistentFlags().StringP("context", "c", context, "Context name to use from the project")
}

func tryRunContextTaskOverride(cmd *cobra.Command, projectFile string, args []string, wantsHelp bool) (bool, error) {
	schema, err := loadProjectSchema(projectFile)
	if err != nil {
		return false, err
	}
	if schema == nil || schema.Tasks == nil {
		return false, nil
	}

	if hasRootSubcmdOverride(schema, "context") {
		if len(args) == 0 {
			if _, ok := schema.Tasks.Get("context:help"); ok {
				return true, runTaskHelpBlockOrTask(cmd, projectFile, "context:help", nil)
			}
			return true, cmd.Help()
		}

		for i := len(args); i >= 1; i-- {
			taskID := "context:" + strings.Join(args[:i], ":")
			if _, ok := schema.Tasks.Get(taskID); !ok {
				continue
			}
			if wantsHelp {
				return true, runTaskHelpBlockOrTask(cmd, projectFile, taskID, nil)
			}
			return true, runTaskByID(cmd, projectFile, taskID, args[i:])
		}

		if wantsHelp {
			if _, ok := schema.Tasks.Get("context:help"); ok {
				return true, runTaskHelpBlockOrTask(cmd, projectFile, "context:help", nil)
			}
		}

		return true, cmd.Help()
	}

	if _, ok := schema.Tasks.Get("context"); ok {
		if wantsHelp {
			return true, runTaskHelpBlockOrTask(cmd, projectFile, "context", nil)
		}
		return true, runTaskByID(cmd, projectFile, "context", args)
	}

	return false, nil
}

func hasRootSubcmdOverride(schema *types.Project, name string) bool {
	if schema == nil {
		return false
	}
	for _, raw := range schema.Subcmds {
		if normalizeSubcmdPath(raw) == name {
			return true
		}
	}
	return false
}
