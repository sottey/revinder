package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sottey/revinder_bridge/internal/models"
	"github.com/sottey/revinder_bridge/internal/store"
)

type handler struct {
	store *store.Store
	token string
}

func NewRouter(store *store.Store, token string, logger *slog.Logger) http.Handler {
	h := &handler{store: store, token: token}

	r := chi.NewRouter()
	r.Use(requestLogger(logger))
	r.Get("/health", h.health)

	r.Group(func(r chi.Router) {
		r.Use(h.requireAuth)
		r.Post("/api/items", h.createItem)
		r.Get("/api/items", h.items)
		r.Get("/api/items/pending", h.pendingItems)
		r.Get("/api/items/{id}", h.getItem)
		r.Post("/api/items/{id}/processed", h.markItemProcessed)
		r.Delete("/api/items/{id}", h.deleteItem)
		r.Post("/api/tasks", h.createTask)
		r.Get("/api/tasks/pending", h.pendingTasks)
		r.Get("/api/tasks/synced", h.syncedTasks)
		r.Get("/api/tasks/{id}", h.getTask)
		r.Post("/api/tasks/{id}/synced", h.markSynced)
		r.Post("/api/tasks/{id}/pending", h.markPending)
	})

	return r
}

func (h *handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) createItem(w http.ResponseWriter, r *http.Request) {
	var req models.CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.RevinderID = strings.TrimSpace(req.RevinderID)
	req.Source = strings.TrimSpace(req.Source)
	req.Type = strings.TrimSpace(req.Type)
	req.Text = strings.TrimSpace(req.Text)
	req.Title = strings.TrimSpace(req.Title)
	req.ListName = strings.TrimSpace(req.ListName)
	for i := range req.Tags {
		req.Tags[i] = strings.TrimSpace(req.Tags[i])
	}

	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type required")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text required")
		return
	}
	if req.Title == "" {
		req.Title = req.Text
	}

	resp, err := h.store.CreateItem(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create item")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) items(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.Items(strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get items")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *handler) pendingItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.PendingItems()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get pending items")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *handler) getItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	item, found, err := h.store.GetItem(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get item")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "invalid id")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *handler) markItemProcessed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	updated, err := h.store.MarkItemProcessed(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark item processed")
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "invalid id")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{Success: true})
}

func (h *handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	deleted, err := h.store.DeleteItem(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "invalid id")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{Success: true})
}

func (h *handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Source = strings.TrimSpace(req.Source)
	req.ListName = strings.TrimSpace(req.ListName)
	req.RevinderBridgeID = strings.TrimSpace(req.RevinderBridgeID)
	for i := range req.Tags {
		req.Tags[i] = strings.TrimSpace(req.Tags[i])
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title required")
		return
	}
	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source required")
		return
	}

	resp, err := h.store.CreateTask(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) pendingTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.store.PendingTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get pending tasks")
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *handler) syncedTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.store.SyncedTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get synced tasks")
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (h *handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, found, err := h.store.GetTask(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "invalid id")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *handler) markSynced(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	updated, err := h.store.MarkSynced(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark task synced")
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "invalid id")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{Success: true})
}

func (h *handler) markPending(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	updated, err := h.store.MarkPending(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark task pending")
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "invalid id")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse{Success: true})
}

func (h *handler) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+h.token || h.token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.Info(
				"http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
