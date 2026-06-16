package jsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sottey/revinder/consumers/revinder_task_consumer/internal/bridge"
)

func TestWriterAppendsTaskRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.jsonl")
	createdAt := time.Date(2026, 6, 16, 19, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	dueAt := time.Date(2026, 6, 17, 9, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	notes := "replace quarterly"
	priority := "normal"

	writer := New(path)
	if err := writer.Process(context.Background(), bridge.Item{
		ID:         12,
		RevinderID: "alexa-request-1",
		Source:     "alexa",
		Type:       "task",
		Text:       "replace air filter",
		Title:      "replace air filter",
		Notes:      &notes,
		DueAt:      &dueAt,
		Priority:   &priority,
		ListName:   "Home",
		Tags:       []string{"home"},
		Metadata: map[string]any{
			"all_day": false,
		},
		CreatedAt: createdAt,
	}); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("missing JSONL record")
	}

	var record Record
	if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.ID != 12 {
		t.Fatalf("ID = %d, want 12", record.ID)
	}
	if record.Text != "replace air filter" {
		t.Fatalf("Text = %q, want replace air filter", record.Text)
	}
	if record.Title != "" {
		t.Fatalf("Title = %q, want empty when title equals text", record.Title)
	}
	if record.Notes == nil || *record.Notes != notes {
		t.Fatalf("Notes = %v, want %q", record.Notes, notes)
	}
	if record.DueAt == nil || !record.DueAt.Equal(dueAt) {
		t.Fatalf("DueAt = %v, want %v", record.DueAt, dueAt)
	}
	if record.Priority == nil || *record.Priority != priority {
		t.Fatalf("Priority = %v, want %q", record.Priority, priority)
	}
	if record.ListName != "Home" {
		t.Fatalf("ListName = %q, want Home", record.ListName)
	}
	if len(record.Tags) != 1 || record.Tags[0] != "home" {
		t.Fatalf("Tags = %v, want [home]", record.Tags)
	}
	if record.Metadata["all_day"] != false {
		t.Fatalf("Metadata = %v, want all_day false", record.Metadata)
	}
	if !record.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", record.CreatedAt, createdAt)
	}
	if scanner.Scan() {
		t.Fatalf("unexpected extra record %q", scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}
