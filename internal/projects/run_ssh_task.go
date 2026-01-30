package projects

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/frostyeti/cast/internal/errors"
	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

// prefixedWriter wraps an io.Writer and prefixes each line with a host identifier
type prefixedWriter struct {
	prefix string
	writer io.Writer
	mu     sync.Mutex
	buf    strings.Builder
}

func newPrefixedWriter(prefix string, writer io.Writer) *prefixedWriter {
	return &prefixedWriter{
		prefix: prefix,
		writer: writer,
	}
}

func (pw *prefixedWriter) Write(p []byte) (n int, err error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	originalLen := len(p)
	pw.buf.Write(p)

	for {
		line, rest, found := strings.Cut(pw.buf.String(), "\n")
		if !found {
			break
		}
		fmt.Fprintf(pw.writer, "[%s]: %s\n", pw.prefix, line)
		pw.buf.Reset()
		pw.buf.WriteString(rest)
	}

	return originalLen, nil
}

func (pw *prefixedWriter) Flush() {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.buf.Len() > 0 {
		fmt.Fprintf(pw.writer, "[%s]: %s\n", pw.prefix, pw.buf.String())
		pw.buf.Reset()
	}
}

type sshJobResult struct {
	Host  string
	Error error
}

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

	maxParallel := 5

	maxParallelEnv, envOk := ctx.Task.Env["CAST_SSH_MAX_PARALLEL"]
	if envOk {
		mp, err := strconv.Atoi(maxParallelEnv)
		if err == nil && mp > 0 {
			maxParallel = mp
		}
	}

	maxParallelValue, ok := ctx.Task.With["max-parallel"]
	if ok {
		maxParallelStr, isString := maxParallelValue.(string)
		if isString && maxParallelStr != "" {
			mp, err := strconv.Atoi(maxParallelStr)
			if err != nil || mp <= 0 {
				return res.Fail(errors.New("Invalid max-parallel value for SSH task: " + maxParallelStr))
			}
			maxParallel = mp
		}
	}

	if maxParallel > 0 {
		// Run in parallel with worker pool
		err := runSSHTargetsParallel(ctx.Context, ctx, targets, maxParallel)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return res.Cancel("Task " + ctx.Task.Id + " cancelled")
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return res.Cancel("Task " + ctx.Task.Id + " cancelled due to timeout")
			}
			return res.Fail(err)
		}
	} else {
		// Run sequentially
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
	}

	res.End()
	return res.Ok()
}

func runSSHTargetsParallel(ctx context.Context, taskContext TaskContext, targets []HostInfo, maxParallel int) error {
	// Create a cancellable context for all workers
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	// Channel to send targets to workers
	jobs := make(chan HostInfo, len(targets))
	// Channel to receive results from workers
	results := make(chan sshJobResult, len(targets))

	// Track if we should stop sending new jobs
	var stopSending sync.Once
	var hasError bool
	var errorMu sync.Mutex
	var firstError error

	// Start worker pool
	var wg sync.WaitGroup
	workerCount := maxParallel
	if workerCount > len(targets) {
		workerCount = len(targets)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-workerCtx.Done():
					return
				case target, ok := <-jobs:
					if !ok {
						return
					}
					err := runSSHTarget(workerCtx, taskContext, target)
					results <- sshJobResult{Host: target.Host, Error: err}
				}
			}
		}()
	}

	// Goroutine to send jobs
	go func() {
		for _, target := range targets {
			// Check if we should stop sending new jobs
			errorMu.Lock()
			shouldStop := hasError
			errorMu.Unlock()

			if shouldStop {
				break
			}

			select {
			case <-workerCtx.Done():
			case jobs <- target:
			}
		}
		close(jobs)
	}()

	// Goroutine to close results channel after all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var collectedErrors []string
	for result := range results {
		if result.Error != nil {
			errorMu.Lock()
			if !hasError {
				hasError = true
				firstError = result.Error
				stopSending.Do(func() {}) // Mark that we should stop sending
			}
			collectedErrors = append(collectedErrors, fmt.Sprintf("[%s]: %s", result.Host, result.Error.Error()))
			errorMu.Unlock()
		}
	}

	// Check if context was cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Return combined error if any failures occurred
	if len(collectedErrors) > 0 {
		if len(collectedErrors) == 1 {
			return firstError
		}
		return errors.New("SSH tasks failed on multiple hosts:\n" + strings.Join(collectedErrors, "\n"))
	}

	return nil
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
		return errors.New("Failed to create SSH authentication: " + err.Error())
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

		// Use prefixed writers for output
		stdoutWriter := newPrefixedWriter(target.Host, os.Stdout)
		stderrWriter := newPrefixedWriter(target.Host, os.Stderr)

		// Create pipes for stdout and stderr to handle line-by-line output
		stdoutPipe, err := sess.StdoutPipe()
		if err != nil {
			signal <- SshRun{Error: errors.New("Failed to create stdout pipe: " + err.Error())}
			return
		}
		stderrPipe, err := sess.StderrPipe()
		if err != nil {
			signal <- SshRun{Error: errors.New("Failed to create stderr pipe: " + err.Error())}
			return
		}

		// Start the command
		if err := sess.Start(run); err != nil {
			err2 := errors.New("Failed to start command on SSH target " + target.Host + ": " + err.Error())
			err2 = errors.WithCause(err2, err)
			signal <- SshRun{Error: err2}
			return
		}

		// Stream output with prefixes
		var outputWg sync.WaitGroup
		outputWg.Add(2)

		go func() {
			defer outputWg.Done()
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				stdoutWriter.Write([]byte(scanner.Text() + "\n"))
			}
			stdoutWriter.Flush()
		}()

		go func() {
			defer outputWg.Done()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				stderrWriter.Write([]byte(scanner.Text() + "\n"))
			}
			stderrWriter.Flush()
		}()

		// Wait for output streaming to complete
		outputWg.Wait()

		// Wait for command to finish
		err = sess.Wait()
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
