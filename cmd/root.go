/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "cast",
	Version:           "0.2.0-alpha.4",
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
	registerDynamicSubcommands()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")

	rootCmd.FParseErrWhitelist.UnknownFlags = true
	rootCmd.CompletionOptions.DisableDefaultCmd = false
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	defaultHelpFn := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if printDirectTaskHelpForRootRequest(cmd) {
			return
		}
		defaultHelpFn(cmd, args)
	})

	rootCmd.Flags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	rootCmd.Flags().StringP("context", "c", context, "Context name to use from the project")
	rootCmd.Flags().StringArrayP("dotenv", "E", []string{}, "List of dotenv files to load")
	rootCmd.Flags().StringToStringP("env", "e", map[string]string{}, "List of environment variables to set")
}

func printDirectTaskHelpForRootRequest(cmd *cobra.Command) bool {
	if cmd != rootCmd {
		return false
	}

	rawArgs := append([]string{}, os.Args[1:]...)
	cleanArgs, wantsHelp := sanitizeDynamicArgs(rawArgs)
	if !wantsHelp || len(cleanArgs) == 0 {
		return false
	}

	target := cleanArgs[0]
	if _, reserved := reservedRootCommandNames()[target]; reserved {
		return false
	}

	tmp := &cobra.Command{}
	tmp.Flags().StringP("project", "p", env.Get("CAST_PROJECT"), "")
	tmp.Flags().StringP("context", "c", env.Get("CAST_CONTEXT"), "")
	tmp.Flags().StringArrayP("dotenv", "E", []string{}, "")
	tmp.Flags().StringToStringP("env", "e", map[string]string{}, "")
	tmp.FParseErrWhitelist.UnknownFlags = true
	_ = tmp.Flags().Parse(rawArgs)

	projectFile, err := resolveProjectFileFromFlagOrCwd(tmp)
	if err != nil || strings.TrimSpace(projectFile) == "" {
		return false
	}

	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return false
	}

	contextName, _ := tmp.Flags().GetString("context")
	if strings.TrimSpace(contextName) == "" {
		contextName = parseContextFromArgs(rawArgs)
	}
	if strings.TrimSpace(contextName) == "" {
		contextName = env.Get("CAST_CONTEXT")
	}
	if strings.TrimSpace(contextName) == "" {
		contextName = "default"
	}

	project.ContextName = contextName
	if err := project.Init(); err != nil {
		return false
	}

	task, ok := lookupTaskForContext(project, target, contextName)
	if !ok {
		return false
	}

	if task.Help != nil && strings.TrimSpace(*task.Help) != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(*task.Help))
		return true
	}

	if task.Desc != nil && strings.TrimSpace(*task.Desc) != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(*task.Desc))
		return true
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), target)
	return true
}
