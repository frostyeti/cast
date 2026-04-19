//go:build windows

package sshagent

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Microsoft/go-winio"
	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const openSSHPipe = `\\.\pipe\openssh-ssh-agent`

func Available() bool {
	if strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) != "" {
		return true
	}
	conn, err := winio.DialPipe(openSSHPipe, nil)
	if err == nil {
		_ = conn.Close()
		return true
	}
	return false
}

func AvailabilityMessage() string {
	return "neither SSH_AUTH_SOCK nor the Windows OpenSSH agent pipe \\.\\pipe\\openssh-ssh-agent is available"
}

func GophAuth() (goph.Auth, error) {
	method, err := SSHAuthMethod()
	if err != nil {
		return nil, err
	}
	return goph.Auth{method}, nil
}

func SSHAuthMethod() (ssh.AuthMethod, error) {
	conn, err := dialAgentConn()
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers), nil
}

func dialAgentConn() (net.Conn, error) {
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			return conn, nil
		}
	}

	conn, err := winio.DialPipe(openSSHPipe, nil)
	if err != nil {
		return nil, fmt.Errorf(AvailabilityMessage()+": %w", err)
	}
	return conn, nil
}
