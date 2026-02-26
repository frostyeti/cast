package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"github.com/melbahja/goph"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
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

			var auth goph.Auth
			identity := ""
			password := ""

			if targetHost.IdentityFile != nil {
				identity = *targetHost.IdentityFile
			}
			if targetHost.Password != nil {
				password = *targetHost.Password
			}

			if identity == "" && password != "" {
				auth = goph.Password(password)
			} else if goph.HasAgent() {
				auth, err = goph.UseAgent()
			} else if identity != "" {
				auth, err = goph.Key(identity, password)
			} else {
				fmt.Printf("Warning: no authentication method provided for SSH target %s\n", hostAlias)
				continue
			}

			if err != nil {
				fmt.Printf("Warning: failed to create SSH authentication for %s: %v\n", hostAlias, err)
				continue
			}

			port := 22
			if targetHost.Port != nil {
				port = int(*targetHost.Port)
			}

			user := os.Getenv("USER")
			if targetHost.User != nil {
				user = *targetHost.User
			}

			client, err := goph.NewConn(&goph.Config{
				User: user,
				Addr: targetHost.Host,
				Port: uint(port),
				Auth: auth,
				Callback: func(host string, remote net.Addr, key ssh.PublicKey) error {
					return nil
				},
			})

			if err != nil {
				fmt.Printf("Warning: failed to connect to SSH target %s: %v\n", targetHost.Host, err)
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

			client.Close()

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
