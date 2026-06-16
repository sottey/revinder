package jsonl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sottey/revinder/consumers/revinder_task_consumer/internal/bridge"
)

type Writer struct {
	path string
}

type Record struct {
	ID         int64          `json:"id"`
	RevinderID string         `json:"revinder_id,omitempty"`
	Source     string         `json:"source"`
	Text       string         `json:"text"`
	Title      string         `json:"title,omitempty"`
	Notes      *string        `json:"notes"`
	DueAt      *time.Time     `json:"due_at"`
	Priority   *string        `json:"priority"`
	ListName   string         `json:"list_name"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}

func New(path string) *Writer {
	return &Writer{path: path}
}

func (w *Writer) Process(ctx context.Context, item bridge.Item) error {
	record := Record{
		ID:         item.ID,
		RevinderID: item.RevinderID,
		Source:     item.Source,
		Text:       strings.TrimSpace(item.Text),
		Title:      strings.TrimSpace(item.Title),
		Notes:      item.Notes,
		DueAt:      item.DueAt,
		Priority:   item.Priority,
		ListName:   strings.TrimSpace(item.ListName),
		Tags:       item.Tags,
		Metadata:   item.Metadata,
		CreatedAt:  item.CreatedAt,
	}
	if record.Title == record.Text {
		record.Title = ""
	}
	if record.Tags == nil {
		record.Tags = []string{}
	}
	if record.Metadata == nil {
		record.Metadata = map[string]any{}
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(w.path), 0700); err != nil {
		return err
	}

	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}
