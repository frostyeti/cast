package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/frostyeti/cast/internal/id"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

type Server struct {
	addr      string
	port      int
	scheduler gocron.Scheduler
	projects  map[string]*projects.Project
	db        *sql.DB
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
	mux.HandleFunc("GET /api/v1/projects", s.handleGetProjects)
	mux.HandleFunc("GET /api/v1/projects/{id}/jobs", s.handleGetJobs)
	mux.HandleFunc("GET /api/v1/projects/{id}/tasks", s.handleGetTasks)
	mux.HandleFunc("GET /api/v1/projects/{id}/jobs/{jobId}/runs", s.handleGetJobRuns)
	mux.HandleFunc("POST /api/v1/projects/{id}/jobs/{jobId}/trigger", s.handleTriggerJob)
	mux.HandleFunc("POST /api/v1/projects/{id}/tasks/{taskId}/trigger", s.handleTriggerTask)

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

	s.projects[proj.Schema.Id] = proj
	log.Printf("Loaded project %s (%s) from %s", proj.Schema.Name, proj.Schema.Id, file)
}

func (s *Server) scheduleJobs() {
	for _, p := range s.projects {
		if p.Schema.On != nil && p.Schema.On.Schedule != nil {
			for _, cronExpr := range p.Schema.On.Schedule.Crons {
				jobTask := func(projectID string) {
					log.Printf("Cron triggered for project %s", projectID)
					s.runJob(projectID, "default")
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
	}
}

func (s *Server) runJob(projectID, jobID string) {
	proj, ok := s.projects[projectID]
	if !ok {
		log.Printf("Project %s not found for job %s", projectID, jobID)
		return
	}

	log.Printf("Executing job %s in project %s", jobID, projectID)

	runID := uuid.New().String()
	startTime := time.Now()

	jobRun := JobRun{
		ID:        runID,
		ProjectID: projectID,
		JobID:     jobID,
		Status:    "running",
		CreatedAt: startTime,
	}

	if err := insertJobRun(s.db, jobRun); err != nil {
		log.Printf("Failed to insert job run: %v", err)
	}

	go func() {
		job, ok := proj.Schema.Jobs.Get(jobID)
		if !ok {
			log.Printf("Job %s not found in project %s", jobID, projectID)
			return
		}

		var logsBuffer bytes.Buffer
		var finalErr error

		for _, step := range job.Steps {
			if step.TaskName != nil {
				logsBuffer.WriteString(fmt.Sprintf("--- Running task: %s ---\n", *step.TaskName))
				params := projects.RunTasksParams{
					Targets:     []string{*step.TaskName},
					Context:     context.Background(),
					ContextName: "default",
					Stdout:      &logsBuffer,
					Stderr:      &logsBuffer,
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
		jobRun.CompletedAt = &endTime
		jobRun.Logs = logsBuffer.String()

		if finalErr != nil {
			errStr := finalErr.Error()
			jobRun.Error = &errStr
			jobRun.Status = "failed"
			jobRun.Logs += fmt.Sprintf("\nError: %v", finalErr)
		} else {
			jobRun.Status = "success"
			log.Printf("Job %s completed successfully", jobID)
		}

		if err := updateJobRun(s.db, jobRun); err != nil {
			log.Printf("Failed to update job run: %v", err)
		}
	}()
}

// HTTP Handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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

	var list []any
	if p.Schema.Jobs != nil {
		for _, j := range p.Schema.Jobs.Values() {
			list = append(list, j)
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

	var list []any
	if p.Schema.Tasks != nil {
		for _, t := range p.Schema.Tasks.Values() {
			list = append(list, t)
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

	runs, err := getJobRuns(s.db, projectID, jobID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get job runs: %v", err), http.StatusInternalServerError)
		return
	}

	if runs == nil {
		runs = []JobRun{}
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

	s.runJob(id, jobId)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf("Triggered job %s in project %s", jobId, id)))
}

func (s *Server) handleTriggerTask(w http.ResponseWriter, r *http.Request) {
	projId := r.PathValue("id")
	taskId := r.PathValue("taskId")

	proj, ok := s.projects[projId]
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	go func() {
		params := projects.RunTasksParams{
			Targets:     []string{taskId},
			Context:     context.Background(),
			ContextName: "default",
		}
		_, err := proj.RunTask(params)
		if err != nil {
			log.Printf("Task %s failed: %v", taskId, err)
		} else {
			log.Printf("Task %s completed successfully", taskId)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf("Triggered task %s in project %s", taskId, projId)))
}
