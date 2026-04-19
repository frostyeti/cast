package projects

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/paths"
	"github.com/frostyeti/go/env"
	goph "github.com/melbahja/goph"
	"github.com/pkg/sftp"
)

type scpJobResult struct {
	Host  string
	Error error
}

func expandScpValue(taskEnv map[string]string, value string) (string, error) {
	if !strings.ContainsRune(value, '$') {
		return value, nil
	}

	return env.ExpandWithOptions(value, &env.ExpandOptions{
		Get: func(key string) string {
			if taskEnv == nil {
				return ""
			}
			return taskEnv[key]
		},
		Set: func(string, string) error {
			return nil
		},
		CommandSubstitution: false,
	})
}

func normalizeOptionalScpPath(path string) (string, bool) {
	optional := false
	trimmed := strings.TrimSpace(path)

	for strings.HasPrefix(trimmed, "?") {
		optional = true
		trimmed = strings.TrimSpace(trimmed[1:])
	}

	for strings.HasSuffix(trimmed, "?") {
		optional = true
		trimmed = strings.TrimSpace(trimmed[:len(trimmed)-1])
	}

	return trimmed, optional
}

func splitScpFileSpec(file string) (string, string, bool, error) {
	parts := strings.SplitN(file, ":", 2)
	if len(parts) != 2 {
		return "", "", false, errors.New("Invalid SCP file format, expected 'source:destination'")
	}

	source, optional := normalizeOptionalScpPath(parts[0])
	destination := strings.TrimSpace(parts[1])
	if source == "" || destination == "" {
		return "", "", false, errors.New("Invalid SCP file format, expected 'source:destination'")
	}

	return source, destination, optional, nil
}

func taskScpWorkingDir(ctx TaskContext) string {
	if ctx.Task != nil && ctx.Task.Cwd != "" {
		return ctx.Task.Cwd
	}

	if ctx.Project != nil && ctx.Project.Dir != "" {
		return ctx.Project.Dir
	}

	return "."
}

func resolveScpLocalPath(ctx TaskContext, value string) (string, error) {
	expanded, err := expandScpValue(ctx.Task.Env, value)
	if err != nil {
		return "", err
	}

	if expanded == "" {
		return "", errors.New("empty local path for SCP transfer")
	}

	if filepath.IsAbs(expanded) {
		return expanded, nil
	}

	return paths.ResolvePath(taskScpWorkingDir(ctx), expanded)
}

func resolveScpRemotePath(ctx TaskContext, value string) (string, error) {
	expanded, err := expandScpValue(ctx.Task.Env, value)
	if err != nil {
		return "", err
	}

	if expanded == "" {
		return "", errors.New("empty remote path for SCP transfer")
	}

	return expanded, nil
}

func remoteScpFileExists(client *goph.Client, remotePath string) (bool, error) {
	ftp, err := client.NewSftp()
	if err != nil {
		return false, err
	}
	defer func() {
		_ = ftp.Close()
	}()

	if _, err := ftp.Stat(remotePath); err != nil {
		var statusErr *sftp.StatusError
		if stderrors.As(err, &statusErr) && statusErr.FxCode() == sftp.ErrSSHFxNoSuchFile {
			return false, nil
		}

		if stderrors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func isMissingScpFile(err error) bool {
	if err == nil {
		return false
	}

	var statusErr *sftp.StatusError
	if stderrors.As(err, &statusErr) && statusErr.FxCode() == sftp.ErrSSHFxNoSuchFile {
		return true
	}

	return stderrors.Is(err, os.ErrNotExist)
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
		} else if arr2, ok := v.([]string); ok {
			files = append(files, arr2...)
		}
	}

	if len(files) == 0 {
		return res.Fail(errors.New("No files specified for SCP task"))
	}

	direction := ""
	if v, ok := ctx.Task.With["direction"]; ok {
		if s, ok := v.(string); ok {
			direction = strings.TrimSpace(strings.ToLower(s))
		}
	}

	if direction == "" {
		uri, err := url.Parse(ctx.Task.Uses)
		if err != nil {
			return res.Fail(errors.New("Invalid SSH URI: " + err.Error()))
		}

		if uri.Scheme != "scp" && uri.Scheme != "" {
			return res.Fail(errors.New("Invalid SSH URI scheme: " + uri.Scheme))
		}

		direction = strings.TrimSpace(strings.ToLower(uri.Query().Get("direction")))
		if direction == "" {
			download := strings.EqualFold(uri.Query().Get("download"), "true")
			if download {
				direction = "download"
			} else {
				path := strings.TrimSpace(strings.ToLower(uri.Path))
				if path != "" && path != "scp" {
					direction = path
				}
			}
		}
	}

	if direction == "" {
		direction = "upload"
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
	authCfg, err := projectSSHAuthConfig(taskContext.Project, target, "")
	if err != nil {
		return err
	}

	client, _, err := newSSHClient(authCfg)
	if err != nil {
		return err
	}

	defer func() {
		_ = client.Close()
	}()

	for _, file := range files {
		source, destination, optional, err := splitScpFileSpec(file)
		if err != nil {
			return err
		}

		var transferErr error

		if direction != "download" {
			sourcePath, err := resolveScpLocalPath(taskContext, source)
			if err != nil {
				err2 := errors.New("Failed to resolve SCP source path: " + err.Error())
				err2 = errors.WithCause(err2, err)
				return err2
			}

			if optional {
				if _, err := os.Stat(sourcePath); err != nil {
					if os.IsNotExist(err) {
						continue
					}

					err2 := errors.New("Failed to stat optional SCP source path: " + err.Error())
					err2 = errors.WithCause(err2, err)
					return err2
				}
			}

			destinationPath, err := resolveScpRemotePath(taskContext, destination)
			if err != nil {
				err2 := errors.New("Failed to resolve SCP destination path: " + err.Error())
				err2 = errors.WithCause(err2, err)
				return err2
			}

			_, _ = fmt.Fprintf(os.Stdout, "[%s]: Uploading %s to %s\n", target.Host, source, destination)
			transferErr = Upload(ctx, client, sourcePath, destinationPath)
		} else {
			remoteSource, err := resolveScpRemotePath(taskContext, source)
			if err != nil {
				err2 := errors.New("Failed to resolve SCP source path: " + err.Error())
				err2 = errors.WithCause(err2, err)
				return err2
			}

			if optional {
				exists, err := remoteScpFileExists(client, remoteSource)
				if err != nil {
					err2 := errors.New("Failed to check optional SCP source path: " + err.Error())
					err2 = errors.WithCause(err2, err)
					return err2
				}
				if !exists {
					continue
				}
			}

			destinationPath, err := resolveScpLocalPath(taskContext, destination)
			if err != nil {
				err2 := errors.New("Failed to resolve SCP destination path: " + err.Error())
				err2 = errors.WithCause(err2, err)
				return err2
			}

			_, _ = fmt.Fprintf(os.Stdout, "[%s]: Downloading %s to %s\n", target.Host, source, destination)
			transferErr = Download(ctx, client, remoteSource, destinationPath)
		}

		if transferErr != nil {
			if optional && direction == "download" && isMissingScpFile(transferErr) {
				continue
			}

			if errors.Is(transferErr, context.Canceled) || errors.Is(transferErr, context.DeadlineExceeded) {
				return transferErr
			}
			err2 := errors.New("Failed to transfer file " + source + " to " + destination + ": " + transferErr.Error())
			err2 = errors.WithCause(err2, transferErr)
			return err2
		}

		_, _ = fmt.Fprintf(os.Stdout, "[%s]: Transfer complete: %s\n", target.Host, file)
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
	defer func() {
		_ = local.Close()
	}()

	ftp, err := c.NewSftp()
	if err != nil {
		return
	}
	defer func() {
		_ = ftp.Close()
	}()

	remote, err := ftp.Create(remotePath)
	if err != nil {
		return
	}
	defer func() {
		_ = remote.Close()
	}()

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
	ftp, err := c.NewSftp()
	if err != nil {
		return
	}
	defer func() {
		_ = ftp.Close()
	}()

	remote, err := ftp.Open(remotePath)
	if err != nil {
		return
	}
	defer func() {
		_ = remote.Close()
	}()

	local, err := os.Create(localPath)
	if err != nil {
		return
	}
	defer func() {
		_ = local.Close()
	}()

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
