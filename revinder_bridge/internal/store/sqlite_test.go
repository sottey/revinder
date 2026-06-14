package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sottey/revinder_bridge/internal/models"
)

func TestStoreCreatesTaskAndReturnsPendingTasks(t *testing.T) {
	db := openTestStore(t)

	dueAt := time.Date(2026, 6, 15, 9, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	notes := "replace quarterly"
	resp, err := db.CreateTask(models.CreateTaskRequest{
		RevinderBridgeID: "alexa-request-1",
		Title:            "replace air filter",
		Source:           "alexa",
		DueAt:            &dueAt,
		AllDay:           true,
		Notes:            &notes,
		Tags:             []string{"hvac", "quarterly"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.ID != 1 {
		t.Fatalf("ID = %d, want %d", resp.ID, 1)
	}
	if resp.Status != "pending" {
		t.Fatalf("Status = %q, want %q", resp.Status, "pending")
	}

	tasks, err := db.PendingTasks()
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want %d", len(tasks), 1)
	}

	task := tasks[0]
	if task.RevinderBridgeID != "alexa-request-1" {
		t.Fatalf("RevinderBridgeID = %q, want %q", task.RevinderBridgeID, "alexa-request-1")
	}
	if task.Title != "replace air filter" {
		t.Fatalf("Title = %q, want %q", task.Title, "replace air filter")
	}
	if task.Source != "alexa" {
		t.Fatalf("Source = %q, want %q", task.Source, "alexa")
	}
	if task.ListName != "Home" {
		t.Fatalf("ListName = %q, want %q", task.ListName, "Home")
	}
	if task.DueAt == nil || !task.DueAt.Equal(dueAt) {
		t.Fatalf("DueAt = %v, want %v", task.DueAt, dueAt)
	}
	if !task.AllDay {
		t.Fatal("AllDay = false, want true")
	}
	if task.Notes == nil || *task.Notes != notes {
		t.Fatalf("Notes = %v, want %q", task.Notes, notes)
	}
	if len(task.Tags) != 2 || task.Tags[0] != "hvac" || task.Tags[1] != "quarterly" {
		t.Fatalf("Tags = %v, want %v", task.Tags, []string{"hvac", "quarterly"})
	}
	if task.Status != "pending" {
		t.Fatalf("Status = %q, want %q", task.Status, "pending")
	}
	if task.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
}

func TestStoreCreateTaskReturnsExistingTaskForDuplicateRevinderBridgeID(t *testing.T) {
	db := openTestStore(t)

	first, err := db.CreateTask(models.CreateTaskRequest{
		RevinderBridgeID: "alexa-request-1",
		Title:            "replace air filter",
		Source:           "alexa",
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := db.CreateTask(models.CreateTaskRequest{
		RevinderBridgeID: "alexa-request-1",
		Title:            "replace furnace filter",
		Source:           "alexa",
	})
	if err != nil {
		t.Fatal(err)
	}

	if second.ID != first.ID {
		t.Fatalf("duplicate ID = %d, want %d", second.ID, first.ID)
	}
	if second.Status != first.Status {
		t.Fatalf("duplicate Status = %q, want %q", second.Status, first.Status)
	}

	tasks, err := db.PendingTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want %d", len(tasks), 1)
	}
	if tasks[0].Title != "replace air filter" {
		t.Fatalf("Title = %q, want %q", tasks[0].Title, "replace air filter")
	}
}

func TestStorePreservesExplicitListName(t *testing.T) {
	db := openTestStore(t)

	resp, err := db.CreateTask(models.CreateTaskRequest{
		Title:    "replace air filter",
		Source:   "alexa",
		ListName: "Errands",
	})
	if err != nil {
		t.Fatal(err)
	}

	task, found, err := db.GetTask(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if task.ListName != "Errands" {
		t.Fatalf("ListName = %q, want %q", task.ListName, "Errands")
	}
}

func TestStoreMarkSyncedRemovesTaskFromPending(t *testing.T) {
	db := openTestStore(t)

	resp, err := db.CreateTask(models.CreateTaskRequest{
		Title:  "replace air filter",
		Source: "alexa",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := db.MarkSynced(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("updated = false, want true")
	}

	tasks, err := db.PendingTasks()
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 0 {
		t.Fatalf("len(tasks) = %d, want %d", len(tasks), 0)
	}

	task, found, err := db.GetTask(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if task.Status != "synced" {
		t.Fatalf("Status = %q, want %q", task.Status, "synced")
	}
	if task.SyncedAt == nil {
		t.Fatal("SyncedAt = nil, want timestamp")
	}

	syncedTasks, err := db.SyncedTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(syncedTasks) != 1 {
		t.Fatalf("len(syncedTasks) = %d, want %d", len(syncedTasks), 1)
	}

	if syncedTasks[0].Status != "synced" {
		t.Fatalf("Status = %q, want %q", syncedTasks[0].Status, "synced")
	}
}

func TestStoreMarkPendingResetsSyncedTask(t *testing.T) {
	db := openTestStore(t)

	resp, err := db.CreateTask(models.CreateTaskRequest{
		Title:  "replace air filter",
		Source: "alexa",
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated, err := db.MarkSynced(resp.ID); err != nil {
		t.Fatal(err)
	} else if !updated {
		t.Fatal("MarkSynced updated = false, want true")
	}

	updated, err := db.MarkPending(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("MarkPending updated = false, want true")
	}

	task, found, err := db.GetTask(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if task.Status != "pending" {
		t.Fatalf("Status = %q, want %q", task.Status, "pending")
	}
	if task.SyncedAt != nil {
		t.Fatalf("SyncedAt = %v, want nil", task.SyncedAt)
	}
}

func TestStoreMarkSyncedReturnsFalseForMissingTask(t *testing.T) {
	db := openTestStore(t)

	updated, err := db.MarkSynced(999)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("updated = true, want false")
	}
}

func TestStoreMarkPendingReturnsFalseForMissingTask(t *testing.T) {
	db := openTestStore(t)

	updated, err := db.MarkPending(999)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("updated = true, want false")
	}
}

func TestStoreGetTaskReturnsFalseForMissingTask(t *testing.T) {
	db := openTestStore(t)

	_, found, err := db.GetTask(999)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("found = true, want false")
	}
}

func TestOpenCreatesDatabaseParentDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "nested", "revinder_bridge.sqlite")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	info, err := os.Stat(filepath.Dir(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", filepath.Dir(dbPath))
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal(err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := Open(filepath.Join(t.TempDir(), "revinder_bridge.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
