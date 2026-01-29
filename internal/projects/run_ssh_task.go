package projects

import (
	"context"
	"net"
	"os"

	"github.com/frostyeti/cast/internal/errors"
	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

func runSshTask(ctx TaskContext) *TaskResult {
	//https://github.com/melbahja/goph

	res := NewTaskResult()
	uses := ctx.Task.Uses
	if uses != "ssh" {
		return res.Fail(errors.New("Invalid uses for SSH task: " + uses))
	}

	targets := ctx.Task.Hosts
	if len(targets) == 0 {
		return res.Fail(errors.New("No targets found for SSH task"))
	}

	for _, target := range targets {
		err := runSSHTarget(ctx.Context, ctx, target)
		if errors.Is(err, context.Canceled) {
			return res.Cancel("Task " + ctx.Task.Id + " cancelled")
		}

		if errors.Is(err, context.DeadlineExceeded) {
			return res.Cancel("Task " + ctx.Task.Id + " cancelled due to timeout")
		}

		if err != nil {
			return res.Fail(errors.New("Failed to run SSH task on target " + target.Host + ": " + err.Error()))
		}
	}

	res.End()

	// Placeholder for running an SSH command
	// This would typically involve executing the command over SSH
	return res.Ok()
}

type SshRun struct {
	Error error
}

func runSSHTarget(ctx context.Context, taskContext TaskContext, target HostInfo) error {
	signal := make(chan SshRun)

	var auth goph.Auth
	var err error
	identity := ""
	password := ""
	run := ""
	if target.IdentityFile != "" {
		identity = target.IdentityFile
	}

	if target.Password != "" {
		password = target.Password
	}

	run = taskContext.Task.Run

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
		signal <- SshRun{Error: errors.New("Failed to create SSH authentication: " + err.Error())}
	}

	port := 22
	if target.Port > 0 {
		port = int(target.Port)
	}
	user := ""
	if target.User != "" {
		user = target.User
	}

	client, err := goph.NewConn(&goph.Config{
		User: user,
		Addr: target.Host,
		Port: uint(port),
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

	var sess *ssh.Session

	if sess, err = client.NewSession(); err != nil {
		err2 := errors.New("Failed to create SSH session: " + err.Error())
		return err2
	}

	defer sess.Close()

	go func() {

		if len(taskContext.Task.Env) > 0 {
			// only set env values that are explicitly set in the task
			for k, v := range taskContext.Task.Env {
				sess.Setenv(k, v)
			}
		}

		sess.Stdout = os.Stdout
		sess.Stderr = os.Stderr
		err = sess.Run(run)

		if err != nil {
			err2 := errors.New("Failed to run command on SSH target " + target.Host + ": " + err.Error())
			err2 = errors.WithCause(err2, err)
			signal <- SshRun{Error: err2}
			return
		}

		signal <- SshRun{Error: nil}
	}()

	select {
	case <-ctx.Done():
		sess.Signal(ssh.SIGINT)
		return ctx.Err()
	case result := <-signal:
		return result.Error
	}
}
