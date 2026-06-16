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
	if err := s.addColumnIfMissing("tasks", "revinder_bridge_id", "TEXT"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing("tasks", "tags", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_revinder_bridge_id
ON tasks(revinder_bridge_id)
WHERE revinder_bridge_id IS NOT NULL AND revinder_bridge_id != '';
`); err != nil {
		return err
	}
	_, err = s.db.Exec(`
CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    revinder_id TEXT,
    source TEXT NOT NULL,
    type TEXT NOT NULL,
    text TEXT NOT NULL,
    title TEXT NOT NULL,
    notes TEXT,
    due_at DATETIME,
    priority TEXT,
    list_name TEXT NOT NULL DEFAULT 'default',
    tags TEXT NOT NULL DEFAULT '[]',
    metadata TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL,
    processed_at DATETIME
);
`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_items_revinder_id
ON items(revinder_id)
WHERE revinder_id IS NOT NULL AND revinder_id != '';
`)
	return err
}

func (s *Store) addColumnIfMissing(table string, name string, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
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

	_, err = s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + name + " " + definition)
	return err
}

func (s *Store) CreateItem(req models.CreateItemRequest) (models.CreateItemResponse, error) {
	listName := req.ListName
	if listName == "" {
		listName = "default"
	}
	if req.RevinderID != "" {
		resp, found, err := s.getCreateItemResponseByRevinderID(req.RevinderID)
		if err != nil {
			return models.CreateItemResponse{}, err
		}
		if found {
			return resp, nil
		}
	}

	tags, err := marshalStringSlice(req.Tags)
	if err != nil {
		return models.CreateItemResponse{}, err
	}
	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return models.CreateItemResponse{}, err
	}

	createdAt := time.Now()
	result, err := s.db.Exec(`
INSERT INTO items (revinder_id, source, type, text, title, notes, due_at, priority, list_name, tags, metadata, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)
`, emptyStringNil(req.RevinderID), req.Source, req.Type, req.Text, req.Title, req.Notes, nullableTime(req.DueAt), req.Priority, listName, tags, metadata, createdAt.Format(time.RFC3339Nano))
	if err != nil {
		if req.RevinderID != "" {
			resp, found, lookupErr := s.getCreateItemResponseByRevinderID(req.RevinderID)
			if lookupErr != nil {
				return models.CreateItemResponse{}, lookupErr
			}
			if found {
				return resp, nil
			}
		}
		return models.CreateItemResponse{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.CreateItemResponse{}, err
	}

	return models.CreateItemResponse{
		ID:     id,
		Status: "pending",
	}, nil
}

func (s *Store) Items(status string, itemType string) ([]models.Item, error) {
	query := `
SELECT id, revinder_id, source, type, text, title, notes, due_at, priority, list_name, tags, metadata, status, created_at, processed_at
FROM items
`
	var rows *sql.Rows
	var err error
	switch {
	case status == "" && itemType == "":
		rows, err = s.db.Query(query + `ORDER BY id`)
	case status != "" && itemType == "":
		rows, err = s.db.Query(query+`WHERE status = ?
ORDER BY id`, status)
	case status == "" && itemType != "":
		rows, err = s.db.Query(query+`WHERE type = ?
ORDER BY id`, itemType)
	default:
		rows, err = s.db.Query(query+`WHERE status = ? AND type = ?
ORDER BY id`, status, itemType)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Item{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Store) PendingItems() ([]models.Item, error) {
	return s.Items("pending", "")
}

func (s *Store) GetItem(id int64) (models.Item, bool, error) {
	row := s.db.QueryRow(`
SELECT id, revinder_id, source, type, text, title, notes, due_at, priority, list_name, tags, metadata, status, created_at, processed_at
FROM items
WHERE id = ?
`, id)

	item, err := scanItem(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Item{}, false, nil
		}
		return models.Item{}, false, err
	}

	return item, true, nil
}

func (s *Store) MarkItemProcessed(id int64) (bool, error) {
	result, err := s.db.Exec(`
UPDATE items
SET status = 'processed', processed_at = ?
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

func (s *Store) MarkItemFailed(id int64) (bool, error) {
	result, err := s.db.Exec(`
UPDATE items
SET status = 'failed'
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

func (s *Store) DeleteItem(id int64) (bool, error) {
	result, err := s.db.Exec(`
DELETE FROM items
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

func scanItem(scanner taskScanner) (models.Item, error) {
	var item models.Item
	var revinderID sql.NullString
	var notes sql.NullString
	var dueAt sql.NullString
	var priority sql.NullString
	var tags string
	var metadata string
	var createdAt string
	var processedAt sql.NullString

	err := scanner.Scan(
		&item.ID,
		&revinderID,
		&item.Source,
		&item.Type,
		&item.Text,
		&item.Title,
		&notes,
		&dueAt,
		&priority,
		&item.ListName,
		&tags,
		&metadata,
		&item.Status,
		&createdAt,
		&processedAt,
	)
	if err != nil {
		return models.Item{}, err
	}

	if revinderID.Valid {
		item.RevinderID = revinderID.String
	}
	if notes.Valid {
		item.Notes = &notes.String
	}
	if dueAt.Valid {
		parsedDueAt, err := time.Parse(time.RFC3339Nano, dueAt.String)
		if err != nil {
			return models.Item{}, err
		}
		item.DueAt = &parsedDueAt
	}
	if priority.Valid {
		item.Priority = &priority.String
	}

	if tags == "" {
		tags = "[]"
	}
	if err := json.Unmarshal([]byte(tags), &item.Tags); err != nil {
		return models.Item{}, err
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}

	if metadata == "" {
		metadata = "{}"
	}
	if err := json.Unmarshal([]byte(metadata), &item.Metadata); err != nil {
		return models.Item{}, err
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return models.Item{}, err
	}
	item.CreatedAt = parsedCreatedAt

	if processedAt.Valid {
		parsedProcessedAt, err := time.Parse(time.RFC3339Nano, processedAt.String)
		if err != nil {
			return models.Item{}, err
		}
		item.ProcessedAt = &parsedProcessedAt
	}

	return item, nil
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

func (s *Store) getCreateItemResponseByRevinderID(revinderID string) (models.CreateItemResponse, bool, error) {
	row := s.db.QueryRow(`
SELECT id, status
FROM items
WHERE revinder_id = ?
`, revinderID)

	var resp models.CreateItemResponse
	if err := row.Scan(&resp.ID, &resp.Status); err != nil {
		if err == sql.ErrNoRows {
			return models.CreateItemResponse{}, false, nil
		}
		return models.CreateItemResponse{}, false, err
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

func marshalStringSlice(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func marshalMetadata(value map[string]any) (string, error) {
	if value == nil {
		value = map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
