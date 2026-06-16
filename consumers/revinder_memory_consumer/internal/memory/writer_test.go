package memory

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sottey/revinder/consumers/revinder_memory_consumer/internal/bridge"
)

func TestWriterAppendsMemoryRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.jsonl")
	createdAt := time.Date(2026, 6, 16, 20, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	writer := New(path)
	if err := writer.Write(context.Background(), bridge.Item{
		ID:         12,
		RevinderID: "alexa-request-1",
		Source:     "alexa",
		Type:       "memory",
		Text:       "my dog's name is Minnie",
		Title:      "my dog's name is Minnie",
		Tags:       []string{"pets"},
		Metadata: map[string]any{
			"device": "kitchen",
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
	if record.Text != "my dog's name is Minnie" {
		t.Fatalf("Text = %q, want memory text", record.Text)
	}
	if record.Title != "" {
		t.Fatalf("Title = %q, want empty when title equals text", record.Title)
	}
	if len(record.Tags) != 1 || record.Tags[0] != "pets" {
		t.Fatalf("Tags = %v, want [pets]", record.Tags)
	}
	if record.Metadata["device"] != "kitchen" {
		t.Fatalf("Metadata = %v, want device kitchen", record.Metadata)
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
