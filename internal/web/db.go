package web

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/frostyeti/cast/internal/paths"
	_ "modernc.org/sqlite"
)

type JobRun struct {
	ID          string
	ProjectID   string
	JobID       string
	Status      string
	Logs        string
	CreatedAt   time.Time
	CompletedAt *time.Time
	Error       *string
}

func initDB() (*sql.DB, error) {
	var dbPath string
	if runtime.GOOS == "windows" {
		dbPath = filepath.Join("C:\\", "ProgramData", "cast")
	} else {
		dataDir, err := paths.UserDataDir()
		if err != nil {
			return nil, err
		}
		dbPath = dataDir
	}

	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, err
	}

	dbFile := filepath.Join(dbPath, "cast.db")
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	schema := `
	CREATE TABLE IF NOT EXISTS job_runs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		job_id TEXT NOT NULL,
		status TEXT NOT NULL,
		logs TEXT,
		error TEXT,
		created_at DATETIME NOT NULL,
		completed_at DATETIME
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func insertJobRun(db *sql.DB, run JobRun) error {
	_, err := db.Exec(`
		INSERT INTO job_runs (id, project_id, job_id, status, logs, error, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.ProjectID, run.JobID, run.Status, run.Logs, run.Error, run.CreatedAt, run.CompletedAt)
	return err
}

func updateJobRun(db *sql.DB, run JobRun) error {
	_, err := db.Exec(`
		UPDATE job_runs
		SET status = ?, logs = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, run.Status, run.Logs, run.Error, run.CompletedAt, run.ID)
	return err
}

func getJobRuns(db *sql.DB, projectID, jobID string) ([]JobRun, error) {
	rows, err := db.Query(`
		SELECT id, project_id, job_id, status, logs, error, created_at, completed_at
		FROM job_runs
		WHERE project_id = ? AND job_id = ?
		ORDER BY created_at DESC
	`, projectID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var run JobRun
		if err := rows.Scan(
			&run.ID,
			&run.ProjectID,
			&run.JobID,
			&run.Status,
			&run.Logs,
			&run.Error,
			&run.CreatedAt,
			&run.CompletedAt,
		); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
