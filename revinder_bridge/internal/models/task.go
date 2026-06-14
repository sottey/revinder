package models

import "time"

type Task struct {
	ID               int64      `json:"id"`
	RevinderBridgeID string     `json:"revinder_bridge_id,omitempty"`
	Title            string     `json:"title"`
	Source           string     `json:"source"`
	ListName         string     `json:"list_name"`
	DueAt            *time.Time `json:"due_at"`
	AllDay           bool       `json:"all_day"`
	Notes            *string    `json:"notes"`
	Tags             []string   `json:"tags"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`
}

type CreateTaskRequest struct {
	RevinderBridgeID string     `json:"revinder_bridge_id"`
	Title            string     `json:"title"`
	Source           string     `json:"source"`
	ListName         string     `json:"list_name"`
	DueAt            *time.Time `json:"due_at"`
	AllDay           bool       `json:"all_day"`
	Notes            *string    `json:"notes"`
	Tags             []string   `json:"tags"`
}

type CreateTaskResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type Item struct {
	ID          int64          `json:"id"`
	RevinderID  string         `json:"revinder_id,omitempty"`
	Source      string         `json:"source"`
	Type        string         `json:"type"`
	Text        string         `json:"text"`
	Title       string         `json:"title"`
	Notes       *string        `json:"notes"`
	DueAt       *time.Time     `json:"due_at"`
	Priority    *string        `json:"priority"`
	ListName    string         `json:"list_name"`
	Tags        []string       `json:"tags"`
	Metadata    map[string]any `json:"metadata"`
	Status      string         `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	ProcessedAt *time.Time     `json:"processed_at,omitempty"`
}

type CreateItemRequest struct {
	RevinderID string         `json:"revinder_id"`
	Source     string         `json:"source"`
	Type       string         `json:"type"`
	Text       string         `json:"text"`
	Title      string         `json:"title"`
	Notes      *string        `json:"notes"`
	DueAt      *time.Time     `json:"due_at"`
	Priority   *string        `json:"priority"`
	ListName   string         `json:"list_name"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
}

type CreateItemResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}
