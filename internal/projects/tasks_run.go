package projects

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/eval"
	"github.com/frostyeti/cast/internal/paths"
	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/dotenv"
	"github.com/frostyeti/go/env"
)

type RunTasksParams struct {
	Targets     []string
	Context     context.Context
	ContextName string
	Args        []string
	ProjectName string
	Stdout      io.Writer
	Stderr      io.Writer
}

func findFallbackTask(uses string, projectDir string) (string, bool) {
	tasksDir := os.Getenv("CAST_TASKS_DIR")
	if tasksDir == "" {
		tasksDir = filepath.Join(projectDir, ".cast", "tasks")
	} else if !filepath.IsAbs(tasksDir) {
		tasksDir = filepath.Join(projectDir, tasksDir)
	}

	possiblePaths := []string{
		filepath.Join(tasksDir, uses, "cast.task"),
		filepath.Join(tasksDir, uses, "cast.yaml"),
		filepath.Join(tasksDir, uses+".yaml"),
		filepath.Join(tasksDir, uses+".yml"),
		filepath.Join(tasksDir, uses+".task"),
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		globalTasksDir := filepath.Join(homeDir, ".local", "share", "cast", "tasks")
		possiblePaths = append(possiblePaths,
			filepath.Join(globalTasksDir, uses, "cast.task"),
			filepath.Join(globalTasksDir, uses, "cast.yaml"),
			filepath.Join(globalTasksDir, uses+".yaml"),
			filepath.Join(globalTasksDir, uses+".yml"),
			filepath.Join(globalTasksDir, uses+".task"),
		)
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}

	return "", false
}

func (p *Project) RunTask(params RunTasksParams) ([]*TaskResult, error) {
	p.ContextName = params.ContextName
	err := p.Init()
	if err != nil {
		return nil, err
	}

	allTasks := p.Tasks.Values()

	cyclicalTasks := types.FindCyclicalReferences(allTasks)
	if len(cyclicalTasks) > 0 {
		return nil, NewCyclicalReferenceError(cyclicalTasks)
	}

	results := []*TaskResult{}
	projectEnv := p.Env.Clone()

	taskList, err := p.Tasks.FlattenTasks(params.Targets, params.ContextName)
	if err != nil {
		return nil, err
	}

	castEnv := projectEnv.Get("CAST_ENV")
	castPath := projectEnv.Get("CAST_PATH")
	castOutputs := projectEnv.Get("CAST_OUTPUTS")
	if p.cleanupEnv {
		defer func() {
			if paths.IsFile(castEnv) {
				os.Remove(castEnv)
			}
		}()
	}

	if p.cleanupOutputs {
		defer func() {
			if paths.IsFile(castOutputs) {
				os.Remove(castOutputs)
			}
		}()
	}

	if p.cleanupPath {
		defer func() {
			if paths.IsFile(castPath) {
				os.Remove(castPath)
			}
		}()
	}

	globalOutputs := map[string]interface{}{}

	hasFailed := false

	for _, task := range taskList {

		e := projectEnv.Clone()
		res := NewTaskResult()
		m := &Task{
			Id:   task.Id,
			Name: task.Name,
		}

		name := task.Name

		res.Task = m

		hosts := []HostInfo{}
		hostNames := []string{}
		for _, hostId := range task.Hosts {
			host, ok := p.Hosts[hostId]
			if ok {
				hosts = append(hosts, host)
				continue
			}

			for _, h := range p.Hosts {
				for _, tas := range h.Tags {
					if tas == hostId {
						if !slices.Contains(hostNames, h.Host) {
							hosts = append(hosts, h)
							hostNames = append(hostNames, h.Host)
						}
					}
				}
			}
		}

		opts := &env.ExpandOptions{
			Get: func(key string) string {
				value := e.Get(key)
				return value
			},
			Set: func(key, value string) error {
				e.Set(key, value)
				return nil
			},
			CommandSubstitution: true,
			Keys:                e.Keys(),
		}

		outWriter := params.Stdout
		if outWriter == nil {
			outWriter = os.Stdout
		}

			if len(task.DotEnv) > 0 {
				for _, envFile := range task.DotEnv {
					optional := false
					if strings.HasPrefix(envFile, "?") {
						optional = true
						envFile = envFile[1:]
					} else if strings.HasSuffix(envFile, "?") {
						optional = true
						envFile = envFile[:len(envFile)-1]
					}

					if !filepath.IsAbs(envFile) {
						absPath, err := paths.ResolvePath(p.Dir, envFile)
						if err != nil {
							fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
							err = errors.Newf("failed to resolve dotenv file %s for task %s: %w", envFile, task.Name, err)
							fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
							res.Fail(err)
							hasFailed = true
							results = append(results, res)
							continue
						}
						envFile = absPath
					}

					if paths.IsFile(envFile) {
						data, err := os.ReadFile(envFile)
						if err != nil {
							fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
							err = errors.Newf("failed to read dotenv file %s for task %s: %w", envFile, task.Name, err)
							fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
							res.Fail(err)
							hasFailed = true
							results = append(results, res)
							continue
						}

						doc, err := dotenv.Parse(string(data))
						if err != nil {
							err := errors.Newf("failed to parse dotenv file %s for task %s: %w", envFile, task.Name, err)
							fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
							fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
							res.Fail(err)
							hasFailed = true
							results = append(results, res)
							continue
						}

						for _, node := range doc.ToArray() {
							if node.Type != dotenv.VARIABLE {
								continue
							}

							key := node.Key
							value := node.Value
							if key == nil || *key == "" {
								continue
							}

							v, err := env.ExpandWithOptions(value, opts)
							if err != nil {
								err := errors.Newf("failed to expand variable %s from dotenv file %s for task %s: %w", *key, envFile, task.Name, err)
								fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
								fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
								res.Fail(err)
								hasFailed = true
								results = append(results, res)
								continue
							}

							e.Set(*key, v)
						}
					} else {
						if optional {
							continue
						}
						err := errors.Newf("dotenv file %s does not exist for task %s", envFile, task.Name)
						fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
						fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
						res.Fail(err)
						hasFailed = true
						results = append(results, res)
						continue
					}
				}
			}

		for _, k := range task.Env.Keys() {
			value := task.Env.Get(k)
			v, err := env.ExpandWithOptions(value, opts)
			if err != nil {
				err := errors.Newf("failed to expand env variable %s for task %s: %w", k, task.Name, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				hasFailed = true
				results = append(results, res)
				continue
			}
			e.Set(k, v)
		}

		uses := ""
		if task.Uses != nil {
			uses = *task.Uses
		}

		run := ""
		if task.Run != nil {
			run = *task.Run
		}

		timeout, _ := time.ParseDuration("0s")

		m.Uses = uses
		m.Run = run
		m.Env = e.ToMap()

		m.With = task.With.ToMap()
		m.Timeout = timeout
		m.Hosts = hosts
		m.Args = params.Args
		m.Cwd = ""
		m.Template = ""
		if task.Template != nil {
			m.Template = *task.Template
		}

		scope := p.Scope.Clone()
		scope.Set("env", m.Env)
		scope.Set("outputs", globalOutputs)
		scope.Set("args", m.Args)
		scope.Set("success", !hasFailed)

		for k, v := range globalOutputs {
			// if string, ok := v.(string); ok {
			if str, ok := v.(string); ok {
				m.Env[strings.ToUpper(fmt.Sprintf("OUTPUTS_%s", k))] = str
				continue
			}

			key := k
			if stringMap, ok := v.(map[string]string); ok {
				for sk, sv := range stringMap {
					m.Env[strings.ToUpper(fmt.Sprintf("OUTPUTS_%s_%s", key, sk))] = sv
				}
			}
		}

		force := false
		pred := false
		if task.Force != nil {
			value, err := eval.Eval(*task.Force, scope.ToMap())
			if err != nil {
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				hasFailed = true
				results = append(results, res)
				continue
			}
			force, _ = value.(bool)
		}

		if task.If != nil {
			value, err := eval.Eval(*task.If, scope.ToMap())
			if err != nil {
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				hasFailed = true
				results = append(results, res)
				continue
			}
			pred, _ = value.(bool)
		} else {
			pred = true
		}

		if !pred && !force {
			res.Status = runstatus.Skipped
			results = append(results, res)
			fmt.Fprintf(outWriter, "\x1b[1m%s\x1b[22m (skipped)\n", name)
			continue
		}

		if m.Template == "true" || m.Template == "gotmpl" {
			tmpl, err := template.New("run").Funcs(sprig.FuncMap()).Parse(m.Run)
			if err != nil {
				err := errors.Newf("failed to evaluate template in run for task %s: %w", task.Name, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}
			sb := &strings.Builder{}
			err = tmpl.Execute(sb, scope.ToMap())
			if err != nil {
				err := errors.Newf("failed to evaluate template in run for task %s: %w", task.Name, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}

			m.Run = sb.String()
		}

		if strings.ContainsRune(m.Cwd, '{') {
			tmpl, err := template.New("cwd").Funcs(sprig.FuncMap()).Parse(m.Cwd)
			if err != nil {
				err := errors.Newf("failed to evaluate cwd for task %s: %w", task.Name, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}
			sb := &strings.Builder{}
			err = tmpl.Execute(sb, scope.ToMap())
			if err != nil {
				err := errors.Newf("failed to evaluate cwd for task %s: %w", task.Name, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}

			m.Cwd = sb.String()
		}

		if strings.ContainsRune(m.Cwd, '$') {
			cwd, err := env.ExpandWithOptions(m.Cwd, opts)
			if err != nil {
				err := errors.Newf("failed to evaluate cwd for task %s: %w", task.Name, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}
			m.Cwd = cwd
		}

		if m.Cwd == "" {
			m.Cwd = p.Dir
		}

		if task.Timeout != nil {
			to := *task.Timeout
			if strings.ContainsRune(to, '{') {
				timeoutStr, err := eval.EvalAsString(to, scope.ToMap())
				if err != nil {
					err := errors.Newf("failed to evaluate timeout for task %s: %w", task.Name, err)
					fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
					fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
					res.Fail(err)
					results = append(results, res)
					continue
				}
				to = timeoutStr
			}

			timeout, err = time.ParseDuration(to)
			if err != nil {
				err := errors.Newf("failed to parse task %s timeout %s: %w", task.Name, to, err)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}
		}

		if hasFailed && !force {
			res.Status = runstatus.Skipped
			results = append(results, res)
			fmt.Fprintf(outWriter, "\x1b[1m%s\x1b[22m (skipped)\n", name)
			continue
		}

		handler, ok := GetTaskHandler(uses)
		if !ok {
			if IsRemoteTask(uses) {
				handler = runRemoteTask
			} else if fallbackPath, found := findFallbackTask(uses, p.Dir); found {
				m.Uses = fallbackPath // update Uses to point to the resolved local file
				handler = runRemoteTask
			} else {
				err := errors.Newf("unable to find task handler for %s using %s", task.Name, uses)
				fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m \x1b[31m(failed)\x1b[0m\n", name)
				fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)
				res.Fail(err)
				results = append(results, res)
				continue
			}
		}

		var taskCtx context.Context
		var cancel context.CancelFunc
		if timeout > 0 {
			taskCtx, cancel = context.WithTimeout(params.Context, timeout)
		} else {
			taskCtx, cancel = context.WithCancel(params.Context)
		}

		ctx := &TaskContext{
			Project:     p,
			Context:     taskCtx,
			Schema:      &task,
			Task:        m,
			Args:        task.Args,
			ContextName: params.ContextName,
			Outputs:     globalOutputs,
			Stdout:      params.Stdout,
			Stderr:      params.Stderr,
		}

		if ctx.Stdout == nil {
			ctx.Stdout = os.Stdout
		}
		if ctx.Stderr == nil {
			ctx.Stderr = os.Stderr
		}

		fmt.Fprintf(outWriter, "\n\x1b[1m%s\x1b[22m\n", name)

		// Run handler in a goroutine to support timeout
		resultChan := make(chan *TaskResult, 1)
		go func() {
			result := handler(*ctx)
			resultChan <- result
		}()

		var r2 *TaskResult
		if timeout > 0 {
			select {
			case r2 = <-resultChan:
				// Task completed before timeout
			case <-time.After(timeout):
				// Task timed out
				cancel()
				r2 = NewTaskResult()
				r2.Status = runstatus.Cancelled
				r2.Err = errors.Newf("task %s timed out after %s", task.Name, timeout)
				fmt.Fprintf(outWriter, "\x1b[33m%s (timed out after %s)\x1b[0m\n", name, timeout)
			}
		} else {
			// No timeout, wait indefinitely
			r2 = <-resultChan
		}
		cancel()
		r2.Task = m

		if r2.Status == runstatus.Error || r2.Status == runstatus.Cancelled {
			err := r2.Err
			fmt.Fprintf(outWriter, "\x1b[31m%v\x1b[0m\n", err)

			hasFailed = true
		}

		if r2.Status == runstatus.Ok {
			opts := &env.ExpandOptions{
				Get: func(key string) string {
					value := m.Env[key]
					return value
				},
				Set: func(key, value string) error {
					m.Env[key] = value
					return nil
				},
				CommandSubstitution: true,
			}

			if paths.IsFile(castPath) {
				data, err := os.ReadFile(castPath)
				if err != nil {
					return nil, err
				}

				scanner := bufio.NewScanner(strings.NewReader(string(data)))
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					line := scanner.Text()
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					projectEnv.PrependPath(line)
				}
			}

			if paths.IsFile(castEnv) {
				data, err := os.ReadFile(castEnv)
				if err != nil {
					return nil, err
				}

				doc, err := dotenv.Parse(string(data))
				if err != nil {
					return nil, err
				}

				for _, node := range doc.ToArray() {
					if node.Type != dotenv.VARIABLE {
						continue
					}

					key := node.Key
					value := node.Value
					if key == nil || *key == "" {
						continue
					}

					v, err := env.ExpandWithOptions(value, opts)
					if err != nil {
						return nil, err
					}

					projectEnv.Set(*key, v)
				}
			}

			if paths.IsFile(castOutputs) {
				outputs := map[string]string{}
				data, err := os.ReadFile(castOutputs)
				if err != nil {
					return nil, err
				}

				doc, err := dotenv.Parse(string(data))
				if err != nil {
					return nil, err
				}

				for _, node := range doc.ToArray() {
					if node.Type != dotenv.VARIABLE {
						continue
					}

					key := node.Key
					value := node.Value
					if key == nil || *key == "" {
						continue
					}

					v, err := env.ExpandWithOptions(value, opts)
					if err != nil {
						return nil, err
					}

					outputs[*key] = v
				}
				res.Output = outputs

				globalOutputs[task.Id] = outputs
			}

		}
		results = append(results, r2)
	}

	return results, nil
}

type cyclicalReferenceError struct {
	Cycles []types.Task
}

func (e *cyclicalReferenceError) Error() string {
	msg := "Cyclical references found in tasks:\n"
	for _, cycle := range e.Cycles {
		msg += " - " + cycle.Id + "\n"
	}
	return msg
}

func NewCyclicalReferenceError(cycles []types.Task) error {
	return &cyclicalReferenceError{
		Cycles: cycles,
	}
}
