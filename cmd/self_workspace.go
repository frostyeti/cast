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
	RunE:    runWorkspaceListCommand,
}

func runWorkspaceListCommand(cmd *cobra.Command, args []string) error {
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

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No workspace projects found")
		return nil
	}

	type workspaceAliasEntry struct {
		alias string
		path  string
	}

	aliased := []workspaceAliasEntry{}
	unaliased := []string{}

	for _, info := range entries {
		if info == nil {
			continue
		}

		pathValue := strings.TrimSpace(info.Rel)
		if pathValue == "" {
			pathValue = info.Path
		}

		alias := strings.TrimSpace(info.Alias)
		if alias == "" {
			unaliased = append(unaliased, pathValue)
			continue
		}

		aliased = append(aliased, workspaceAliasEntry{alias: alias, path: pathValue})
	}

	sort.Slice(aliased, func(i, j int) bool {
		if aliased[i].alias == aliased[j].alias {
			return aliased[i].path < aliased[j].path
		}
		return aliased[i].alias < aliased[j].alias
	})
	sort.Strings(unaliased)

	if len(aliased) > 0 {
		maxAliasLen := 0
		for _, entry := range aliased {
			if len(entry.alias) > maxAliasLen {
				maxAliasLen = len(entry.alias)
			}
		}
		aliasWidth := maxAliasLen + 2

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "ALIASES")
		for _, entry := range aliased {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-*s%s\n", aliasWidth, entry.alias, entry.path)
		}
	}

	if len(unaliased) > 0 {
		if len(aliased) > 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "PATHS")
		for _, path := range unaliased {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
		}
	}

	return nil
}
