package projects

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
		for _, hostId := range task.Hosts {
			host, ok := p.Hosts[hostId]
			if ok {
				hosts = append(hosts, host)
				continue
			}

			for _, h := range p.Hosts {
				for _, tas := range h.Tags {
					if tas == hostId {
						hosts = append(hosts, h)
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

		if len(task.DotEnv) > 0 {
			for _, envFile := range task.DotEnv {
				if paths.IsFile(envFile) {
					data, err := os.ReadFile(envFile)
					if err != nil {
						os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
						err = errors.Newf("failed to read dotenv file %s for task %s: %w", envFile, task.Name, err)
						os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
						res.Fail(err)
						hasFailed = true
						results = append(results, res)
						continue
					}

					doc, err := dotenv.Parse(string(data))
					if err != nil {
						err := errors.Newf("failed to parse dotenv file %s for task %s: %w", envFile, task.Name, err)
						os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
						os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
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
							os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
							os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
							res.Fail(err)
							hasFailed = true
							results = append(results, res)
							continue
						}

						e.Set(*key, v)
					}
				} else {
					err := errors.Newf("dotenv file %s does not exist for task %s", envFile, task.Name)
					os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
					os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
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
				os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
				os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
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

		scope := p.Scope.Clone()
		scope.Set("env", m.Env)

		force := false
		pred := false
		if task.Force != nil {
			value, err := eval.Eval(*task.Force, scope.ToMap())
			if err != nil {
				os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
				os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
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
				os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
				os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
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
			os.Stdout.WriteString("\x1b[1m" + name + "\x1b[22m (skipped)\n")
			continue
		}

		if strings.ContainsRune(m.Cwd, '{') {
			cwd, err := eval.EvalAsString(m.Cwd, scope.ToMap())
			if err != nil {
				err := errors.Newf("failed to evaluate cwd for task %s: %w", task.Name, err)
				os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
				os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
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
					os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
					os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
					res.Fail(err)
					results = append(results, res)
					continue
				}
				to = timeoutStr
			}

			timeout, err = time.ParseDuration(to)
			if err != nil {
				err := errors.Newf("failed to parse task %s timeout %s: %w", task.Name, to, err)
				os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
				os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
				res.Fail(err)
				results = append(results, res)
				continue
			}
		}

		if hasFailed && !force {
			res.Status = runstatus.Skipped
			results = append(results, res)
			os.Stdout.WriteString("\x1b[1m" + name + "\x1b[22m (skipped)\n")
			continue
		}

		handler, ok := GetTaskHandler(uses)
		if !ok {
			err := errors.Newf("unable to find task handler for %s using %s", task.Name, uses)
			os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m \x1b[31m(failed)\x1b[0m\n")
			os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))
			res.Fail(err)
			results = append(results, res)
			continue
		}

		ctx := &TaskContext{
			Context:     params.Context,
			Schema:      &task,
			Task:        m,
			Args:        task.Args,
			ContextName: params.ContextName,
		}

		os.Stdout.WriteString("\n\x1b[1m" + name + "\x1b[22m\n")
		r2 := handler(*ctx)
		r2.Task = m

		if r2.Status == runstatus.Error || r2.Status == runstatus.Cancelled {
			err := r2.Err
			os.Stdout.WriteString(fmt.Sprintf("\x1b[31m%v\x1b[0m\n", err))

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
				outputs := map[string]interface{}{}
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
