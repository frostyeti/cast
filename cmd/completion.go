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

		// Also check for subdirectories with castfiles if not explicitly in workspace
		entries, err := os.ReadDir(filepath.Dir(projectFile))
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
					tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
					for _, f := range tryFiles {
						if _, err := os.Stat(filepath.Join(filepath.Dir(projectFile), entry.Name(), f)); err == nil {
							candidate := "@" + entry.Name()
							alreadyAdded := false
							for _, c := range completions {
								if c == candidate {
									alreadyAdded = true
									break
								}
							}
							if !alreadyAdded && strings.HasPrefix(candidate, toComplete) {
								completions = append(completions, candidate)
							}
							break
						}
					}
				}
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
			} else {
				// Fallback: check if it's a directory
				fullPath := filepath.Join(filepath.Dir(projectFile), alias)
				if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
					tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
					for _, f := range tryFiles {
						targetFile := filepath.Join(fullPath, f)
						if _, err := os.Stat(targetFile); err == nil {
							project = &projects.Project{}
							_ = project.LoadFromYaml(targetFile)
							break
						}
					}
				}
			}
			break
		}
	}

	// Init to resolve tasks fully
	err = project.Init()
	if err != nil {
		cobra.CompDebugln(err.Error(), true)
	}

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
