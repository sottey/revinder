package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sottey/revinder_bridge/internal/models"
	"github.com/sottey/revinder_bridge/internal/store"
)

func TestHealthDoesNotRequireAuth(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"status": "ok"})
}

func TestProtectedEndpointRequiresAuth(t *testing.T) {
	router := newTestRouter(t)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/items"},
		{http.MethodGet, "/api/items"},
		{http.MethodGet, "/api/items/pending"},
		{http.MethodGet, "/api/items/1"},
		{http.MethodPost, "/api/items/1/processed"},
		{http.MethodPost, "/api/items/1/failed"},
		{http.MethodDelete, "/api/items/1"},
		{http.MethodPost, "/api/tasks"},
		{http.MethodGet, "/api/tasks/pending"},
		{http.MethodGet, "/api/tasks/synced"},
		{http.MethodGet, "/api/tasks/1"},
		{http.MethodPost, "/api/tasks/1/synced"},
		{http.MethodPost, "/api/tasks/1/pending"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestProtectedEndpointDeniesEmptyConfiguredToken(t *testing.T) {
	router := newTestRouterWithToken(t, "")

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/pending", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCreateTaskValidationTitleRequired(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks", []byte(`{"source":"alexa"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "title required"})
}

func TestCreateTaskValidationSourceRequired(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks", []byte(`{"title":"replace air filter"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "source required"})
}

func TestCreateTaskValidationInvalidJSON(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks", []byte(`{`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid JSON"})
}

func TestTaskLifecycle(t *testing.T) {
	router := newTestRouter(t)

	createReq := authorizedRequest(http.MethodPost, "/api/tasks", []byte(`{
		"revinder_bridge_id": " alexa-request-1 ",
		"title": " replace air filter ",
		"source": " alexa ",
		"due_at": "2026-06-15T09:00:00-07:00",
		"all_day": false,
		"notes": null,
		"tags": [" hvac ", " quarterly "]
	}`))
	createRec := httptest.NewRecorder()

	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want %d; body = %s", createRec.Code, http.StatusOK, createRec.Body.String())
	}

	var createResp models.CreateTaskResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if createResp.ID != 1 {
		t.Fatalf("ID = %d, want %d", createResp.ID, 1)
	}
	if createResp.Status != "pending" {
		t.Fatalf("Status = %q, want %q", createResp.Status, "pending")
	}

	pendingReq := authorizedRequest(http.MethodGet, "/api/tasks/pending", nil)
	pendingRec := httptest.NewRecorder()

	router.ServeHTTP(pendingRec, pendingReq)

	if pendingRec.Code != http.StatusOK {
		t.Fatalf("pending status = %d, want %d", pendingRec.Code, http.StatusOK)
	}

	var tasks []models.Task
	if err := json.Unmarshal(pendingRec.Body.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want %d", len(tasks), 1)
	}
	if tasks[0].Title != "replace air filter" {
		t.Fatalf("Title = %q, want %q", tasks[0].Title, "replace air filter")
	}
	if tasks[0].RevinderBridgeID != "alexa-request-1" {
		t.Fatalf("RevinderBridgeID = %q, want %q", tasks[0].RevinderBridgeID, "alexa-request-1")
	}
	if tasks[0].Source != "alexa" {
		t.Fatalf("Source = %q, want %q", tasks[0].Source, "alexa")
	}
	if tasks[0].ListName != "Home" {
		t.Fatalf("ListName = %q, want %q", tasks[0].ListName, "Home")
	}
	if len(tasks[0].Tags) != 2 || tasks[0].Tags[0] != "hvac" || tasks[0].Tags[1] != "quarterly" {
		t.Fatalf("Tags = %v, want %v", tasks[0].Tags, []string{"hvac", "quarterly"})
	}

	getReq := authorizedRequest(http.MethodGet, "/api/tasks/1", nil)
	getRec := httptest.NewRecorder()

	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	var task models.Task
	if err := json.Unmarshal(getRec.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	if task.ID != 1 {
		t.Fatalf("ID = %d, want %d", task.ID, 1)
	}

	syncedReq := authorizedRequest(http.MethodPost, "/api/tasks/1/synced", nil)
	syncedRec := httptest.NewRecorder()

	router.ServeHTTP(syncedRec, syncedReq)

	if syncedRec.Code != http.StatusOK {
		t.Fatalf("synced status = %d, want %d", syncedRec.Code, http.StatusOK)
	}
	assertJSONBody(t, syncedRec.Body.Bytes(), models.SuccessResponse{Success: true})

	emptyPendingReq := authorizedRequest(http.MethodGet, "/api/tasks/pending", nil)
	emptyPendingRec := httptest.NewRecorder()

	router.ServeHTTP(emptyPendingRec, emptyPendingReq)

	if emptyPendingRec.Code != http.StatusOK {
		t.Fatalf("empty pending status = %d, want %d", emptyPendingRec.Code, http.StatusOK)
	}
	assertJSONBody(t, emptyPendingRec.Body.Bytes(), []models.Task{})

	syncedListReq := authorizedRequest(http.MethodGet, "/api/tasks/synced", nil)
	syncedListRec := httptest.NewRecorder()

	router.ServeHTTP(syncedListRec, syncedListReq)

	if syncedListRec.Code != http.StatusOK {
		t.Fatalf("synced list status = %d, want %d", syncedListRec.Code, http.StatusOK)
	}

	var syncedTasks []models.Task
	if err := json.Unmarshal(syncedListRec.Body.Bytes(), &syncedTasks); err != nil {
		t.Fatal(err)
	}
	if len(syncedTasks) != 1 {
		t.Fatalf("len(syncedTasks) = %d, want %d", len(syncedTasks), 1)
	}
	if syncedTasks[0].SyncedAt == nil {
		t.Fatal("SyncedAt = nil, want timestamp")
	}

	pendingResetReq := authorizedRequest(http.MethodPost, "/api/tasks/1/pending", nil)
	pendingResetRec := httptest.NewRecorder()

	router.ServeHTTP(pendingResetRec, pendingResetReq)

	if pendingResetRec.Code != http.StatusOK {
		t.Fatalf("pending reset status = %d, want %d", pendingResetRec.Code, http.StatusOK)
	}
	assertJSONBody(t, pendingResetRec.Body.Bytes(), models.SuccessResponse{Success: true})
}

func TestCreateTaskWithDuplicateRevinderBridgeIDReturnsExistingTask(t *testing.T) {
	router := newTestRouter(t)

	firstReq := authorizedRequest(http.MethodPost, "/api/tasks", []byte(`{
		"revinder_bridge_id": "alexa-request-1",
		"title": "replace air filter",
		"source": "alexa"
	}`))
	firstRec := httptest.NewRecorder()

	router.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body = %s", firstRec.Code, http.StatusOK, firstRec.Body.String())
	}

	var firstResp models.CreateTaskResponse
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstResp); err != nil {
		t.Fatal(err)
	}

	secondReq := authorizedRequest(http.MethodPost, "/api/tasks", []byte(`{
		"revinder_bridge_id": "alexa-request-1",
		"title": "replace furnace filter",
		"source": "alexa"
	}`))
	secondRec := httptest.NewRecorder()

	router.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d; body = %s", secondRec.Code, http.StatusOK, secondRec.Body.String())
	}

	var secondResp models.CreateTaskResponse
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondResp); err != nil {
		t.Fatal(err)
	}
	if secondResp.ID != firstResp.ID {
		t.Fatalf("duplicate ID = %d, want %d", secondResp.ID, firstResp.ID)
	}

	pendingReq := authorizedRequest(http.MethodGet, "/api/tasks/pending", nil)
	pendingRec := httptest.NewRecorder()

	router.ServeHTTP(pendingRec, pendingReq)

	var tasks []models.Task
	if err := json.Unmarshal(pendingRec.Body.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want %d", len(tasks), 1)
	}
	if tasks[0].Title != "replace air filter" {
		t.Fatalf("Title = %q, want %q", tasks[0].Title, "replace air filter")
	}
}

func TestItemLifecycle(t *testing.T) {
	router := newTestRouter(t)

	createReq := authorizedRequest(http.MethodPost, "/api/items", []byte(`{
		"revinder_id": " alexa-request-1 ",
		"source": " alexa ",
		"type": " task ",
		"text": " on Tuesday at 8pm do that one thing ",
		"title": " do that one thing ",
		"due_at": "2026-06-16T20:00:00-07:00",
		"list_name": " Home ",
		"tags": [" home ", " cottage "],
		"metadata": {"device": "kitchen"}
	}`))
	createRec := httptest.NewRecorder()

	router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want %d; body = %s", createRec.Code, http.StatusOK, createRec.Body.String())
	}

	var createResp models.CreateItemResponse
	if err := json.Unmarshal(createRec.Body.Bytes(), &createResp); err != nil {
		t.Fatal(err)
	}
	if createResp.ID != 1 {
		t.Fatalf("ID = %d, want %d", createResp.ID, 1)
	}
	if createResp.Status != "pending" {
		t.Fatalf("Status = %q, want %q", createResp.Status, "pending")
	}

	pendingReq := authorizedRequest(http.MethodGet, "/api/items/pending", nil)
	pendingRec := httptest.NewRecorder()

	router.ServeHTTP(pendingRec, pendingReq)

	if pendingRec.Code != http.StatusOK {
		t.Fatalf("pending status = %d, want %d", pendingRec.Code, http.StatusOK)
	}

	var items []models.Item
	if err := json.Unmarshal(pendingRec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}
	if items[0].RevinderID != "alexa-request-1" {
		t.Fatalf("RevinderID = %q, want %q", items[0].RevinderID, "alexa-request-1")
	}
	if items[0].Text != "on Tuesday at 8pm do that one thing" {
		t.Fatalf("Text = %q, want %q", items[0].Text, "on Tuesday at 8pm do that one thing")
	}
	if items[0].Title != "do that one thing" {
		t.Fatalf("Title = %q, want %q", items[0].Title, "do that one thing")
	}
	if len(items[0].Tags) != 2 || items[0].Tags[0] != "home" || items[0].Tags[1] != "cottage" {
		t.Fatalf("Tags = %v, want %v", items[0].Tags, []string{"home", "cottage"})
	}

	getReq := authorizedRequest(http.MethodGet, "/api/items/1", nil)
	getRec := httptest.NewRecorder()

	router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRec.Code, http.StatusOK)
	}

	processedReq := authorizedRequest(http.MethodPost, "/api/items/1/processed", nil)
	processedRec := httptest.NewRecorder()

	router.ServeHTTP(processedRec, processedReq)

	if processedRec.Code != http.StatusOK {
		t.Fatalf("processed status = %d, want %d", processedRec.Code, http.StatusOK)
	}
	assertJSONBody(t, processedRec.Body.Bytes(), models.SuccessResponse{Success: true})

	failedReq := authorizedRequest(http.MethodPost, "/api/items/1/failed", nil)
	failedRec := httptest.NewRecorder()

	router.ServeHTTP(failedRec, failedReq)

	if failedRec.Code != http.StatusOK {
		t.Fatalf("failed status = %d, want %d", failedRec.Code, http.StatusOK)
	}
	assertJSONBody(t, failedRec.Body.Bytes(), models.SuccessResponse{Success: true})

	emptyPendingReq := authorizedRequest(http.MethodGet, "/api/items/pending", nil)
	emptyPendingRec := httptest.NewRecorder()

	router.ServeHTTP(emptyPendingRec, emptyPendingReq)

	if emptyPendingRec.Code != http.StatusOK {
		t.Fatalf("empty pending status = %d, want %d", emptyPendingRec.Code, http.StatusOK)
	}
	assertJSONBody(t, emptyPendingRec.Body.Bytes(), []models.Item{})

	deleteReq := authorizedRequest(http.MethodDelete, "/api/items/1", nil)
	deleteRec := httptest.NewRecorder()

	router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusOK)
	}
	assertJSONBody(t, deleteRec.Body.Bytes(), models.SuccessResponse{Success: true})
}

func TestCreateItemWithDuplicateRevinderIDReturnsExistingItem(t *testing.T) {
	router := newTestRouter(t)

	firstReq := authorizedRequest(http.MethodPost, "/api/items", []byte(`{
		"revinder_id": "alexa-request-1",
		"source": "alexa",
		"type": "task",
		"text": "replace air filter"
	}`))
	firstRec := httptest.NewRecorder()

	router.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body = %s", firstRec.Code, http.StatusOK, firstRec.Body.String())
	}

	var firstResp models.CreateItemResponse
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstResp); err != nil {
		t.Fatal(err)
	}

	secondReq := authorizedRequest(http.MethodPost, "/api/items", []byte(`{
		"revinder_id": "alexa-request-1",
		"source": "alexa",
		"type": "task",
		"text": "replace furnace filter"
	}`))
	secondRec := httptest.NewRecorder()

	router.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d; body = %s", secondRec.Code, http.StatusOK, secondRec.Body.String())
	}

	var secondResp models.CreateItemResponse
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondResp); err != nil {
		t.Fatal(err)
	}
	if secondResp.ID != firstResp.ID {
		t.Fatalf("duplicate ID = %d, want %d", secondResp.ID, firstResp.ID)
	}
}

func TestCreateItemValidation(t *testing.T) {
	router := newTestRouter(t)

	for _, tc := range []struct {
		name string
		body string
		want map[string]string
	}{
		{
			name: "source required",
			body: `{"type":"task","text":"replace air filter"}`,
			want: map[string]string{"error": "source required"},
		},
		{
			name: "type required",
			body: `{"source":"alexa","text":"replace air filter"}`,
			want: map[string]string{"error": "type required"},
		},
		{
			name: "text required",
			body: `{"source":"alexa","type":"task"}`,
			want: map[string]string{"error": "text required"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := authorizedRequest(http.MethodPost, "/api/items", []byte(tc.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			assertJSONBody(t, rec.Body.Bytes(), tc.want)
		})
	}
}

func TestMarkSyncedInvalidID(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks/999/synced", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid id"})
}

func TestMarkSyncedInvalidIDFormat(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks/not-a-number/synced", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid task id"})
}

func TestGetTaskInvalidID(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodGet, "/api/tasks/999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid id"})
}

func TestGetTaskInvalidIDFormat(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodGet, "/api/tasks/not-a-number", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid task id"})
}

func TestMarkPendingInvalidID(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks/999/pending", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid id"})
}

func TestMarkPendingInvalidIDFormat(t *testing.T) {
	router := newTestRouter(t)

	req := authorizedRequest(http.MethodPost, "/api/tasks/not-a-number/pending", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	assertJSONBody(t, rec.Body.Bytes(), map[string]string{"error": "invalid task id"})
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	return newTestRouterWithToken(t, "test-token")
}

func newTestRouterWithToken(t *testing.T, token string) http.Handler {
	t.Helper()

	db, err := store.Open(filepath.Join(t.TempDir(), "revinder_bridge.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(db, token, logger)
}

func authorizedRequest(method string, target string, body []byte) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func assertJSONBody[T any](t *testing.T, body []byte, want T) {
	t.Helper()

	var got T
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("body = %s, want %s", gotJSON, wantJSON)
	}
}
