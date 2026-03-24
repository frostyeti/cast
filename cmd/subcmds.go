package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/cast/internal/types"
	"github.com/spf13/cobra"
)

type subcmdNode struct {
	segment  string
	fullPath string
	children map[string]*subcmdNode
	tasks    map[string]string
}

const dynamicSubcmdAnnotation = "cast.dynamic.subcmd"

func newSubcmdNode(segment, fullPath string) *subcmdNode {
	return &subcmdNode{
		segment:  segment,
		fullPath: fullPath,
		children: map[string]*subcmdNode{},
		tasks:    map[string]string{},
	}
}

func (n *subcmdNode) sortedChildren() []string {
	keys := make([]string, 0, len(n.children))
	for k := range n.children {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (n *subcmdNode) sortedTasks() []string {
	keys := make([]string, 0, len(n.tasks))
	for k := range n.tasks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func registerDynamicSubcommands() {
	projectFile, err := resolveProjectFileFromFlagOrCwd(rootCmd)
	if err != nil || projectFile == "" {
		return
	}
	_ = registerRootDynamicSubcommandsForProjectFile(projectFile)
}

func clearDynamicSubcommands(parent *cobra.Command) {
	for _, c := range parent.Commands() {
		if c.Annotations != nil && c.Annotations[dynamicSubcmdAnnotation] == "true" {
			parent.RemoveCommand(c)
			continue
		}
		clearDynamicSubcommands(c)
	}
}

func buildSubcmdTree(schema *types.Project) *subcmdNode {
	if schema == nil || len(schema.Subcmds) == 0 {
		return nil
	}

	root := newSubcmdNode("", "")
	for _, raw := range schema.Subcmds {
		normalized := normalizeSubcmdPath(raw)
		if normalized == "" {
			continue
		}

		node := root
		prefix := ""
		for _, part := range strings.Split(normalized, ":") {
			if part == "" {
				continue
			}
			if prefix == "" {
				prefix = part
			} else {
				prefix += ":" + part
			}
			next, ok := node.children[part]
			if !ok {
				next = newSubcmdNode(part, prefix)
				node.children[part] = next
			}
			node = next
		}
	}

	if schema.Tasks == nil {
		return root
	}

	for _, taskName := range schema.Tasks.Keys() {
		matched := false
		for _, raw := range schema.Subcmds {
			normalized := normalizeSubcmdPath(raw)
			if normalized == "" {
				continue
			}
			prefix := normalized + ":"
			if !strings.HasPrefix(taskName, prefix) {
				continue
			}

			rest := strings.TrimPrefix(taskName, prefix)
			if rest == "" {
				continue
			}

			node := root
			for _, part := range strings.Split(normalized, ":") {
				if part == "" {
					continue
				}
				next, ok := node.children[part]
				if !ok {
					next = newSubcmdNode(part, part)
					node.children[part] = next
				}
				node = next
			}

			segments := strings.Split(rest, ":")
			for i := 0; i < len(segments)-1; i++ {
				seg := segments[i]
				if seg == "" {
					continue
				}
				next, ok := node.children[seg]
				if !ok {
					fullPath := normalized + ":" + strings.Join(segments[:i+1], ":")
					next = newSubcmdNode(seg, fullPath)
					node.children[seg] = next
				}
				node = next
			}

			leaf := segments[len(segments)-1]
			if leaf == "" {
				continue
			}
			node.tasks[leaf] = taskName
			matched = true
			break
		}

		if matched {
			continue
		}

		segments := strings.Split(taskName, ":")
		if len(segments) < 2 {
			continue
		}

		node := root
		prefix := ""
		for i := 0; i < len(segments)-1; i++ {
			part := segments[i]
			if part == "" {
				continue
			}
			if prefix == "" {
				prefix = part
			} else {
				prefix += ":" + part
			}
			next, ok := node.children[part]
			if !ok {
				next = newSubcmdNode(part, prefix)
				node.children[part] = next
			}
			node = next
		}

		leaf := segments[len(segments)-1]
		if leaf != "" {
			node.tasks[leaf] = taskName
		}
	}

	return root
}

func registerSubcmdNode(parent *cobra.Command, node *subcmdNode, projectFile string) {
	if node == nil || node.segment == "" {
		return
	}

	for _, existing := range parent.Commands() {
		if existing.Name() == node.segment {
			return
		}
	}

	cmd := &cobra.Command{
		Use:                node.segment,
		Short:              fmt.Sprintf("Subcommand for %s tasks", strings.ReplaceAll(node.fullPath, ":", " ")),
		Args:               cobra.ArbitraryArgs,
		DisableFlagParsing: true,
		RunE: func(c *cobra.Command, args []string) error {
			cleanArgs, wantsHelp := sanitizeDynamicArgs(args)

			if len(cleanArgs) == 0 {
				if helpTaskID, ok := node.tasks["help"]; ok {
					return runTaskHelpBlockOrTask(c, projectFile, helpTaskID, nil)
				}
				return c.Help()
			}

			if wantsHelp {
				if taskID, ok := node.tasks[cleanArgs[0]]; ok {
					return runTaskHelpBlockOrTask(c, projectFile, taskID, nil)
				}
				if helpTaskID, ok := node.tasks["help"]; ok {
					return runTaskHelpBlockOrTask(c, projectFile, helpTaskID, nil)
				}
				return c.Help()
			}

			if taskID, ok := node.tasks[cleanArgs[0]]; ok {
				if len(cleanArgs) == 1 {
					for _, nested := range c.Commands() {
						if nested.Name() == cleanArgs[0] {
							return nested.RunE(nested, []string{})
						}
					}
				}
				return runTaskByID(c, projectFile, taskID, cleanArgs[1:])
			}

			for _, nested := range c.Commands() {
				if nested.Name() == cleanArgs[0] {
					return nested.RunE(nested, cleanArgs[1:])
				}
			}

			return c.Help()
		},
	}
	cmd.Annotations = map[string]string{dynamicSubcmdAnnotation: "true"}
	cmd.PersistentFlags().StringP("project", "p", "", "Path to the project file (castfile.yaml)")
	cmd.PersistentFlags().StringP("context", "c", "", "Context name to use from the project")

	cmd.SetHelpCommand(&cobra.Command{Hidden: true})

	for _, child := range node.sortedChildren() {
		registerSubcmdNode(cmd, node.children[child], projectFile)
	}

	for _, leaf := range node.sortedTasks() {
		taskID := node.tasks[leaf]
		leafTaskID := taskID
		leafCmd := &cobra.Command{
			Use:                leaf,
			Short:              leafTaskShort(projectFile, leafTaskID),
			Args:               cobra.ArbitraryArgs,
			DisableFlagParsing: true,
			RunE: func(c *cobra.Command, args []string) error {
				cleanArgs, wantsHelp := sanitizeDynamicArgs(args)
				if len(cleanArgs) == 1 && cleanArgs[0] == "help" {
					return runTaskHelpBlockOrTask(c, projectFile, leafTaskID, nil)
				}
				if wantsHelp {
					return runTaskHelpBlockOrTask(c, projectFile, leafTaskID, nil)
				}
				return runTaskByID(c, projectFile, leafTaskID, cleanArgs)
			},
		}
		leafCmd.Annotations = map[string]string{dynamicSubcmdAnnotation: "true"}
		cmd.AddCommand(leafCmd)
	}

	if helpTaskID, ok := node.tasks["help"]; ok {
		if helpText := taskHelpText(projectFile, helpTaskID); strings.TrimSpace(helpText) != "" {
			cmd.Long = helpText
			cmd.SetHelpCommand(&cobra.Command{
				Use:    "help",
				Hidden: true,
				RunE: func(c *cobra.Command, _ []string) error {
					_, _ = fmt.Fprintln(c.OutOrStdout(), helpText)
					return nil
				},
			})
			cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
				_, _ = fmt.Fprintln(c.OutOrStdout(), helpText)
			})
		}
	}

	parent.AddCommand(cmd)
}

func normalizeSubcmdPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, ":")
	for strings.Contains(path, "::") {
		path = strings.ReplaceAll(path, "::", ":")
	}
	return path
}

func reservedRootCommandNames() map[string]struct{} {
	reserved := map[string]struct{}{}
	for _, c := range rootCmd.Commands() {
		reserved[c.Name()] = struct{}{}
		for _, a := range c.Aliases {
			reserved[a] = struct{}{}
		}
	}
	return reserved
}

func loadProjectSchema(projectFile string) (*types.Project, error) {
	p := &projects.Project{}
	if err := p.LoadFromYaml(projectFile); err != nil {
		return nil, err
	}
	return &p.Schema, nil
}

func taskHelpText(projectFile, taskID string) string {
	schema, err := loadProjectSchema(projectFile)
	if err != nil || schema.Tasks == nil {
		return ""
	}
	if task, ok := schema.Tasks.Get(taskID); ok && task.Help != nil {
		return strings.TrimSpace(*task.Help)
	}
	return ""
}

func leafTaskShort(projectFile, taskID string) string {
	schema, err := loadProjectSchema(projectFile)
	if err != nil || schema.Tasks == nil {
		return ""
	}
	if task, ok := schema.Tasks.Get(taskID); ok && task.Desc != nil {
		return strings.TrimSpace(*task.Desc)
	}
	return ""
}

func runTaskHelpBlockOrTask(cmd *cobra.Command, projectFile, taskID string, args []string) error {
	helpText := taskHelpText(projectFile, taskID)
	if helpText != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), helpText)
		return nil
	}

	descText := leafTaskShort(projectFile, taskID)
	if descText != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), descText)
		return nil
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), taskID)
	return nil
}

func runTaskByID(cmd *cobra.Command, projectFile, taskID string, taskArgs []string) error {
	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return errors.Newf("failed to load project file %s: %w", projectFile, err)
	}

	contextName, _ := cmd.Flags().GetString("context")
	if contextName == "" {
		contextName, _ = cmd.InheritedFlags().GetString("context")
	}
	if contextName == "" {
		if inherited, err := rootCmd.Flags().GetString("context"); err == nil && strings.TrimSpace(inherited) != "" {
			contextName = strings.TrimSpace(inherited)
		} else if parsed := parseContextFromArgs(os.Args[1:]); parsed != "" {
			contextName = parsed
		}
	}
	if contextName == "" {
		contextName = os.Getenv("CAST_CONTEXT")
	}
	if contextName == "" {
		contextName = "default"
	}

	project.ContextName = contextName
	if err := project.Init(); err != nil {
		return errors.Newf("failed to initialize project %s: %w", projectFile, err)
	}

	if shellTaskNeedsCastContext(project, taskID) {
		oldCtx, hadCtx := os.LookupEnv("CAST_CONTEXT")
		_ = os.Setenv("CAST_CONTEXT", contextName)
		defer func() {
			if hadCtx {
				_ = os.Setenv("CAST_CONTEXT", oldCtx)
				return
			}
			_ = os.Unsetenv("CAST_CONTEXT")
		}()
	}

	params := projects.RunTasksParams{
		Targets:     []string{taskID},
		Args:        taskArgs,
		Context:     cmd.Context(),
		ContextName: contextName,
		Stdout:      cmd.OutOrStdout(),
		Stderr:      cmd.ErrOrStderr(),
	}

	results, err := project.RunTask(params)
	if err != nil {
		return errors.Newf("failure with project %s: %w", projectFile, err)
	}

	for _, r := range results {
		if r.Status == runstatus.Error {
			return errors.New("task execution failed")
		}
		if r.Status == runstatus.Cancelled {
			return errors.New("task execution cancelled")
		}
	}

	return nil
}

func shellTaskNeedsCastContext(project *projects.Project, taskID string) bool {
	if project == nil || project.Schema.Tasks == nil {
		return false
	}

	task, ok := project.Schema.Tasks.Get(taskID)
	if !ok {
		return false
	}

	if task.Uses == nil || strings.TrimSpace(*task.Uses) != "shell" {
		return false
	}

	if task.Run == nil || strings.TrimSpace(*task.Run) == "" {
		return false
	}

	parts := strings.Fields(strings.TrimSpace(*task.Run))
	if len(parts) == 0 {
		return false
	}

	if strings.EqualFold(parts[0], "echo") {
		return false
	}

	_, err := exec.LookPath(parts[0])
	return err == nil
}

func registerRootDynamicSubcommandsForProjectFile(projectFile string) error {
	clearDynamicSubcommands(rootCmd)

	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return err
	}

	if len(project.Schema.Subcmds) == 0 || project.Schema.Tasks == nil || project.Schema.Tasks.Len() == 0 {
		return nil
	}

	root := buildSubcmdTree(&project.Schema)
	if root == nil {
		return nil
	}

	reserved := reservedRootCommandNames()
	for _, childName := range root.sortedChildren() {
		node := root.children[childName]
		if _, conflict := reserved[node.segment]; conflict {
			continue
		}
		registerSubcmdNode(rootCmd, node, projectFile)
	}

	return nil
}

func parseContextFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c" || a == "--context":
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
		case strings.HasPrefix(a, "--context="):
			return strings.TrimSpace(strings.TrimPrefix(a, "--context="))
		}
	}
	return ""
}

func sanitizeDynamicArgs(args []string) ([]string, bool) {
	clean := make([]string, 0, len(args))
	wantsHelp := false
	afterDoubleDash := false

	skipValueFlags := map[string]struct{}{
		"-p":        {},
		"--project": {},
		"-c":        {},
		"--context": {},
		"-E":        {},
		"--dotenv":  {},
		"-e":        {},
		"--env":     {},
	}

	for i := 0; i < len(args); i++ {
		a := args[i]

		if afterDoubleDash {
			clean = append(clean, a)
			continue
		}

		if a == "--" {
			afterDoubleDash = true
			continue
		}

		if a == "-h" || a == "--help" {
			wantsHelp = true
			continue
		}

		if a == "help" && len(clean) == 0 {
			wantsHelp = true
			continue
		}

		if _, ok := skipValueFlags[a]; ok {
			if i+1 < len(args) {
				i++
			}
			continue
		}

		if strings.HasPrefix(a, "--project=") || strings.HasPrefix(a, "--context=") || strings.HasPrefix(a, "--dotenv=") || strings.HasPrefix(a, "--env=") {
			continue
		}

		clean = append(clean, a)
	}

	return clean, wantsHelp
}
