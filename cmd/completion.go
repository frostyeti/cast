package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
)

var supportedProjectCompletionFiles = []string{"castfile", ".castfile", "castfile.yaml"}

func provideProjectCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projectFile, contextName, err := completionProjectAndContext(cmd, args)
	if err != nil || strings.TrimSpace(projectFile) == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	project, err := loadProjectForCompletion(projectFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// If completing a workspace directive e.g. @child
	if strings.HasPrefix(toComplete, "@") {
		completions := workspaceAliasCompletions(project, projectFile, toComplete)
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// Check if any argument is a workspace target
	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			resolvedProjectFile, ok := resolveWorkspaceAliasProjectFile(project, projectFile, arg)
			if ok {
				project, err = loadProjectForCompletion(resolvedProjectFile)
				if err == nil {
					contextName = contextNameFromWorkspaceAlias(arg, contextName)
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

	completions := []string{}
	completions = append(completions, completionTaskTargets(project, contextName, toComplete)...)
	completions = append(completions, completionJobTargets(project, contextName, toComplete)...)

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completionProjectAndContext(cmd *cobra.Command, args []string) (string, string, error) {
	projectFile, err := projectFileForFlagCompletion(cmd, args)
	if err != nil {
		return "", "", err
	}

	contextName := contextNameForCompletion(cmd, args, projectFile)
	return projectFile, contextName, nil
}

func nearestProjectFile() (string, error) {
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

	return "", nil
}

func loadProjectForCompletion(projectFile string) (*projects.Project, error) {
	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return nil, err
	}
	if err := project.InitWorkspace(); err != nil {
		return nil, err
	}
	return project, nil
}

func workspaceAliasCompletions(project *projects.Project, projectFile, toComplete string) []string {
	seen := map[string]struct{}{}
	completions := []string{}
	if project != nil {
		if project.Schema.Workspace == nil {
			return completions
		}

		for alias := range project.Workspace {
			candidate := "@" + alias
			if strings.HasPrefix(candidate, toComplete) {
				seen[candidate] = struct{}{}
				completions = append(completions, candidate)
			}
		}
	}

	sort.Strings(completions)
	return completions
}

func resolveWorkspaceAliasProjectFile(project *projects.Project, projectFile, alias string) (string, bool) {
	alias = strings.TrimPrefix(strings.TrimSpace(alias), "@")
	if idx := strings.Index(alias, ":"); idx >= 0 {
		alias = alias[:idx]
	}
	if alias == "" {
		return "", false
	}
	if project != nil {
		if wp, ok := project.Workspace[alias]; ok {
			return wp.Path, true
		}
		for name, wp := range project.Workspace {
			if strings.EqualFold(name, alias) {
				return wp.Path, true
			}
		}
	}

	fullPath := filepath.Join(filepath.Dir(projectFile), alias)
	info, err := os.Stat(fullPath)
	if err != nil || !info.IsDir() {
		return "", false
	}
	for _, f := range []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"} {
		targetFile := filepath.Join(fullPath, f)
		if _, err := os.Stat(targetFile); err == nil {
			return targetFile, true
		}
	}
	return "", false
}

func provideProjectFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	completions := []string{}
	seen := map[string]struct{}{}

	hasWorkspace := false
	var workspaceProject *projects.Project
	var workspaceProjectFile string

	if projectFile, err := nearestProjectFile(); err == nil && strings.TrimSpace(projectFile) != "" {
		workspaceProjectFile = projectFile
		if project, err := loadProjectForCompletion(projectFile); err == nil {
			workspaceProject = project
			hasWorkspace = project.Schema.Workspace != nil
		}
	}

	if strings.HasPrefix(toComplete, "@") || toComplete == "" {
		if hasWorkspace && workspaceProject != nil && strings.TrimSpace(workspaceProjectFile) != "" {
			for _, candidate := range workspaceAliasCompletions(workspaceProject, workspaceProjectFile, toComplete) {
				if _, ok := seen[candidate]; !ok {
					seen[candidate] = struct{}{}
					completions = append(completions, candidate)
				}
			}
		}
	}

	if strings.HasPrefix(toComplete, "@") {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	baseDir, namePrefix := completionPathParts(toComplete)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return completions, cobra.ShellCompDirectiveNoSpace
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, namePrefix) {
			continue
		}
		candidate := name
		if baseDir != "." {
			candidate = filepath.Join(baseDir, name)
		}
		if entry.IsDir() {
			if !completionDirectoryHasProjectFile(candidate) {
				continue
			}

			if _, ok := seen[candidate+string(filepath.Separator)]; !ok {
				seen[candidate+string(filepath.Separator)] = struct{}{}
				completions = append(completions, candidate+string(filepath.Separator))
			}
			continue
		}
		if !isProjectFileName(name) {
			continue
		}
		if _, ok := seen[candidate]; !ok {
			seen[candidate] = struct{}{}
			completions = append(completions, candidate)
		}
	}
	sort.Strings(completions)
	return completions, cobra.ShellCompDirectiveNoSpace
}

func completionPathParts(toComplete string) (string, string) {
	sep := string(filepath.Separator)
	if toComplete == "." {
		return ".", ""
	}
	if toComplete == sep {
		return sep, ""
	}

	baseDir := "."
	namePrefix := toComplete

	if strings.Contains(toComplete, sep) {
		if strings.HasSuffix(toComplete, sep) {
			baseDir = strings.TrimSuffix(toComplete, sep)
			if strings.TrimSpace(baseDir) == "" {
				baseDir = sep
			}
			namePrefix = ""
		} else {
			baseDir = filepath.Dir(toComplete)
			namePrefix = filepath.Base(toComplete)
			if strings.TrimSpace(baseDir) == "" {
				baseDir = "."
			}
		}
	}

	return baseDir, namePrefix
}

func isProjectFileName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, supported := range supportedProjectCompletionFiles {
		if name == supported {
			return true
		}
	}

	return false
}

func completionDirectoryHasProjectFile(dir string) bool {
	_, ok := projectFileFromDirectoryForCompletion(dir)
	return ok
}

func provideContextFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	projectFile, err := projectFileForFlagCompletion(cmd, args)
	if err != nil || strings.TrimSpace(projectFile) == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	schema, err := loadProjectSchema(projectFile)
	if err != nil || schema == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	contexts := knownProjectContexts(schema)
	completions := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		if strings.HasPrefix(ctx, toComplete) {
			completions = append(completions, ctx)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

func projectFileForFlagCompletion(cmd *cobra.Command, args []string) (string, error) {
	tmp := completionFlagSnapshot(cmd, args)
	projectFile, err := resolveProjectFileFromFlagOrCwd(tmp)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(projectFile)
	if err == nil && info != nil && info.IsDir() {
		if resolved, ok := projectFileFromDirectoryForCompletion(projectFile); ok {
			return resolved, nil
		}

		nearest, nearestErr := nearestProjectFile()
		if nearestErr != nil {
			return "", nearestErr
		}
		if strings.TrimSpace(nearest) != "" {
			return nearest, nil
		}
	}

	return projectFile, nil
}

func projectFileFromDirectoryForCompletion(dir string) (string, bool) {
	for _, file := range supportedProjectCompletionFiles {
		candidate := filepath.Join(dir, file)
		info, err := os.Stat(candidate)
		if err == nil && info != nil && !info.IsDir() {
			return candidate, true
		}
	}

	return "", false
}

func completionFlagSnapshot(cmd *cobra.Command, args []string) *cobra.Command {
	projectDefault := strings.TrimSpace(env.Get("CAST_PROJECT"))
	contextDefault := strings.TrimSpace(env.Get("CAST_CONTEXT"))

	if cmd != nil {
		if value, err := cmd.Flags().GetString("project"); err == nil && strings.TrimSpace(value) != "" {
			projectDefault = strings.TrimSpace(value)
		}
		if value, err := cmd.InheritedFlags().GetString("project"); err == nil && strings.TrimSpace(value) != "" {
			projectDefault = strings.TrimSpace(value)
		}
		if value, err := cmd.Flags().GetString("context"); err == nil && strings.TrimSpace(value) != "" {
			contextDefault = strings.TrimSpace(value)
		}
		if value, err := cmd.InheritedFlags().GetString("context"); err == nil && strings.TrimSpace(value) != "" {
			contextDefault = strings.TrimSpace(value)
		}
	}

	tmp := &cobra.Command{}
	tmp.Flags().StringP("project", "p", projectDefault, "")
	tmp.Flags().StringP("context", "c", contextDefault, "")
	tmp.Flags().StringArrayP("dotenv", "E", []string{}, "")
	tmp.Flags().StringToStringP("env", "e", map[string]string{}, "")
	tmp.FParseErrWhitelist.UnknownFlags = true
	_ = tmp.Flags().Parse(args)

	return tmp
}

func contextNameForCompletion(cmd *cobra.Command, args []string, projectFile string) string {
	tmp := completionFlagSnapshot(cmd, args)

	contextName, _ := tmp.Flags().GetString("context")
	contextName = strings.TrimSpace(contextName)

	if contextName == "" {
		contextName = strings.TrimSpace(parseContextFromArgs(args))
	}

	if contextName == "" {
		contextName = strings.TrimSpace(env.Get("CAST_CONTEXT"))
	}

	if contextName == "" && strings.TrimSpace(projectFile) != "" {
		schema, err := loadProjectSchema(projectFile)
		if err == nil && schema != nil && schema.Config != nil && schema.Config.Context != nil {
			contextName = strings.TrimSpace(*schema.Config.Context)
		}
	}

	if contextName == "" {
		contextName = "default"
	}

	return contextName
}

func contextNameFromWorkspaceAlias(arg, fallback string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(arg, "@"))
	idx := strings.Index(trimmed, ":")
	if idx < 0 {
		return fallback
	}

	value := strings.TrimSpace(trimmed[idx+1:])
	if value == "" {
		return fallback
	}

	return value
}

type completionTarget struct {
	description string
	priority    int
}

func completionTaskTargets(project *projects.Project, contextName, toComplete string) []string {
	if project == nil {
		return nil
	}

	knownContexts := knownCompletionContexts(project, contextName)
	targets := map[string]completionTarget{}

	for _, taskName := range project.Tasks.Keys() {
		task, _ := project.Tasks.Get(taskName)
		description := ""
		if task.Desc != nil {
			description = strings.TrimSpace(*task.Desc)
		}

		name, suffix, contextual := contextualTargetName(taskName, knownContexts)
		if contextual {
			if suffix != contextName {
				continue
			}
			addCompletionTarget(targets, name, description, 2)
			continue
		}

		addCompletionTarget(targets, taskName, description, 1)
	}

	return completionTargetValues(targets, toComplete)
}

func completionJobTargets(project *projects.Project, contextName, toComplete string) []string {
	if project == nil || project.Schema.Jobs == nil {
		return nil
	}

	knownContexts := knownCompletionContexts(project, contextName)
	targets := map[string]completionTarget{}

	for _, jobName := range project.Schema.Jobs.Keys() {
		job, _ := project.Schema.Jobs.Get(jobName)
		description := strings.TrimSpace(job.Desc)
		if description == "" {
			description = strings.TrimSpace(job.Name)
		}

		name, suffix, contextual := contextualTargetName(jobName, knownContexts)
		if contextual {
			if suffix != contextName {
				continue
			}
			addCompletionTarget(targets, name, description, 2)
			continue
		}

		addCompletionTarget(targets, jobName, description, 1)
	}

	return completionTargetValues(targets, toComplete)
}

func knownCompletionContexts(project *projects.Project, contextName string) map[string]struct{} {
	known := map[string]struct{}{}

	if project != nil {
		for _, ctx := range knownProjectContexts(&project.Schema) {
			known[ctx] = struct{}{}
		}
	}

	if strings.TrimSpace(contextName) != "" {
		known[contextName] = struct{}{}
	}

	return known
}

func contextualTargetName(name string, contexts map[string]struct{}) (string, string, bool) {
	idx := strings.LastIndex(name, ":")
	if idx <= 0 || idx >= len(name)-1 {
		return "", "", false
	}

	suffix := name[idx+1:]
	if _, ok := contexts[suffix]; !ok {
		return "", "", false
	}

	base := name[:idx]
	if strings.TrimSpace(base) == "" {
		return "", "", false
	}

	return base, suffix, true
}

func addCompletionTarget(targets map[string]completionTarget, name, description string, priority int) {
	if strings.TrimSpace(name) == "" {
		return
	}

	existing, ok := targets[name]
	if ok && existing.priority > priority {
		return
	}

	targets[name] = completionTarget{description: description, priority: priority}
}

func completionTargetValues(targets map[string]completionTarget, toComplete string) []string {
	if len(targets) == 0 {
		return nil
	}

	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)

	completions := make([]string, 0, len(names))
	for _, name := range names {
		if !strings.HasPrefix(name, toComplete) {
			continue
		}

		target := targets[name]
		if target.description != "" {
			completions = append(completions, name+"\t"+target.description)
			continue
		}
		completions = append(completions, name)
	}

	return completions
}

func knownProjectContexts(schema *types.Project) []string {
	seen := map[string]struct{}{}
	contexts := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || value == "*" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		contexts = append(contexts, value)
	}

	add("default")
	if schema.Config != nil && schema.Config.Context != nil {
		add(*schema.Config.Context)
	}
	if schema.Config != nil {
		for _, ctx := range schema.Config.Contexts {
			add(ctx)
		}
	}
	sort.Strings(contexts)
	return contexts
}
