package projects

import (
	"context"
	"io"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/runstatus"
)

type RunJobParams struct {
	JobID         string
	Context       context.Context
	ContextName   string
	Args          []string
	Stdout        io.Writer
	Stderr        io.Writer
	RunDownstream bool
}

// GetDownstreamJobs returns the job ID and all jobs that transitively depend on it, topologically sorted.
func (p *Project) GetDownstreamJobs(startJobID string) ([]string, error) {
	if p.Schema.Jobs == nil {
		return nil, errors.New("no jobs defined in project")
	}

	startJob, ok := p.Schema.Jobs.Get(startJobID)
	if !ok {
		return nil, errors.Newf("job %s not found", startJobID)
	}
	startJobID = startJob.Id

	// 1. Build adjacency list for jobs: job -> jobs that need it
	graph := make(map[string][]string)
	inDegree := make(map[string]int) // Number of dependencies IN THE SUBGRAPH

	for _, job := range p.Schema.Jobs.Values() {
		// Initialize
		if _, ok := graph[job.Id]; !ok {
			graph[job.Id] = []string{}
		}

		if job.Needs != nil {
			for _, need := range *job.Needs {
				needJob, ok := p.Schema.Jobs.Get(need.Id)
				if !ok {
					return nil, errors.Newf("needed job %s not found", need.Id)
				}
				needId := needJob.Id
				graph[needId] = append(graph[needId], job.Id)
			}
		}
	}

	// 2. Find all jobs reachable from startJobID (DFS/BFS)
	visited := make(map[string]bool)
	queue := []string{startJobID}
	visited[startJobID] = true

	var subgraph []string

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		subgraph = append(subgraph, curr)

		for _, neighbor := range graph[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// 3. Topological sort the subgraph
	// First calculate in-degrees ONLY considering edges within the subgraph
	for _, node := range subgraph {
		inDegree[node] = 0
	}

	for _, u := range subgraph {
		for _, v := range graph[u] {
			if visited[v] { // if v is in subgraph
				inDegree[v]++
			}
		}
	}

	// Kahn's algorithm
	var sorted []string
	var zeroInDegree []string

	// Find nodes with 0 in-degree in the subgraph (startJobID should be one)
	for _, node := range subgraph {
		if inDegree[node] == 0 {
			zeroInDegree = append(zeroInDegree, node)
		}
	}

	for len(zeroInDegree) > 0 {
		curr := zeroInDegree[0]
		zeroInDegree = zeroInDegree[1:]
		sorted = append(sorted, curr)

		for _, neighbor := range graph[curr] {
			if visited[neighbor] {
				inDegree[neighbor]--
				if inDegree[neighbor] == 0 {
					zeroInDegree = append(zeroInDegree, neighbor)
				}
			}
		}
	}

	if len(sorted) != len(subgraph) {
		return nil, errors.New("cycle detected in job dependencies")
	}

	return sorted, nil
}

func (p *Project) RunJob(params RunJobParams) error {
	p.ContextName = params.ContextName
	if err := p.Init(); err != nil {
		return err
	}

	jobsToRun := []string{params.JobID}
	var err error

	if params.RunDownstream {
		jobsToRun, err = p.GetDownstreamJobs(params.JobID)
		if err != nil {
			return err
		}
	}

	for _, jobID := range jobsToRun {
		job, ok := p.Schema.Jobs.Get(jobID)
		if !ok {
			return errors.Newf("job %s not found", jobID)
		}

		for _, step := range job.Steps {
			if step.TaskName != nil {
				runParams := RunTasksParams{
					Targets:     []string{*step.TaskName},
					Context:     params.Context,
					ContextName: params.ContextName,
					Args:        params.Args,
					Stdout:      params.Stdout,
					Stderr:      params.Stderr,
				}

				results, err := p.RunTask(runParams)
				if err != nil {
					return errors.Newf("job %s failed at step %s: %w", jobID, *step.TaskName, err)
				}

				for _, res := range results {
					if res.Status == runstatus.Error {
						return errors.Newf("job %s failed at step %s: %w", jobID, *step.TaskName, res.Err)
					}
					if res.Status == runstatus.Cancelled {
						return errors.Newf("job %s cancelled at step %s", jobID, *step.TaskName)
					}
				}
			}
			// Future: handling for run/uses inside steps directly
		}
	}

	return nil
}
