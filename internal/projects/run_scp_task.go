package projects

import (
	"context"
	"io"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

func runScpTask(ctx TaskContext) *TaskResult {

	//https://github.com/melbahja/goph

	res := NewTaskResult()
	uses := ctx.Task.Uses
	if uses != "scp" {
		return res.Fail(errors.New("Invalid uses for SCP task: " + uses))
	}

	files := []string{}
	if v, ok := ctx.Task.With["files"]; ok {
		if arr, ok := v.([]interface{}); ok {
			for _, item := range arr {
				if str, ok := item.(string); ok {
					files = append(files, str)
				}
			}
		}
	}

	if len(files) == 0 {
		return res.Fail(errors.New("No files specified for SCP task"))
	}

	uri, err := url.Parse(ctx.Task.Uses)
	if err != nil {
		return res.Fail(errors.New("Invalid SSH URI: " + err.Error()))
	}

	if uri.Scheme != "scp" {
		return res.Fail(errors.New("Invalid SSH URI scheme: " + uri.Scheme))
	}

	direction := uri.Query().Get("direction")
	if direction == "" {
		direction = uri.Path
	}

	download := uri.Query().Get("download") == "true"
	if direction == "" {
		if download {
			direction = "download"
		} else {
			direction = "upload"
		}
	}

	targets := ctx.Task.Hosts

	if len(targets) == 0 {
		return res.Fail(errors.New("No targets found for SSH task"))
	}

	for _, target := range targets {
		if err := runScpTarget(ctx.Context, direction, ctx, target, files); err != nil {
			return res.Fail(err)
		}
	}

	res.End()

	// Placeholder for running an SSH command
	// This would typically involve executing the command over SSH
	return res.Ok()
}

func runScpTarget(ctx context.Context, direction string, taskContext TaskContext, target HostInfo, files []string) error {
	var auth goph.Auth
	var err error
	identity := ""
	password := ""
	if target.IdentityFile != "" {
		identity = target.IdentityFile
	}
	if target.Password != "" {
		password = target.Password
		if password != "" {
			p, ok := taskContext.Task.Env[password]
			if ok {
				password = p
			}
		}
	}

	if identity == "" && password != "" {
		auth = goph.Password(password)
	} else if goph.HasAgent() {
		auth, err = goph.UseAgent()
	} else if identity != "" {
		auth, err = goph.Key(identity, password)
	} else {
		return errors.New("No authentication method provided for SSH task")
	}

	if err != nil {
		return errors.New("Failed to create SSH authentication: " + err.Error())
	}

	port := uint(22)
	if target.Port > 0 {
		port = target.Port
	}
	user := ""
	if target.User != "" {
		user = target.User
	}

	client, err := goph.NewConn(&goph.Config{
		User: user,
		Addr: target.Host,
		Port: port,
		Auth: auth,
		Callback: func(host string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	})

	if err != nil {
		err2 := errors.New("Failed to connect to SSH target " + target.Host + ": " + err.Error())
		err2 = errors.WithCause(err2, err)
		return err2
	}

	defer client.Close()

	for _, file := range files {
		parts := strings.Split(file, ":")
		if len(parts) != 2 {
			err2 := errors.New("Invalid SCP file format, expected 'source:destination'")
			err2 = errors.WithCause(err2, err)
			return err2
		}
		source := parts[0]
		destination := parts[1]

		if direction != "download" {
			os.Stdout.WriteString("Uploading " + source + " to " + destination + " on " + target.Host + "\n")
			err = Upload(ctx, client, source, destination)
		} else {
			os.Stdout.WriteString("Downloading " + source + " to " + destination + " from " + target.Host + "\n")
			err = Download(ctx, client, destination, source)
		}

		if err != nil {
			err2 := errors.New("Failed to transfer file " + source + " to " + destination + ": " + err.Error())
			err2 = errors.WithCause(err2, err)
			return err2
		}
	}

	return nil
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

func Upload(ctx context.Context, c *goph.Client, localPath string, remotePath string) (err error) {

	local, err := os.Open(localPath)
	if err != nil {
		return
	}
	defer local.Close()

	ftp, err := c.NewSftp()
	if err != nil {
		return
	}
	defer ftp.Close()

	remote, err := ftp.Create(remotePath)
	if err != nil {
		return
	}
	defer remote.Close()

	_, err = io.Copy(remote, readerFunc(func(p []byte) (int, error) {

		// golang non-blocking channel: https://gobyexample.com/non-blocking-channel-operations
		select {

		// if context has been canceled
		case <-ctx.Done():
			// stop process and propagate "context canceled" error
			return 0, ctx.Err()
		default:
			// otherwise just run default io.Reader implementation
			return local.Read(p)
		}
	}))
	return
}

// Download file from remote server!
func Download(ctx context.Context, c *goph.Client, remotePath string, localPath string) (err error) {

	local, err := os.Create(localPath)
	if err != nil {
		return
	}
	defer local.Close()

	ftp, err := c.NewSftp()
	if err != nil {
		return
	}
	defer ftp.Close()

	remote, err := ftp.Open(remotePath)
	if err != nil {
		return
	}
	defer remote.Close()

	_, err = io.Copy(local, readerFunc(func(p []byte) (int, error) {

		// golang non-blocking channel: https://gobyexample.com/non-blocking-channel-operations
		select {

		// if context has been canceled
		case <-ctx.Done():
			// stop process and propagate "context canceled" error
			return 0, ctx.Err()
		default:
			// otherwise just run default io.Reader implementation
			return remote.Read(p)
		}
	}))
	if err != nil {
		return err
	}

	return local.Sync()
}
