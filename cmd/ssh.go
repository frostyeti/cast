package cmd

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"text/template"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"github.com/melbahja/goph"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var sshCmd = &cobra.Command{
	Use:   "ssh <host>",
	Short: "Open an interactive SSH session to an inventory host",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostAlias := args[0]

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

		var targetHost *types.HostInfo
		for k, h := range project.Schema.Inventory.Hosts {
			if k == hostAlias || h.Host == hostAlias {
				// We need to copy it
				targetHost = &types.HostInfo{}
				*targetHost = h
				break
			}
		}

		if targetHost == nil {
			return errors.New("host not found in inventory: " + hostAlias)
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
			return errors.New("no authentication method provided for SSH target")
		}

		if err != nil {
			return errors.New("failed to create SSH authentication: " + err.Error())
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
				return nil // InsecureIgnoreHostKey
			},
		})

		if err != nil {
			return errors.New("failed to connect to SSH target " + targetHost.Host + ": " + err.Error())
		}
		defer client.Close()

		sess, err := client.NewSession()
		if err != nil {
			return errors.New("failed to create SSH session: " + err.Error())
		}
		defer sess.Close()

		scriptPath, _ := cmd.Flags().GetString("script")
		useTemplate, _ := cmd.Flags().GetBool("template")

		if scriptPath != "" {
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				return fmt.Errorf("failed to read script: %w", err)
			}

			scriptBody := string(content)
			if useTemplate {
				tmpl, err := template.New("script").Parse(scriptBody)
				if err != nil {
					return fmt.Errorf("failed to parse template: %w", err)
				}
				var buf bytes.Buffer
				// we pass environment variables or something to the template
				// let's pass a basic context for now
				if err := tmpl.Execute(&buf, map[string]interface{}{
					"Host": targetHost,
				}); err != nil {
					return fmt.Errorf("failed to execute template: %w", err)
				}
				scriptBody = buf.String()
			}

			sess.Stdout = os.Stdout
			sess.Stderr = os.Stderr
			return sess.Run(scriptBody)
		}

		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,     // enable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		// Request pseudo terminal
		fd := int(os.Stdin.Fd())
		state, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("terminal make raw: %s", err)
		}
		defer term.Restore(fd, state)

		termWidth, termHeight, err := term.GetSize(fd)
		if err != nil {
			termWidth, termHeight = 80, 24
		}

		if err := sess.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
			return fmt.Errorf("request for pseudo terminal failed: %s", err)
		}

		sess.Stdin = os.Stdin
		sess.Stdout = os.Stdout
		sess.Stderr = os.Stderr

		if err := sess.Shell(); err != nil {
			return fmt.Errorf("failed to start shell: %s", err)
		}

		if err := sess.Wait(); err != nil {
			if e, ok := err.(*ssh.ExitError); ok {
				return errors.New(fmt.Sprintf("exit status: %d", e.ExitStatus()))
			}
			return fmt.Errorf("failed to exit shell cleanly: %s", err)
		}

		return nil
	},
}

func init() {
	sshCmd.Flags().StringP("script", "s", "", "Local script file to execute on the target")
	sshCmd.Flags().BoolP("template", "t", false, "Parse the script as a Go template before execution")
	rootCmd.AddCommand(sshCmd)
}
