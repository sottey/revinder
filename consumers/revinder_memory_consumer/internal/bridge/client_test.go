package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPendingItemsSendsMemoryTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/items" {
			t.Fatalf("path = %s, want /api/items", r.URL.Path)
		}
		if r.URL.RawQuery != "status=pending&type=memory" {
			t.Fatalf("query = %s, want status=pending&type=memory", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", r.Header.Get("Authorization"))
		}

		_ = json.NewEncoder(w).Encode([]Item{
			{ID: 1, Source: "alexa", Type: "memory", Text: "my dog's name is Minnie", Status: "pending"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	items, err := client.PendingItems(context.Background())
	if err != nil {
		t.Fatalf("PendingItems() error = %v", err)
	}
	if len(items) != 1 || items[0].Type != "memory" {
		t.Fatalf("items = %+v, want one memory item", items)
	}
}
