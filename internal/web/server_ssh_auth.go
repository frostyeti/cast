package web

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/frostyeti/cast/internal/paths"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type webSSHHostConfig struct {
	Host         string
	Port         uint
	User         string
	IdentityFile string
	Password     string
	ForceAgent   bool
}

func resolveWebSSHHostConfig(cfg webSSHHostConfig) (webSSHHostConfig, error) {
	resolved := cfg
	if strings.TrimSpace(resolved.IdentityFile) != "" {
		identity, err := paths.Resolve(strings.TrimSpace(resolved.IdentityFile))
		if err != nil {
			return resolved, fmt.Errorf("failed to resolve ssh identity path %q: %w", resolved.IdentityFile, err)
		}
		resolved.IdentityFile = identity
	}
	if strings.TrimSpace(resolved.Password) == "" {
		for _, key := range []string{"CAST_SSH_PASS", "SSH_PASS"} {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" {
				resolved.Password = value
				break
			}
		}
	}
	if strings.TrimSpace(resolved.User) == "" {
		resolved.User = os.Getenv("USER")
	}
	if resolved.Port == 0 {
		resolved.Port = 22
	}
	return resolved, nil
}

func buildWebSSHAuthMethods(cfg webSSHHostConfig) ([]ssh.AuthMethod, error) {
	if cfg.ForceAgent {
		method, err := webSSHAgentMethod()
		if err != nil {
			return nil, fmt.Errorf("ssh agent was required for %s but is not available: %w", cfg.Host, err)
		}
		return []ssh.AuthMethod{method}, nil
	}

	methods := []ssh.AuthMethod{}
	if strings.TrimSpace(cfg.Password) != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}
	if strings.TrimSpace(cfg.IdentityFile) != "" {
		key, err := os.ReadFile(cfg.IdentityFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read ssh identity %s: %w", cfg.IdentityFile, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ssh identity %s: %w", cfg.IdentityFile, err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if method, err := webSSHAgentMethod(); err == nil {
		methods = append(methods, method)
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("no ssh authentication methods available for %s; set host.identity, host.password, host.agent, CAST_SSH_PASS, or SSH_PASS", cfg.Host)
	}
	return methods, nil
}

func webSSHAgentMethod() (ssh.AuthMethod, error) {
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if sock == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK is not set")
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), nil
}
