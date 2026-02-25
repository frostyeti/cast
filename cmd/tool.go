package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:     "tool",
	Aliases: []string{"tools"},
	Short:   "Manage tools (deno, mise, and others)",
	Long:    `Install and manage tools like deno, mise, or proxy commands to mise.`,
}

var toolInstallCmd = &cobra.Command{
	Use:                "install [tool]",
	Short:              "Install a tool",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, a := range args {
			if a == "-h" || a == "--help" {
				return cmd.Help()
			}
		}

		if len(args) == 0 {
			// Fallback to empty install which installs from mise.toml
			return runMiseCmd([]string{"install"})
		}

		// Find the tool name (first non-flag argument)
		tool := ""
		for _, a := range args {
			if len(a) > 0 && a[0] != '-' {
				tool = a
				break
			}
		}

		switch tool {
		case "deno":
			return installDeno()
		case "mise":
			return installMise()
		default:
			// Fallback to mise install
			return runMiseCmd(append([]string{"install"}, args...))
		}
	},
}

var toolWhereCmd = &cobra.Command{
	Use:                "where [tool]",
	Short:              "Proxy 'where' command to mise",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, a := range args {
			if a == "-h" || a == "--help" {
				return cmd.Help()
			}
		}
		if len(args) == 0 {
			return cmd.Help()
		}
		return runMiseCmd(append([]string{"where"}, args...))
	},
}

var toolUseCmd = &cobra.Command{
	Use:                "use [tool]",
	Short:              "Proxy 'use' command to mise",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, a := range args {
			if a == "-h" || a == "--help" {
				return cmd.Help()
			}
		}
		if len(args) == 0 {
			// mise use without args runs interactive selector
			return runMiseCmd([]string{"use"})
		}
		return runMiseCmd(append([]string{"use"}, args...))
	},
}

func init() {
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolInstallCmd)
	toolCmd.AddCommand(toolWhereCmd)
	toolCmd.AddCommand(toolUseCmd)

	// Flags for `tool install`
	toolInstallCmd.Flags().BoolP("force", "f", false, "Force reinstall even if already installed")
	toolInstallCmd.Flags().StringP("jobs", "j", "4", "Number of jobs to run in parallel")
	toolInstallCmd.Flags().BoolP("dry-run", "n", false, "Show what would be installed without actually installing")
	toolInstallCmd.Flags().BoolP("verbose", "v", false, "Show installation output")
	toolInstallCmd.Flags().String("before", "", "Only install versions released before this date")
	toolInstallCmd.Flags().Bool("raw", false, "Directly pipe stdin/stdout/stderr from plugin to user")
	toolInstallCmd.Flags().StringP("cd", "C", "", "Change directory before running command")
	toolInstallCmd.Flags().StringP("env", "E", "", "Set the environment for loading `mise.<ENV>.toml`")
	toolInstallCmd.Flags().BoolP("quiet", "q", false, "Suppress non-error messages")
	toolInstallCmd.Flags().BoolP("yes", "y", false, "Answer yes to all confirmation prompts")
	toolInstallCmd.Flags().Bool("locked", false, "Require lockfile URLs to be present during installation")
	toolInstallCmd.Flags().Bool("silent", false, "Suppress all task output and mise non-error messages")

	// Flags for `tool where`
	toolWhereCmd.Flags().StringP("cd", "C", "", "Change directory before running command")
	toolWhereCmd.Flags().StringP("env", "E", "", "Set the environment for loading `mise.<ENV>.toml`")
	toolWhereCmd.Flags().StringP("jobs", "j", "8", "How many jobs to run in parallel")
	toolWhereCmd.Flags().BoolP("quiet", "q", false, "Suppress non-error messages")
	toolWhereCmd.Flags().BoolP("verbose", "v", false, "Show extra output")
	toolWhereCmd.Flags().BoolP("yes", "y", false, "Answer yes to all confirmation prompts")
	toolWhereCmd.Flags().Bool("raw", false, "Read/write directly to stdin/stdout/stderr instead of by line")
	toolWhereCmd.Flags().Bool("locked", false, "Require lockfile URLs to be present during installation")
	toolWhereCmd.Flags().Bool("silent", false, "Suppress all task output and mise non-error messages")

	// Flags for `tool use`
	toolUseCmd.Flags().StringP("env", "e", "", "Create/modify an environment-specific config file like .mise.<env>.toml")
	toolUseCmd.Flags().BoolP("force", "f", false, "Force reinstall even if already installed")
	toolUseCmd.Flags().BoolP("global", "g", false, "Use the global config file instead of the local one")
	toolUseCmd.Flags().StringP("jobs", "j", "4", "Number of jobs to run in parallel")
	toolUseCmd.Flags().BoolP("dry-run", "n", false, "Perform a dry run, showing what would be installed and modified")
	toolUseCmd.Flags().StringP("path", "p", "", "Specify a path to a config file or directory")
	toolUseCmd.Flags().String("before", "", "Only install versions released before this date")
	toolUseCmd.Flags().Bool("fuzzy", false, "Save fuzzy version to config file")
	toolUseCmd.Flags().Bool("pin", false, "Save exact version to config file")
	toolUseCmd.Flags().Bool("raw", false, "Directly pipe stdin/stdout/stderr from plugin to user")
	toolUseCmd.Flags().String("remove", "", "Remove the plugin(s) from config file")
	toolUseCmd.Flags().StringP("cd", "C", "", "Change directory before running command")
	toolUseCmd.Flags().BoolP("quiet", "q", false, "Suppress non-error messages")
	toolUseCmd.Flags().BoolP("verbose", "v", false, "Show extra output")
	toolUseCmd.Flags().BoolP("yes", "y", false, "Answer yes to all confirmation prompts")
	toolUseCmd.Flags().Bool("locked", false, "Require lockfile URLs to be present during installation")
	toolUseCmd.Flags().Bool("silent", false, "Suppress all task output and mise non-error messages")
}

func installDeno() error {
	if _, err := exec.LookPath("deno"); err == nil {
		fmt.Println("deno is already installed and accessible in $PATH.")
		return nil
	}

	fmt.Println("Installing deno...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-c", "irm https://deno.land/install.ps1 | iex")
	} else {
		cmd = exec.Command("sh", "-c", "curl -fsSL https://deno.land/x/install/install.sh | sh")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installMise() error {
	if _, err := exec.LookPath("mise"); err == nil {
		fmt.Println("mise is already installed and accessible in $PATH.")
		return nil
	}

	fmt.Println("Installing mise...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-c", "irm https://mise.run | iex") // Just in case mise supports it this way
	} else {
		cmd = exec.Command("sh", "-c", "curl https://mise.run | sh")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		fmt.Println("\nmise installed successfully. Please ensure you add `mise activate` into your shell configuration.")
	}
	return err
}

func runMiseCmd(args []string) error {
	if _, err := exec.LookPath("mise"); err != nil {
		fmt.Println("mise is not installed or not in $PATH. Attempting to install mise first...")
		if err := installMise(); err != nil {
			return fmt.Errorf("failed to install mise: %w", err)
		}
	}

	cmd := exec.Command("mise", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
