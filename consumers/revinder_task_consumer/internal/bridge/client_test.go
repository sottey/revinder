package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPendingItemsSendsAuthAndDecodesItems(t *testing.T) {
	dueAt := time.Date(2026, 6, 16, 20, 0, 0, 0, time.FixedZone("PDT", -7*60*60))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/items" {
			t.Fatalf("path = %s, want /api/items", r.URL.Path)
		}
		if r.URL.RawQuery != "status=pending&type=task" {
			t.Fatalf("query = %s, want status=pending&type=task", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", r.Header.Get("Authorization"))
		}

		_ = json.NewEncoder(w).Encode([]Item{
			{
				ID:       1,
				Source:   "alexa",
				Type:     "task",
				Text:     "replace air filter",
				Title:    "replace air filter",
				ListName: "Home",
				DueAt:    &dueAt,
				Tags:     []string{"home"},
				Metadata: map[string]any{"all_day": false},
				Status:   "pending",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", "test-token")
	items, err := client.PendingItems(context.Background())
	if err != nil {
		t.Fatalf("PendingItems() error = %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("item count = %d, want 1", len(items))
	}
	if items[0].ID != 1 || items[0].Title != "replace air filter" {
		t.Fatalf("item = %+v, want id 1 title replace air filter", items[0])
	}
	if items[0].DueAt == nil || !items[0].DueAt.Equal(dueAt) {
		t.Fatalf("due_at = %v, want %v", items[0].DueAt, dueAt)
	}
}

func TestPendingItemsReturnsErrorForNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	if _, err := client.PendingItems(context.Background()); err == nil {
		t.Fatal("PendingItems() error = nil, want error")
	}
}

func TestMarkProcessedSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/items/42/processed" {
			t.Fatalf("path = %s, want /api/items/42/processed", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", r.Header.Get("Authorization"))
		}

		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	if err := client.MarkProcessed(context.Background(), 42); err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
}

func TestMarkProcessedReturnsErrorForNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	if err := client.MarkProcessed(context.Background(), 42); err == nil {
		t.Fatal("MarkProcessed() error = nil, want error")
	}
}

func TestMarkFailedSendsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/items/42/failed" {
			t.Fatalf("path = %s, want /api/items/42/failed", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", r.Header.Get("Authorization"))
		}

		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	if err := client.MarkFailed(context.Background(), 42); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
}

func TestMarkFailedReturnsErrorForNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	if err := client.MarkFailed(context.Background(), 42); err == nil {
		t.Fatal("MarkFailed() error = nil, want error")
	}
}
