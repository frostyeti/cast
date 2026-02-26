package web

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/frostyeti/cast/internal/id"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/cast/internal/types"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

//go:embed static
var staticFiles embed.FS

type Server struct {
	addr      string
	port      int
	scheduler gocron.Scheduler
	projects  map[string]*projects.Project
	db        *sql.DB

	streamsMu sync.RWMutex
	streams   map[string]*LogBroadcaster
}

func NewServer(addr string, port int) *Server {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("failed to init scheduler: %v", err)
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	return &Server{
		addr:      addr,
		port:      port,
		scheduler: s,
		projects:  make(map[string]*projects.Project),
		db:        db,
		streams:   make(map[string]*LogBroadcaster),
	}
}

func (s *Server) Start() error {
	log.Printf("Discovering Cast files...")
	s.discoverProjects()

	log.Printf("Scheduling jobs...")
	s.scheduleJobs()

	s.scheduler.Start()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/help", s.handleHelp)
	mux.HandleFunc("GET /api/v1/projects", s.handleGetProjects)
	mux.HandleFunc("GET /api/v1/projects/{id}/jobs", s.handleGetJobs)
	mux.HandleFunc("GET /api/v1/projects/{id}/jobs/{jobId}", s.handleGetJob)
	mux.HandleFunc("GET /api/v1/projects/{id}/tasks", s.handleGetTasks)
	mux.HandleFunc("GET /api/v1/projects/{id}/tasks/{taskId}", s.handleGetTask)
	mux.HandleFunc("GET /api/v1/projects/{id}/jobs/{jobId}/runs", s.handleGetJobRuns)
	mux.HandleFunc("GET /api/v1/projects/{id}/tasks/{taskId}/runs", s.handleGetTaskRuns)
	mux.HandleFunc("POST /api/v1/projects/{id}/jobs/{jobId}/trigger", s.handleTriggerJob)
	mux.HandleFunc("POST /api/v1/projects/{id}/tasks/{taskId}/trigger", s.handleTriggerTask)
	mux.HandleFunc("GET /api/v1/projects/{id}/ssh/{host_alias}", s.handleSSHStream)
	mux.HandleFunc("GET /api/v1/streams/{runId}", s.handleStream)
	mux.HandleFunc("GET /api/v1/runs/{runId}/logs", s.handleDownloadRunLogs)
	mux.HandleFunc("POST /api/webhooks/{webhookId}", s.handleWebhook)

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("failed to create static fs: %v", err)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Clean the path to prevent directory traversal
		p := filepath.Clean(r.URL.Path)
		p = strings.TrimPrefix(p, "/")

		if p == "" {
			p = "index.html"
		}

		// Try to read the file from the embedded filesystem
		file, err := staticFS.Open(p)
		if err != nil {
			if os.IsNotExist(err) {
				// Fallback to index.html for SPA routing
				indexFile, err := staticFS.Open("index.html")
				if err != nil {
					http.Error(w, "index.html not found", http.StatusInternalServerError)
					return
				}
				defer indexFile.Close()

				stat, err := indexFile.Stat()
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// If it's a directory, return index.html
		if stat.IsDir() {
			indexFile, err := staticFS.Open("index.html")
			if err != nil {
				http.Error(w, "index.html not found", http.StatusInternalServerError)
				return
			}
			defer indexFile.Close()

			stat, err := indexFile.Stat()
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
			return
		}

		http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
	})

	bind := fmt.Sprintf("%s:%d", s.addr, s.port)
	log.Printf("Cast web server listening on http://%s", bind)
	return http.ListenAndServe(bind, mux)
}

func (s *Server) discoverProjects() {
	cwd, _ := os.Getwd()

	paths := []string{}

	// 1. CWD
	if p := findCastfileInDir(cwd); p != "" {
		paths = append(paths, p)
	}

	// 2. Global dirs
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join("C:\\", "ProgramData", "cast", "jobs", "*.yaml"))
		paths = append(paths, filepath.Join("C:\\", "ProgramData", "cast", "jobs", "*.yml"))
	} else {
		paths = append(paths, "/etc/cast/jobs/*.yaml", "/etc/cast/jobs/*.yml")
	}

	// 3. Local relative dirs
	paths = append(paths, filepath.Join(cwd, "cast", "jobs", "*.yaml"))
	paths = append(paths, filepath.Join(cwd, "cast", "jobs", "*.yml"))
	paths = append(paths, filepath.Join(cwd, ".cast", "jobs", "*.yaml"))
	paths = append(paths, filepath.Join(cwd, ".cast", "jobs", "*.yml"))

	for _, pattern := range paths {
		if strings.Contains(pattern, "*") {
			matches, _ := filepath.Glob(pattern)
			for _, m := range matches {
				s.loadProject(m)
			}
		} else {
			if _, err := os.Stat(pattern); err == nil {
				s.loadProject(pattern)
			}
		}
	}
}

func findCastfileInDir(dir string) string {
	candidates := []string{"castfile", "castfile.yml", "castfile.yaml", "Castfile", ".castfile"}
	for _, c := range candidates {
		p := filepath.Join(dir, c)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (s *Server) loadProject(file string) {
	proj := &projects.Project{}
	if err := proj.LoadFromYaml(file); err != nil {
		log.Printf("Warning: failed to parse %s: %v", file, err)
		return
	}
	proj.Init()

	// ID / Name Generation
	base := filepath.Base(file)
	if proj.Schema.Name == "" {
		if strings.ToLower(base) == "castfile.yaml" || strings.ToLower(base) == "castfile.yml" || strings.ToLower(base) == "castfile" {
			proj.Schema.Name = filepath.Base(filepath.Dir(file))
		} else {
			ext := filepath.Ext(base)
			proj.Schema.Name = strings.TrimSuffix(base, ext)
		}
	}
	if proj.Schema.Id == "" {
		proj.Schema.Id = id.Convert(strings.ReplaceAll(proj.Schema.Name, " ", "-"))
	}

	// Sanitize the ID when running as a server
	proj.Schema.Id = id.Sanitize(proj.Schema.Id)

	if proj.Schema.Jobs != nil {
		sanitizedJobs := types.NewJobMap()
		for _, j := range proj.Schema.Jobs.Values() {
			j.Id = id.Sanitize(j.Id)
			if j.Needs != nil {
				newNeeds := make(types.Needs, len(*j.Needs))
				for i, need := range *j.Needs {
					need.Id = id.Sanitize(need.Id)
					newNeeds[i] = need
				}
				j.Needs = &newNeeds
			}
			sanitizedJobs.Add(&j)
		}
		proj.Schema.Jobs = sanitizedJobs
	}

	s.projects[proj.Schema.Id] = proj
	log.Printf("Loaded project %s (%s) from %s", proj.Schema.Name, proj.Schema.Id, file)
}

func (s *Server) scheduleJobs() {
	for _, p := range s.projects {
		// Project level crons (trigger default job)
		if p.Schema.On != nil && p.Schema.On.Schedule != nil {
			for _, cronExpr := range p.Schema.On.Schedule.Crons {
				jobTask := func(projectID string) {
					log.Printf("Cron triggered for project %s", projectID)
					trigger := "project cron"
					s.runJob(projectID, "default", nil, &trigger)
				}

				_, err := s.scheduler.NewJob(
					gocron.CronJob(cronExpr, false),
					gocron.NewTask(jobTask, p.Schema.Id),
				)
				if err != nil {
					log.Printf("Failed to schedule cron '%s' for project %s: %v", cronExpr, p.Schema.Id, err)
				}
			}
		}

		// Job level crons
		if p.Schema.Jobs != nil {
			for _, job := range p.Schema.Jobs.Values() {
				if job.Cron != nil && *job.Cron != "" {
					if job.Needs != nil && len(*job.Needs) > 0 {
						log.Printf("Warning: Job %s in project %s has a cron expression but also has dependencies (needs). Ignoring cron schedule.", job.Id, p.Schema.Id)
						continue
					}

					cronExpr := *job.Cron
					jobID := job.Id
					jobTask := func(projectID string, jID string) {
						log.Printf("Job Cron triggered for project %s, job %s", projectID, jID)
						trigger := "job cron"
						s.runJob(projectID, jID, nil, &trigger)
					}

					_, err := s.scheduler.NewJob(
						gocron.CronJob(cronExpr, false),
						gocron.NewTask(jobTask, p.Schema.Id, jobID),
					)
					if err != nil {
						log.Printf("Failed to schedule job cron '%s' for project %s, job %s: %v", cronExpr, p.Schema.Id, jobID, err)
					}
				}
			}
		}
	}
}

func (s *Server) runJob(projectID, jobID string, env map[string]string, triggeredBy *string) string {
	proj, ok := s.projects[projectID]
	if !ok {
		log.Printf("Project %s not found for job %s", projectID, jobID)
		return ""
	}

	job, ok := proj.Schema.Jobs.Get(jobID)
	if !ok {
		log.Printf("Job %s not found in project %s", jobID, projectID)
		return ""
	}

	// Use canonical ID
	jobID = job.Id

	log.Printf("Executing job %s in project %s", jobID, projectID)

	runID := uuid.New().String()
	startTime := time.Now()

	run := Run{
		ID:          runID,
		ProjectID:   projectID,
		Type:        "job",
		TargetID:    jobID,
		Status:      "running",
		CreatedAt:   startTime,
		TriggeredBy: triggeredBy,
	}

	if err := insertRun(s.db, run); err != nil {
		log.Printf("Failed to insert run: %v", err)
	}

	broadcaster := NewLogBroadcaster()
	s.streamsMu.Lock()
	s.streams[runID] = broadcaster
	s.streamsMu.Unlock()

	go func() {
		defer func() {
			broadcaster.Close()
			s.streamsMu.Lock()
			delete(s.streams, runID)
			s.streamsMu.Unlock()
		}()

		var finalErr error

		for _, step := range job.Steps {
			if step.TaskName != nil {
				broadcaster.Write([]byte(fmt.Sprintf("--- Running task: %s ---\n", *step.TaskName)))
				params := projects.RunTasksParams{
					Targets:     []string{*step.TaskName},
					Context:     context.Background(),
					ContextName: "default",
					Env:         env,
					Stdout:      broadcaster,
					Stderr:      broadcaster,
				}
				_, err := proj.RunTask(params)
				if err != nil {
					finalErr = err
					log.Printf("Job %s failed at step %s: %v", jobID, *step.TaskName, err)
					break
				}
			}
		}

		endTime := time.Now()
		run.CompletedAt = &endTime
		run.Logs = broadcaster.String()

		if finalErr != nil {
			errStr := finalErr.Error()
			run.Error = &errStr
			run.Status = "failed"
			run.Logs += fmt.Sprintf("\nError: %v", finalErr)
		} else {
			run.Status = "success"
			log.Printf("Job %s completed successfully", jobID)

			// Find and trigger dependent jobs
			var nextJobs []string
			if proj.Schema.Jobs != nil {
				for _, j := range proj.Schema.Jobs.Values() {
					if j.Needs != nil {
						for _, need := range *j.Needs {
							if strings.EqualFold(need.Id, jobID) {
								nextJobs = append(nextJobs, j.Id)
								break
							}
						}
					}
				}
			}

			if len(nextJobs) > 0 {
				log.Printf("Job %s triggered dependent jobs: %v", jobID, nextJobs)
				broadcaster.Write([]byte(fmt.Sprintf("\n--- Triggering dependent jobs: %v ---\n", nextJobs)))
				for _, next := range nextJobs {
					go func(n string) {
						time.Sleep(100 * time.Millisecond) // slight delay to ensure logs order
						trigger := fmt.Sprintf("job:%s", jobID)
						s.runJob(projectID, n, env, &trigger)
					}(next)
				}
			}
		}

		if err := updateRun(s.db, run); err != nil {
			log.Printf("Failed to update run: %v", err)
		}
	}()
	return runID
}

// HTTP Handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleDownloadRunLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")

	s.streamsMu.RLock()
	broadcaster, active := s.streams[runID]
	s.streamsMu.RUnlock()

	var logs string
	if active {
		logs = broadcaster.String()
	} else {
		var err error
		logs, err = getRunLogs(s.db, runID)
		if err != nil {
			http.Error(w, "Logs not found", http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"run-%s.log\"", runID))
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(logs))
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")

	s.streamsMu.RLock()
	broadcaster, active := s.streams[runID]
	s.streamsMu.RUnlock()

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	if !active {
		// If the stream is no longer active, maybe it's a finished job.
		// Try to find it in the DB and return the whole log as one event, then close.
		logs, err := getRunLogs(s.db, runID)
		if err == nil && logs != "" {
			// Write the historical logs
			fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(logs, "\n", "\ndata: "))
			flusher.Flush()
			return
		}

		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Subscribe to the active stream
	ch, history := broadcaster.Subscribe()
	defer broadcaster.Unsubscribe(ch)

	// Send history first
	if history != "" {
		fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(history, "\n", "\ndata: "))
		flusher.Flush()
	}

	// Listen for new chunks
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				// Stream closed
				return
			}
			if chunk != "" {
				fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(chunk, "\n", "\ndata: "))
				flusher.Flush()
			}
		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

func (s *Server) handleGetProjects(w http.ResponseWriter, r *http.Request) {
	var list []map[string]any
	for _, p := range s.projects {
		list = append(list, map[string]any{
			"id":   p.Schema.Id,
			"name": p.Schema.Name,
			"desc": p.Schema.Desc,
			"file": p.Schema.File,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleGetJobs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, ok := s.projects[id]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	activeRuns, _ := getActiveRunIDs(s.db, id, "job")

	var list []map[string]any
	if p.Schema.Jobs != nil {
		for _, j := range p.Schema.Jobs.Values() {
			jobMap := map[string]any{
				"id":    j.Id,
				"name":  j.Name,
				"desc":  j.Desc,
				"needs": j.Needs,
				"steps": j.Steps,
				"cron":  j.Cron,
			}
			if activeID, active := activeRuns[j.Id]; active {
				jobMap["activeRunId"] = activeID
			}
			list = append(list, jobMap)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, ok := s.projects[id]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	activeRuns, _ := getActiveRunIDs(s.db, id, "task")

	var list []map[string]any
	if p.Schema.Tasks != nil {
		for _, t := range p.Schema.Tasks.Values() {
			taskMap := map[string]any{
				"id":   t.Id,
				"name": t.Name,
				"desc": t.Desc,
				"run":  t.Run,
			}
			if activeID, active := activeRuns[t.Id]; active {
				taskMap["activeRunId"] = activeID
			}
			list = append(list, taskMap)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleGetJobRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	jobID := r.PathValue("jobId")

	if _, ok := s.projects[projectID]; !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	runs, err := getRuns(s.db, projectID, "job", jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get job runs: %v", err), http.StatusInternalServerError)
		return
	}

	if runs == nil {
		runs = []Run{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (s *Server) handleGetTaskRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	taskID := r.PathValue("taskId")

	if _, ok := s.projects[projectID]; !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	runs, err := getRuns(s.db, projectID, "task", taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get task runs: %v", err), http.StatusInternalServerError)
		return
	}

	if runs == nil {
		runs = []Run{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (s *Server) handleTriggerJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	jobId := r.PathValue("jobId")

	if _, ok := s.projects[id]; !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	trigger := "manual"
	runID := s.runJob(id, jobId, nil, &trigger)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Triggered job %s in project %s", jobId, id),
		"runId":   runID,
	})
}

func (s *Server) handleTriggerTask(w http.ResponseWriter, r *http.Request) {
	projId := r.PathValue("id")
	taskId := r.PathValue("taskId")

	proj, ok := s.projects[projId]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	runID := uuid.New().String()
	startTime := time.Now()

	run := Run{
		ID:        runID,
		ProjectID: projId,
		Type:      "task",
		TargetID:  taskId,
		Status:    "running",
		CreatedAt: startTime,
	}

	if err := insertRun(s.db, run); err != nil {
		log.Printf("Failed to insert run: %v", err)
	}

	broadcaster := NewLogBroadcaster()
	s.streamsMu.Lock()
	s.streams[runID] = broadcaster
	s.streamsMu.Unlock()

	go func() {
		defer func() {
			broadcaster.Close()
			s.streamsMu.Lock()
			delete(s.streams, runID)
			s.streamsMu.Unlock()
		}()

		params := projects.RunTasksParams{
			Targets:     []string{taskId},
			Context:     context.Background(),
			ContextName: "default",
			Stdout:      broadcaster,
			Stderr:      broadcaster,
		}
		_, err := proj.RunTask(params)

		endTime := time.Now()
		run.CompletedAt = &endTime
		run.Logs = broadcaster.String()

		if err != nil {
			errStr := err.Error()
			run.Error = &errStr
			run.Status = "failed"
			run.Logs += fmt.Sprintf("\nError: %v", err)
			log.Printf("Task %s failed: %v", taskId, err)
			broadcaster.Write([]byte(fmt.Sprintf("\nError: %v\n", err)))
		} else {
			run.Status = "success"
			log.Printf("Task %s completed successfully", taskId)
		}

		if err := updateRun(s.db, run); err != nil {
			log.Printf("Failed to update run: %v", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Triggered task %s in project %s", taskId, projId),
		"runId":   runID,
	})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	jobId := r.PathValue("jobId")

	p, ok := s.projects[projectId]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	if p.Schema.Jobs == nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	job, ok := p.Schema.Jobs.Get(jobId)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	projectId := r.PathValue("id")
	taskId := r.PathValue("taskId")

	p, ok := s.projects[projectId]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	if p.Schema.Tasks == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	task, ok := p.Schema.Tasks.Get(taskId)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	endpoints := []map[string]string{
		{"method": "GET", "path": "/health", "description": "Health check"},
		{"method": "GET", "path": "/api/help", "description": "List all API endpoints"},
		{"method": "GET", "path": "/api/v1/projects", "description": "List all projects"},
		{"method": "GET", "path": "/api/v1/projects/{id}/jobs", "description": "List all jobs for a project"},
		{"method": "GET", "path": "/api/v1/projects/{id}/jobs/{jobId}", "description": "Get a specific job for a project"},
		{"method": "GET", "path": "/api/v1/projects/{id}/tasks", "description": "List all tasks for a project"},
		{"method": "GET", "path": "/api/v1/projects/{id}/tasks/{taskId}", "description": "Get a specific task for a project"},
		{"method": "GET", "path": "/api/v1/projects/{id}/jobs/{jobId}/runs", "description": "List runs for a specific job"},
		{"method": "POST", "path": "/api/v1/projects/{id}/jobs/{jobId}/trigger", "description": "Trigger a job in a project"},
		{"method": "POST", "path": "/api/v1/projects/{id}/tasks/{taskId}/trigger", "description": "Trigger a task in a project"},
		{"method": "POST", "path": "/api/webhooks/{webhookId}", "description": "Trigger a webhook"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	webhookId := r.PathValue("webhookId")

	// Find the project and webhook configuration
	var targetProj *projects.Project
	var targetWebhook *types.Webhook

	for _, p := range s.projects {
		if p.Schema.On != nil && p.Schema.On.Webhooks != nil {
			if hook, ok := p.Schema.On.Webhooks[webhookId]; ok {
				targetProj = p
				targetWebhook = &hook
				break
			}
		}
	}

	if targetProj == nil || targetWebhook == nil {
		http.Error(w, "webhook not found", http.StatusNotFound)
		return
	}

	// Read payload
	var payload map[string]interface{}
	var bodyBytes []byte

	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		// Reset the body so we can still use it
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > 0 {
			err := json.Unmarshal(bodyBytes, &payload)
			if err != nil {
				http.Error(w, "invalid json payload", http.StatusBadRequest)
				return
			}
		}
	}

	// Validate secret/token if configured
	if targetWebhook.Secret != "" {
		// GitHub style signature
		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			http.Error(w, "missing signature", http.StatusUnauthorized)
			return
		}

		mac := hmac.New(sha256.New, []byte(targetWebhook.Secret))
		mac.Write(bodyBytes)
		expectedMAC := mac.Sum(nil)
		expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)

		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	if targetWebhook.Token != "" {
		// Bearer token style
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+targetWebhook.Token {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Prepare environment variables from query params and payload
	env := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			env["WEBHOOK_QUERY_"+strings.ToUpper(k)] = v[0]
		}
	}

	if payload != nil {
		for k, v := range payload {
			// Basic serialization for top-level keys
			switch val := v.(type) {
			case string:
				env["WEBHOOK_PAYLOAD_"+strings.ToUpper(k)] = val
			case float64, int, bool:
				env["WEBHOOK_PAYLOAD_"+strings.ToUpper(k)] = fmt.Sprintf("%v", val)
			default:
				b, _ := json.Marshal(val)
				env["WEBHOOK_PAYLOAD_"+strings.ToUpper(k)] = string(b)
			}
		}
	}

	if targetWebhook.Job != "" {
		trigger := "webhook"
		runID := s.runJob(targetProj.Schema.Id, targetWebhook.Job, env, &trigger)
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Triggered job %s via webhook", targetWebhook.Job),
			"runId":   runID,
		})
		return
	}

	if targetWebhook.Task != "" {
		runID := uuid.New().String()
		broadcaster := NewLogBroadcaster()
		s.streamsMu.Lock()
		s.streams[runID] = broadcaster
		s.streamsMu.Unlock()

		go func() {
			defer func() {
				broadcaster.Close()
				s.streamsMu.Lock()
				delete(s.streams, runID)
				s.streamsMu.Unlock()
			}()

			params := projects.RunTasksParams{
				Targets:     []string{targetWebhook.Task},
				Context:     context.Background(),
				ContextName: "default",
				Env:         env,
				Stdout:      broadcaster,
				Stderr:      broadcaster,
			}
			_, err := targetProj.RunTask(params)
			if err != nil {
				log.Printf("Webhook Task %s failed: %v", targetWebhook.Task, err)
				broadcaster.Write([]byte(fmt.Sprintf("\nError: %v\n", err)))
			} else {
				log.Printf("Webhook Task %s completed successfully", targetWebhook.Task)
			}
		}()
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Triggered task %s via webhook", targetWebhook.Task),
			"runId":   runID,
		})
		return
	}

	http.Error(w, "webhook configuration missing job or task", http.StatusInternalServerError)
}
