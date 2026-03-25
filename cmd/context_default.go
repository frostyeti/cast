package cmd

import (
	"os"
	"strings"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

func resolveDefaultContextName(cmd *cobra.Command, projectFile string) string {
	contextName, _ := cmd.Flags().GetString("context")
	if strings.TrimSpace(contextName) == "" {
		contextName, _ = cmd.InheritedFlags().GetString("context")
	}
	if strings.TrimSpace(contextName) == "" {
		contextName = parseContextFromArgs(os.Args[1:])
	}
	if strings.TrimSpace(contextName) == "" {
		contextName = env.Get("CAST_CONTEXT")
	}
	if strings.TrimSpace(contextName) != "" {
		return strings.TrimSpace(contextName)
	}

	if strings.TrimSpace(projectFile) != "" {
		project := &projects.Project{}
		if err := project.LoadFromYaml(projectFile); err == nil {
			if project.Schema.Config != nil && project.Schema.Config.Context != nil {
				v := strings.TrimSpace(*project.Schema.Config.Context)
				if v != "" {
					return v
				}
			}
		}
	}

	return "default"
}
