//go:build !windows

package sshagent

import (
	"fmt"
	"net"
	"os"
	"strings"

	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func Available() bool {
	return strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) != ""
}

func AvailabilityMessage() string {
	return "SSH_AUTH_SOCK is not set"
}

func GophAuth() (goph.Auth, error) {
	method, err := SSHAuthMethod()
	if err != nil {
		return nil, err
	}
	return goph.Auth{method}, nil
}

func SSHAuthMethod() (ssh.AuthMethod, error) {
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if sock == "" {
		return nil, fmt.Errorf(AvailabilityMessage())
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), nil
}
