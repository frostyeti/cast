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
	Use:   "install [tool]",
	Short: "Install a tool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

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
	Use:   "where [tool]",
	Short: "Proxy 'where' command to mise",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMiseCmd(append([]string{"where"}, args...))
	},
}

var toolUseCmd = &cobra.Command{
	Use:   "use [tool]",
	Short: "Proxy 'use' command to mise",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMiseCmd(append([]string{"use"}, args...))
	},
}

func init() {
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolInstallCmd)
	toolCmd.AddCommand(toolWhereCmd)
	toolCmd.AddCommand(toolUseCmd)
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
