package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/spf13/cobra"
)

func provideProjectCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string

	// Try to find the nearest castfile
	projectFile := ""
	cwd, err := os.Getwd()
	if err == nil {
		currentDir := cwd
		for currentDir != "/" && currentDir != "" {
			tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
			for _, f := range tryFiles {
				fullPath := filepath.Join(currentDir, f)
				if _, err := os.Stat(fullPath); err == nil {
					projectFile = fullPath
					break
				}
			}
			if projectFile != "" {
				break
			}
			nextDir := filepath.Dir(currentDir)
			if nextDir == currentDir {
				break
			}
			currentDir = nextDir
		}
	}

	if projectFile == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	project := &projects.Project{}
	err = project.LoadFromYaml(projectFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	project.InitWorkspace()

	// If completing a workspace directive e.g. @child
	if strings.HasPrefix(toComplete, "@") {
		for alias := range project.Workspace {
			candidate := "@" + alias
			if strings.HasPrefix(candidate, toComplete) {
				completions = append(completions, candidate)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// Check if any argument is a workspace target
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			alias := arg[1:]
			hashIndex := strings.Index(alias, ":")
			if hashIndex != -1 {
				alias = alias[:hashIndex]
			}
			if wp, ok := project.Workspace[alias]; ok {
				project = &projects.Project{}
				_ = project.LoadFromYaml(wp.Path)
			}
			break
		}
	}

	// Init to resolve tasks fully
	_ = project.Init()

	// Suggest tasks
	for _, taskName := range project.Tasks.Keys() {
		if strings.HasPrefix(taskName, toComplete) {
			task, _ := project.Tasks.Get(taskName)
			desc := taskName
			if task.Desc != nil && *task.Desc != "" {
				desc = taskName + "\t" + *task.Desc
			}
			completions = append(completions, desc)
		}
	}

	// Suggest jobs
	if project.Schema.Jobs != nil {
		for _, jobName := range project.Schema.Jobs.Keys() {
			if strings.HasPrefix(jobName, toComplete) {
				job, _ := project.Schema.Jobs.Get(jobName)
				desc := jobName
				if job.Desc != "" {
					desc = jobName + "\t" + job.Desc
				} else if job.Name != "" {
					desc = jobName + "\t" + job.Name
				}
				completions = append(completions, desc)
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
