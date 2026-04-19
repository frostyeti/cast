package cmd

import (
	"fmt"
	"os"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"github.com/spf13/cobra"
)

var scpCmd = &cobra.Command{
	Use:   "scp <src> <dest>",
	Short: "Copy files to/from inventory hosts",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		dest := args[1]

		targets, _ := cmd.Flags().GetStringSlice("targets")
		if len(targets) == 0 {
			return errors.New("must specify at least one target host via --targets")
		}

		projectFile := ""
		tryFiles := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml"}
		for _, f := range tryFiles {
			if _, err := os.Stat(f); err == nil {
				projectFile = f
				break
			}
		}
		if projectFile == "" {
			return errors.New("no castfile found in current directory")
		}

		project := &projects.Project{}
		err := project.LoadFromYaml(projectFile)
		if err != nil {
			return errors.Newf("failed to load project file %s: %w", projectFile, err)
		}

		if project.Schema.Inventory == nil {
			return errors.New("inventory not found in castfile")
		}

		for _, hostAlias := range targets {
			fmt.Printf("Copying to/from %s...\n", hostAlias)
			var targetHost *types.HostInfo
			for k, h := range project.Schema.Inventory.Hosts {
				if k == hostAlias || h.Host == hostAlias {
					targetHost = &types.HostInfo{}
					*targetHost = h
					break
				}
			}

			if targetHost == nil {
				fmt.Printf("Warning: host not found in inventory: %s\n", hostAlias)
				continue
			}

			port := uint(22)
			if targetHost.Port != nil {
				port = *targetHost.Port
			}

			user := os.Getenv("USER")
			if targetHost.User != nil {
				user = *targetHost.User
			}

			identity := ""
			if targetHost.IdentityFile != nil {
				identity = *targetHost.IdentityFile
			}
			password := ""
			if targetHost.Password != nil {
				password = *targetHost.Password
			}
			useAgent := false
			if targetHost.Agent != nil {
				useAgent = *targetHost.Agent
			}

			authCfg, err := projects.ResolveRootSSHAuthConfig(project, projects.HostInfo{
				Host:         targetHost.Host,
				Port:         port,
				User:         user,
				Password:     password,
				IdentityFile: identity,
				Agent:        useAgent,
			}, "")
			if err != nil {
				fmt.Printf("Warning: %v\n", err)
				continue
			}

			client, _, err := projects.NewRootSSHClient(authCfg)
			if err != nil {
				fmt.Printf("Warning: %v\n", err)
				continue
			}

			// Simple heuristic: if src starts with remote host syntax, it's a pull, otherwise push.
			// But for simplicity of this CLI which asks for --targets, let's assume it's always local -> remote
			// unless we explicitly want a pull flag. The instructions say "push or pull files".
			// A common implementation for `scp <src> <dest> --targets host1,host2` is local -> remote.
			// If we want to support pulling, maybe a --pull flag is needed, since pulling from multiple hosts to the same dest would overwrite it.

			pull, _ := cmd.Flags().GetBool("pull")
			if pull {
				err = client.Download(src, dest)
			} else {
				err = client.Upload(src, dest)
			}

			_ = client.Close()

			if err != nil {
				fmt.Printf("Failed for %s: %v\n", hostAlias, err)
			} else {
				fmt.Printf("Success for %s\n", hostAlias)
			}
		}

		return nil
	},
}

func init() {
	scpCmd.Flags().StringSliceP("targets", "t", []string{}, "Comma-separated list of target hosts")
	scpCmd.Flags().Bool("pull", false, "Pull file from remote to local (default is local to remote)")
	rootCmd.AddCommand(scpCmd)
}
