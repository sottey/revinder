package store

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sottey/revinder_bridge/internal/models"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.createSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) createSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    revinder_bridge_id TEXT,
    title TEXT NOT NULL,
    source TEXT NOT NULL,
    list_name TEXT NOT NULL DEFAULT 'Home',
    due_at DATETIME,
    all_day INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    tags TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL,
    synced_at DATETIME
);
`)
	if err != nil {
		return err
	}
	if err := s.addColumnIfMissing("revinder_bridge_id", "TEXT"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("tags", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	_, err = s.db.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_revinder_bridge_id
ON tasks(revinder_bridge_id)
WHERE revinder_bridge_id IS NOT NULL AND revinder_bridge_id != '';
`)
	return err
}

func (s *Store) addColumnIfMissing(name string, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var columnName string
		var columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = s.db.Exec("ALTER TABLE tasks ADD COLUMN " + name + " " + definition)
	return err
}

func (s *Store) CreateTask(req models.CreateTaskRequest) (models.CreateTaskResponse, error) {
	listName := req.ListName
	if listName == "" {
		listName = "Home"
	}
	if req.RevinderBridgeID != "" {
		resp, found, err := s.getCreateTaskResponseByRevinderBridgeID(req.RevinderBridgeID)
		if err != nil {
			return models.CreateTaskResponse{}, err
		}
		if found {
			return resp, nil
		}
	}

	reqTags := req.Tags
	if reqTags == nil {
		reqTags = []string{}
	}
	tags, err := json.Marshal(reqTags)
	if err != nil {
		return models.CreateTaskResponse{}, err
	}

	createdAt := time.Now()
	result, err := s.db.Exec(`
INSERT INTO tasks (revinder_bridge_id, title, source, list_name, due_at, all_day, notes, tags, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)
`, emptyStringNil(req.RevinderBridgeID), req.Title, req.Source, listName, nullableTime(req.DueAt), boolInt(req.AllDay), req.Notes, string(tags), createdAt.Format(time.RFC3339Nano))
	if err != nil {
		if req.RevinderBridgeID != "" {
			resp, found, lookupErr := s.getCreateTaskResponseByRevinderBridgeID(req.RevinderBridgeID)
			if lookupErr != nil {
				return models.CreateTaskResponse{}, lookupErr
			}
			if found {
				return resp, nil
			}
		}
		return models.CreateTaskResponse{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.CreateTaskResponse{}, err
	}

	return models.CreateTaskResponse{
		ID:     id,
		Status: "pending",
	}, nil
}

func (s *Store) PendingTasks() ([]models.Task, error) {
	return s.tasksByStatus("pending")
}

func (s *Store) SyncedTasks() ([]models.Task, error) {
	return s.tasksByStatus("synced")
}

func (s *Store) GetTask(id int64) (models.Task, bool, error) {
	row := s.db.QueryRow(`
SELECT id, revinder_bridge_id, title, source, list_name, due_at, all_day, notes, tags, status, created_at, synced_at
FROM tasks
WHERE id = ?
`, id)

	task, err := scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Task{}, false, nil
		}
		return models.Task{}, false, err
	}

	return task, true, nil
}

func (s *Store) tasksByStatus(status string) ([]models.Task, error) {
	rows, err := s.db.Query(`
SELECT id, revinder_bridge_id, title, source, list_name, due_at, all_day, notes, tags, status, created_at, synced_at
FROM tasks
WHERE status = ?
ORDER BY id
`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (models.Task, error) {
	var task models.Task
	var revinderBridgeID sql.NullString
	var dueAt sql.NullString
	var allDay int
	var notes sql.NullString
	var tags string
	var createdAt string
	var syncedAt sql.NullString

	err := scanner.Scan(
		&task.ID,
		&revinderBridgeID,
		&task.Title,
		&task.Source,
		&task.ListName,
		&dueAt,
		&allDay,
		&notes,
		&tags,
		&task.Status,
		&createdAt,
		&syncedAt,
	)
	if err != nil {
		return models.Task{}, err
	}

	task.AllDay = allDay != 0
	if revinderBridgeID.Valid {
		task.RevinderBridgeID = revinderBridgeID.String
	}
	if tags == "" {
		tags = "[]"
	}
	if err := json.Unmarshal([]byte(tags), &task.Tags); err != nil {
		return models.Task{}, err
	}
	if task.Tags == nil {
		task.Tags = []string{}
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return models.Task{}, err
	}
	task.CreatedAt = parsedCreatedAt

	if dueAt.Valid {
		parsedDueAt, err := time.Parse(time.RFC3339Nano, dueAt.String)
		if err != nil {
			return models.Task{}, err
		}
		task.DueAt = &parsedDueAt
	}
	if notes.Valid {
		task.Notes = &notes.String
	}
	if syncedAt.Valid {
		parsedSyncedAt, err := time.Parse(time.RFC3339Nano, syncedAt.String)
		if err != nil {
			return models.Task{}, err
		}
		task.SyncedAt = &parsedSyncedAt
	}

	return task, nil
}

func (s *Store) getCreateTaskResponseByRevinderBridgeID(revinderBridgeID string) (models.CreateTaskResponse, bool, error) {
	row := s.db.QueryRow(`
SELECT id, status
FROM tasks
WHERE revinder_bridge_id = ?
`, revinderBridgeID)

	var resp models.CreateTaskResponse
	if err := row.Scan(&resp.ID, &resp.Status); err != nil {
		if err == sql.ErrNoRows {
			return models.CreateTaskResponse{}, false, nil
		}
		return models.CreateTaskResponse{}, false, err
	}

	return resp, true, nil
}

func (s *Store) MarkSynced(id int64) (bool, error) {
	result, err := s.db.Exec(`
UPDATE tasks
SET status = 'synced', synced_at = ?
WHERE id = ?
`, time.Now().Format(time.RFC3339Nano), id)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func (s *Store) MarkPending(id int64) (bool, error) {
	result, err := s.db.Exec(`
UPDATE tasks
SET status = 'pending', synced_at = NULL
WHERE id = ?
`, id)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func emptyStringNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
