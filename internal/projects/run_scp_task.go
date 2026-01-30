package projects

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/frostyeti/cast/internal/errors"
	goph "github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

type scpJobResult struct {
	Host  string
	Error error
}

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

	maxParallel := 5

	maxParallelEnv, envOk := ctx.Task.Env["CAST_SCP_MAX_PARALLEL"]
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
				return res.Fail(errors.New("Invalid max-parallel value for SCP task: " + maxParallelStr))
			}
			maxParallel = mp
		}
	}

	if maxParallel > 0 {
		// Run in parallel with worker pool
		err := runSCPTargetsParallel(ctx.Context, direction, ctx, targets, files, maxParallel)
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
			if err := runScpTarget(ctx.Context, direction, ctx, target, files); err != nil {
				if errors.Is(err, context.Canceled) {
					return res.Cancel("Task " + ctx.Task.Id + " cancelled")
				}
				if errors.Is(err, context.DeadlineExceeded) {
					return res.Cancel("Task " + ctx.Task.Id + " cancelled due to timeout")
				}
				return res.Fail(err)
			}
		}
	}

	res.End()
	return res.Ok()
}

func runSCPTargetsParallel(ctx context.Context, direction string, taskContext TaskContext, targets []HostInfo, files []string, maxParallel int) error {
	// Create a cancellable context for all workers
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	// Channel to send targets to workers
	jobs := make(chan HostInfo, len(targets))
	// Channel to receive results from workers
	results := make(chan scpJobResult, len(targets))

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
					err := runScpTarget(workerCtx, direction, taskContext, target, files)
					results <- scpJobResult{Host: target.Host, Error: err}
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
		return errors.New("SCP tasks failed on multiple hosts:\n" + strings.Join(collectedErrors, "\n"))
	}

	return nil
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
			fmt.Fprintf(os.Stdout, "[%s]: Uploading %s to %s\n", target.Host, source, destination)
			err = Upload(ctx, client, source, destination)
		} else {
			fmt.Fprintf(os.Stdout, "[%s]: Downloading %s to %s\n", target.Host, source, destination)
			err = Download(ctx, client, destination, source)
		}

		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			err2 := errors.New("Failed to transfer file " + source + " to " + destination + ": " + err.Error())
			err2 = errors.WithCause(err2, err)
			return err2
		}

		fmt.Fprintf(os.Stdout, "[%s]: Transfer complete: %s\n", target.Host, file)
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
