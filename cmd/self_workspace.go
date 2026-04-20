package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/spf13/cobra"
)

var selfWorkspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Manage workspace discovery and aliases",
}

var selfWorkspaceListCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List workspace projects and aliases",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		project := &projects.Project{}
		if err := project.LoadFromYaml(projectFile); err != nil {
			return errors.Newf("failed to load project file %s: %w", projectFile, err)
		}

		if err := project.InitWorkspace(); err != nil {
			return errors.Newf("failed to initialize workspace for project %s: %w", projectFile, err)
		}

		entries := append([]*projects.ProjectInfo{}, project.WorkspaceEntries...)
		if len(entries) == 0 {
			for _, info := range project.Workspace {
				entries = append(entries, info)
			}
		}

		sort.Slice(entries, func(i, j int) bool {
			left := strings.TrimSpace(entries[i].Rel)
			right := strings.TrimSpace(entries[j].Rel)
			if left == "" {
				left = entries[i].Path
			}
			if right == "" {
				right = entries[j].Path
			}
			return left < right
		})

		if len(entries) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No workspace projects found")
			return nil
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "ALIAS\tPATH")
		for _, info := range entries {
			if info == nil {
				continue
			}

			alias := strings.TrimSpace(info.Alias)
			if alias == "" {
				alias = "-"
			}

			pathValue := strings.TrimSpace(info.Rel)
			if pathValue == "" {
				pathValue = info.Path
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", alias, pathValue)
		}

		return nil
	},
}
