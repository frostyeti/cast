package projects

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/eval"
	"github.com/frostyeti/cast/internal/paths"
	"github.com/frostyeti/go/env"
	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

type sshAuthConfig struct {
	Host           string
	Port           uint
	User           string
	IdentityFile   string
	Password       string
	UseAgent       bool
	ForceAgent     bool
	Substitution   bool
	Scope          map[string]any
	EnvGet         func(string) string
	ErrorPrefix    string
	ExpandIdentity bool
	ExpandPassword bool
}

func resolveSSHAuthConfig(cfg sshAuthConfig) (sshAuthConfig, error) {
	resolved := cfg

	if resolved.ExpandIdentity {
		identity, err := resolveSSHSecretLikeValue(resolved.IdentityFile, resolved.Scope, resolved.EnvGet, resolved.Substitution, true)
		if err != nil {
			return resolved, errors.Newf("%sfailed to resolve identity path: %w", resolved.ErrorPrefix, err)
		}
		resolved.IdentityFile = identity
	}

	if resolved.ExpandPassword {
		password, err := resolveSSHSecretLikeValue(resolved.Password, resolved.Scope, resolved.EnvGet, resolved.Substitution, false)
		if err != nil {
			return resolved, errors.Newf("%sfailed to resolve password: %w", resolved.ErrorPrefix, err)
		}
		resolved.Password = password
	}

	if strings.TrimSpace(resolved.Password) == "" {
		for _, key := range []string{"CAST_SSH_PASS", "SSH_PASS"} {
			value := ""
			if resolved.EnvGet != nil {
				value = resolved.EnvGet(key)
			}
			if value == "" {
				value = os.Getenv(key)
			}
			if strings.TrimSpace(value) == "" {
				continue
			}
			expanded, err := resolveSSHSecretLikeValue(value, resolved.Scope, resolved.EnvGet, resolved.Substitution, false)
			if err != nil {
				return resolved, errors.Newf("%sfailed to resolve %s: %w", resolved.ErrorPrefix, key, err)
			}
			resolved.Password = expanded
			break
		}
	}

	return resolved, nil
}

func resolveSSHSecretLikeValue(value string, scope map[string]any, envGet func(string) string, substitution bool, isPath bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	if scope != nil && strings.ContainsRune(value, '{') {
		resolved, err := eval.EvalAsString(value, scope)
		if err != nil {
			return "", err
		}
		value = resolved
	}

	if strings.ContainsRune(value, '$') {
		resolved, err := env.Expand(value, env.WithGet(func(key string) string {
			if envGet != nil {
				if v := envGet(key); v != "" {
					return v
				}
			}
			return os.Getenv(key)
		}), env.WithCommandSubstitution(substitution))
		if err != nil {
			return "", err
		}
		value = resolved
	}

	if isPath && value != "" {
		resolved, err := paths.Resolve(value)
		if err != nil {
			return "", err
		}
		value = filepath.Clean(resolved)
	}

	return value, nil
}

func createSSHAuth(cfg sshAuthConfig) (goph.Auth, string, error) {
	hasAgent := goph.HasAgent()
	identity := strings.TrimSpace(cfg.IdentityFile)
	password := cfg.Password

	if cfg.ForceAgent {
		if !hasAgent {
			return nil, "", errors.Newf("%sssh agent was required for %s but SSH_AUTH_SOCK is not available; start an agent, load a key, or disable host.agent", cfg.ErrorPrefix, cfg.Host)
		}
		auth, err := goph.UseAgent()
		if err != nil {
			return nil, "", errors.Newf("%sfailed to use ssh agent for %s: %w", cfg.ErrorPrefix, cfg.Host, err)
		}
		return auth, "agent", nil
	}

	var attempts []struct {
		name string
		fn   func() (goph.Auth, error)
	}

	if cfg.UseAgent && hasAgent {
		attempts = append(attempts, struct {
			name string
			fn   func() (goph.Auth, error)
		}{name: "agent", fn: goph.UseAgent})
	}

	if identity != "" {
		attempts = append(attempts, struct {
			name string
			fn   func() (goph.Auth, error)
		}{name: "identity", fn: func() (goph.Auth, error) {
			return goph.Key(identity, password)
		}})
	}

	if password != "" {
		attempts = append(attempts, struct {
			name string
			fn   func() (goph.Auth, error)
		}{name: "password", fn: func() (goph.Auth, error) {
			return goph.Password(password), nil
		}})
	}

	if !cfg.UseAgent && hasAgent {
		attempts = append(attempts, struct {
			name string
			fn   func() (goph.Auth, error)
		}{name: "agent", fn: goph.UseAgent})
	}

	if len(attempts) == 0 {
		details := []string{}
		if identity != "" {
			details = append(details, fmt.Sprintf("identity=%s", identity))
		}
		if password != "" {
			details = append(details, "password=provided")
		}
		if hasAgent {
			details = append(details, "agent=available")
		} else {
			details = append(details, "agent=unavailable")
		}
		return nil, "", errors.Newf("%sno SSH authentication method was available for %s (%s); set host.identity, host.password, host.agent, CAST_SSH_PASS, or SSH_PASS", cfg.ErrorPrefix, cfg.Host, strings.Join(details, ", "))
	}

	var attemptErrors []string
	for _, attempt := range attempts {
		auth, err := attempt.fn()
		if err == nil {
			return auth, attempt.name, nil
		}
		attemptErrors = append(attemptErrors, attempt.name+": "+err.Error())
	}

	return nil, "", errors.Newf("%sfailed to prepare SSH authentication for %s; tried %s", cfg.ErrorPrefix, cfg.Host, strings.Join(attemptErrors, "; "))
}

func newSSHClient(cfg sshAuthConfig) (*goph.Client, string, error) {
	auth, authSource, err := createSSHAuth(cfg)
	if err != nil {
		return nil, "", err
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}

	client, err := goph.NewConn(&goph.Config{
		User: cfg.User,
		Addr: cfg.Host,
		Port: port,
		Auth: auth,
		Callback: func(host string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	})
	if err != nil {
		return nil, authSource, formatSSHConnectError(cfg, authSource, err)
	}

	return client, authSource, nil
}

func formatSSHConnectError(cfg sshAuthConfig, authSource string, err error) error {
	parts := []string{fmt.Sprintf("host=%s", cfg.Host), fmt.Sprintf("port=%d", cfg.Port)}
	if strings.TrimSpace(cfg.User) != "" {
		parts = append(parts, fmt.Sprintf("user=%s", cfg.User))
	}
	if authSource != "" {
		parts = append(parts, fmt.Sprintf("auth=%s", authSource))
	}
	if strings.TrimSpace(cfg.IdentityFile) != "" {
		parts = append(parts, fmt.Sprintf("identity=%s", cfg.IdentityFile))
	}

	hint := ""
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unable to authenticate"), strings.Contains(msg, "permission denied"):
		hint = "check the username, identity file, agent keys, or password value"
	case strings.Contains(msg, "no such file"):
		hint = "check that the identity path exists after ~ and variable expansion"
	case strings.Contains(msg, "connection refused"):
		hint = "check that the SSH server is running and the port is correct"
	case strings.Contains(msg, "i/o timeout"), strings.Contains(msg, "operation timed out"):
		hint = "check network reachability, firewall rules, and the target host/port"
	case strings.Contains(msg, "unknown host"), strings.Contains(msg, "no such host"):
		hint = "check the host name or DNS resolution"
	}

	if hint != "" {
		return errors.Newf("%sfailed to connect to SSH target %s (%s): %s; %s", cfg.ErrorPrefix, cfg.Host, strings.Join(parts, ", "), msg, hint)
	}

	return errors.Newf("%sfailed to connect to SSH target %s (%s): %s", cfg.ErrorPrefix, cfg.Host, strings.Join(parts, ", "), msg)
}

func projectSSHAuthConfig(p *Project, host HostInfo, prefix string) (sshAuthConfig, error) {
	substitution := true
	if p != nil && p.Schema.Config != nil && p.Schema.Config.Substitution != nil {
		substitution = *p.Schema.Config.Substitution
	}

	envGet := os.Getenv
	var scope map[string]any
	if p != nil {
		if p.Env != nil {
			envGet = p.Env.Get
		}
		if p.Scope != nil {
			scope = p.Scope.ToMap()
		}
	}

	resolved, err := resolveSSHAuthConfig(sshAuthConfig{
		Host:           host.Host,
		Port:           host.Port,
		User:           host.User,
		IdentityFile:   host.IdentityFile,
		Password:       host.Password,
		UseAgent:       host.Agent,
		ForceAgent:     host.Agent,
		Substitution:   substitution,
		Scope:          scope,
		EnvGet:         envGet,
		ErrorPrefix:    prefix,
		ExpandIdentity: true,
		ExpandPassword: true,
	})
	if err != nil {
		return sshAuthConfig{}, err
	}

	if strings.TrimSpace(resolved.User) == "" {
		resolved.User = os.Getenv("USER")
	}

	return resolved, nil
}

// ResolveRootSSHAuthConfig resolves SSH auth settings for root CLI commands.
func ResolveRootSSHAuthConfig(p *Project, host HostInfo, prefix string) (sshAuthConfig, error) {
	return projectSSHAuthConfig(p, host, prefix)
}

// NewRootSSHClient creates an SSH client for root CLI commands.
func NewRootSSHClient(cfg sshAuthConfig) (*goph.Client, string, error) {
	return newSSHClient(cfg)
}
