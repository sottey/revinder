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

func TestStoreCreatesItemAndReturnsPendingItems(t *testing.T) {
	db := openTestStore(t)

	dueAt := time.Date(2026, 6, 16, 20, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	notes := "spoken through kitchen Echo"
	priority := "normal"
	resp, err := db.CreateItem(models.CreateItemRequest{
		RevinderID: "alexa-request-1",
		Source:     "alexa",
		Type:       "task",
		Text:       "on Tuesday at 8pm do that one thing",
		Title:      "do that one thing",
		Notes:      &notes,
		DueAt:      &dueAt,
		Priority:   &priority,
		ListName:   "Home",
		Tags:       []string{"home", "cottage"},
		Metadata: map[string]any{
			"device": "kitchen",
		},
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

	items, err := db.PendingItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}

	item := items[0]
	if item.RevinderID != "alexa-request-1" {
		t.Fatalf("RevinderID = %q, want %q", item.RevinderID, "alexa-request-1")
	}
	if item.Source != "alexa" {
		t.Fatalf("Source = %q, want %q", item.Source, "alexa")
	}
	if item.Type != "task" {
		t.Fatalf("Type = %q, want %q", item.Type, "task")
	}
	if item.Text != "on Tuesday at 8pm do that one thing" {
		t.Fatalf("Text = %q, want %q", item.Text, "on Tuesday at 8pm do that one thing")
	}
	if item.Title != "do that one thing" {
		t.Fatalf("Title = %q, want %q", item.Title, "do that one thing")
	}
	if item.Notes == nil || *item.Notes != notes {
		t.Fatalf("Notes = %v, want %q", item.Notes, notes)
	}
	if item.DueAt == nil || !item.DueAt.Equal(dueAt) {
		t.Fatalf("DueAt = %v, want %v", item.DueAt, dueAt)
	}
	if item.Priority == nil || *item.Priority != priority {
		t.Fatalf("Priority = %v, want %q", item.Priority, priority)
	}
	if item.ListName != "Home" {
		t.Fatalf("ListName = %q, want %q", item.ListName, "Home")
	}
	if len(item.Tags) != 2 || item.Tags[0] != "home" || item.Tags[1] != "cottage" {
		t.Fatalf("Tags = %v, want %v", item.Tags, []string{"home", "cottage"})
	}
	if item.Metadata["device"] != "kitchen" {
		t.Fatalf("Metadata = %v, want device kitchen", item.Metadata)
	}
	if item.Status != "pending" {
		t.Fatalf("Status = %q, want %q", item.Status, "pending")
	}
	if item.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
}

func TestStoreCreateItemReturnsExistingItemForDuplicateRevinderID(t *testing.T) {
	db := openTestStore(t)

	first, err := db.CreateItem(models.CreateItemRequest{
		RevinderID: "alexa-request-1",
		Source:     "alexa",
		Type:       "task",
		Text:       "replace air filter",
		Title:      "replace air filter",
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := db.CreateItem(models.CreateItemRequest{
		RevinderID: "alexa-request-1",
		Source:     "alexa",
		Type:       "task",
		Text:       "replace furnace filter",
		Title:      "replace furnace filter",
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

	items, err := db.PendingItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if items[0].Text != "replace air filter" {
		t.Fatalf("Text = %q, want %q", items[0].Text, "replace air filter")
	}
}

func TestStoreMarkItemProcessedRemovesItemFromPending(t *testing.T) {
	db := openTestStore(t)

	resp, err := db.CreateItem(models.CreateItemRequest{
		Source: "alexa",
		Type:   "task",
		Text:   "replace air filter",
		Title:  "replace air filter",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := db.MarkItemProcessed(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("updated = false, want true")
	}

	items, err := db.PendingItems()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want %d", len(items), 0)
	}

	item, found, err := db.GetItem(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if item.Status != "processed" {
		t.Fatalf("Status = %q, want %q", item.Status, "processed")
	}
	if item.ProcessedAt == nil {
		t.Fatal("ProcessedAt = nil, want timestamp")
	}
}

func TestStoreDeleteItem(t *testing.T) {
	db := openTestStore(t)

	resp, err := db.CreateItem(models.CreateItemRequest{
		Source: "alexa",
		Type:   "note",
		Text:   "remember the battery was replaced",
		Title:  "remember the battery was replaced",
	})
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := db.DeleteItem(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("deleted = false, want true")
	}

	_, found, err := db.GetItem(resp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("found = true, want false")
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
