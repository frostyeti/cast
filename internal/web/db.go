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

type Run struct {
	ID          string
	ProjectID   string
	Type        string
	TargetID    string
	Status      string
	Logs        string
	CreatedAt   time.Time
	CompletedAt *time.Time
	Error       *string
	TriggeredBy *string
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
	CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		status TEXT NOT NULL,
		logs TEXT,
		error TEXT,
		created_at DATETIME NOT NULL,
		completed_at DATETIME,
		triggered_by TEXT
	);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	// Add triggered_by column if upgrading from earlier version without it
	_, _ = db.Exec(`ALTER TABLE runs ADD COLUMN triggered_by TEXT;`)

	// Clear any stale running runs left over from previous sudden shutdowns
	_, _ = db.Exec(`UPDATE runs SET status = 'failed', error = 'Server restarted' WHERE status = 'running'`)

	return db, nil
}

func insertRun(db *sql.DB, run Run) error {
	_, err := db.Exec(`
		INSERT INTO runs (id, project_id, type, target_id, status, logs, error, created_at, completed_at, triggered_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.ProjectID, run.Type, run.TargetID, run.Status, run.Logs, run.Error, run.CreatedAt, run.CompletedAt, run.TriggeredBy)
	return err
}

func updateRun(db *sql.DB, run Run) error {
	_, err := db.Exec(`
		UPDATE runs
		SET status = ?, logs = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, run.Status, run.Logs, run.Error, run.CompletedAt, run.ID)
	return err
}

func getRunLogs(db *sql.DB, id string) (string, error) {
	var logs string
	err := db.QueryRow("SELECT logs FROM runs WHERE id = ?", id).Scan(&logs)
	return logs, err
}

func getRuns(db *sql.DB, projectID, targetType, targetID string) ([]Run, error) {
	rows, err := db.Query(`
		SELECT id, project_id, type, target_id, status, logs, error, created_at, completed_at, triggered_by
		FROM runs
		WHERE project_id = ? AND type = ? AND target_id = ?
		ORDER BY created_at DESC
	`, projectID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(
			&run.ID,
			&run.ProjectID,
			&run.Type,
			&run.TargetID,
			&run.Status,
			&run.Logs,
			&run.Error,
			&run.CreatedAt,
			&run.CompletedAt,
			&run.TriggeredBy,
		); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func getActiveRunIDs(db *sql.DB, projectID, targetType string) (map[string]string, error) {
	rows, err := db.Query(`
		SELECT target_id, id
		FROM runs
		WHERE project_id = ? AND type = ? AND status = 'running'
	`, projectID, targetType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activeRuns := make(map[string]string)
	for rows.Next() {
		var targetID, runID string
		if err := rows.Scan(&targetID, &runID); err != nil {
			return nil, err
		}
		activeRuns[targetID] = runID
	}
	return activeRuns, rows.Err()
}
