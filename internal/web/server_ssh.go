package web

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/frostyeti/cast/internal/types"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func (s *Server) handleSSHStream(w http.ResponseWriter, r *http.Request) {
	projId := r.PathValue("id")
	hostAlias := r.PathValue("host_alias")

	proj, ok := s.projects[projId]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	if proj.Schema.Inventory == nil {
		http.Error(w, "inventory not found", http.StatusNotFound)
		return
	}

	var targetHost *types.HostInfo
	for alias, h := range proj.Schema.Inventory.Hosts {
		if alias == hostAlias || h.Host == hostAlias {
			targetHost = &h
			break
		}
	}

	if targetHost == nil {
		http.Error(w, "host not found in inventory", http.StatusNotFound)
		return
	}

	// Determine connection parameters
	host := targetHost.Host
	port := uint(22)
	if targetHost.Port != nil {
		port = *targetHost.Port
	}
	user := os.Getenv("USER")
	if targetHost.User != nil {
		user = *targetHost.User
	}

	var authMethods []ssh.AuthMethod

	if targetHost.Password != nil {
		authMethods = append(authMethods, ssh.Password(*targetHost.Password))
	} else if targetHost.IdentityFile != nil {
		key, err := os.ReadFile(*targetHost.IdentityFile)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			} else {
				log.Printf("Failed to parse private key: %v", err)
			}
		} else {
			log.Printf("Failed to read private key: %v", err)
		}
	} else {
		// Default to ~/.ssh/id_rsa or id_ed25519
		homeDir, _ := os.UserHomeDir()
		candidates := []string{"id_ed25519", "id_rsa"}
		for _, c := range candidates {
			keyPath := filepath.Join(homeDir, ".ssh", c)
			if key, err := os.ReadFile(keyPath); err == nil {
				if signer, err := ssh.ParsePrivateKey(key); err == nil {
					authMethods = append(authMethods, ssh.PublicKeys(signer))
				}
			}
		}
	}

	if len(authMethods) == 0 {
		http.Error(w, "no authentication methods available", http.StatusUnauthorized)
		return
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to ssh: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create ssh session: %v", err), http.StatusInternalServerError)
		return
	}
	defer session.Close()

	// Request PTY
	if err := session.RequestPty("xterm", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to request pty: %v", err), http.StatusInternalServerError)
		return
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get stdin: %v", err), http.StatusInternalServerError)
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get stdout: %v", err), http.StatusInternalServerError)
		return
	}

	// stderr is merged with stdout in PTY typically, but just in case
	stderr, err := session.StderrPipe()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get stderr: %v", err), http.StatusInternalServerError)
		return
	}

	if err := session.Shell(); err != nil {
		http.Error(w, fmt.Sprintf("failed to start shell: %v", err), http.StatusInternalServerError)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade to websocket: %v", err)
		return
	}
	defer ws.Close()

	// Bridge WS to SSH Stdin
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			stdin.Write(msg)
		}
	}()

	// Bridge SSH Stdout/Stderr to WS
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				ws.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				ws.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for session to finish
	session.Wait()
}
