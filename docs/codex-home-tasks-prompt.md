# Codex Prompt: Home Tasks Alexa → Apple Reminders System

## Goal

Build a self-hosted task capture system that allows tasks spoken to an
Alexa Skill to ultimately appear in Apple Reminders.

The current implementation is the Strongbox API and SQLite backend. The
Alexa Skill and Apple Reminders sync components will be added later.

------------------------------------------------------------------------

## Architecture

``` text
Alexa Skill
    ↓
Go API on Strongbox
    ↓
SQLite
    ↓
Mac sync service on Doc
    ↓
Apple Reminders list: Home
```

------------------------------------------------------------------------

## Technical Requirements

### Language

-   Go
-   Latest stable Go version

### Libraries

-   Chi router
-   SQLite (github.com/mattn/go-sqlite3)

### Design Goals

-   Single binary
-   Cobra CLI
-   No ORM
-   JSON API
-   Auto-create database schema on startup
-   Graceful shutdown
-   Structured logging
-   Clear separation between:
    -   HTTP layer
    -   Database layer
    -   Models

------------------------------------------------------------------------

## Runtime Configuration

The server runs with:

``` bash
HOME_TASKS_TOKEN=some-long-random-secret ./revinder_bridge serve
```

Configuration is loaded from `revinder_bridge.json` beside the binary. If the
file does not exist, it is created automatically with:

``` json
{
  "bind_address": "*",
  "port": 8080,
  "database_path": "revinder_bridge.sqlite"
}
```

Fields:

-   `bind_address`: defaults to `*`
-   `port`: defaults to `8080`
-   `database_path`: defaults to `revinder_bridge.sqlite`

Relative database paths are resolved beside the binary.

------------------------------------------------------------------------

## Database

Create SQLite database automatically if it does not exist.

### Schema

``` sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    revinder_bridge_id TEXT,
    title TEXT NOT NULL,
    source TEXT NOT NULL,
    list_name TEXT NOT NULL DEFAULT 'Home',
    due_at DATETIME,
    all_day INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    tags TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL,
    synced_at DATETIME
);
```

`revinder_bridge_id` is optional. When supplied, it is unique and prevents duplicate task creation from retried clients.

------------------------------------------------------------------------

## Authentication

All API endpoints except health checks require:

``` http
Authorization: Bearer <token>
```

Token should be supplied via environment variable:

``` bash
HOME_TASKS_TOKEN=some-long-random-secret
```

Return HTTP 401 when token is invalid.

------------------------------------------------------------------------

## API Endpoints

### Health Check

``` http
GET /health
```

Response:

``` json
{
  "status": "ok"
}
```

------------------------------------------------------------------------

### Create Task

``` http
POST /api/tasks
```

Request:

``` json
{
  "revinder_bridge_id": "alexa-request-1",
  "title": "replace air filter",
  "source": "alexa",
  "list_name": "Home",
  "due_at": "2026-06-15T09:00:00-07:00",
  "all_day": false,
  "notes": null,
  "tags": ["hvac"]
}
```

Response:

``` json
{
  "id": 1,
  "status": "pending"
}
```

Validation:

-   title required
-   source required
-   list_name defaults to Home if omitted
-   revinder_bridge_id prevents duplicates when supplied

------------------------------------------------------------------------

### Get Pending Tasks

``` http
GET /api/tasks/pending
```

Returns:

``` json
[
  {
    "id": 1,
    "revinder_bridge_id": "alexa-request-1",
    "title": "replace air filter",
    "source": "alexa",
    "list_name": "Home",
    "due_at": "2026-06-15T09:00:00-07:00",
    "all_day": false,
    "notes": null,
    "tags": ["hvac"],
    "status": "pending",
    "created_at": "2026-06-12T16:00:00-07:00"
  }
]
```

Only return tasks where:

``` sql
status = 'pending'
```

------------------------------------------------------------------------

### Get Synced Tasks

``` http
GET /api/tasks/synced
```

Returns tasks where:

``` sql
status = 'synced'
```

------------------------------------------------------------------------

### Get Task

``` http
GET /api/tasks/{id}
```

Returns one task by id.

Invalid id format returns HTTP 400.
Missing task id returns HTTP 404.

------------------------------------------------------------------------

### Mark Task Synced

``` http
POST /api/tasks/{id}/synced
```

Behavior:

-   Set status='synced'
-   Set synced_at=current timestamp

Response:

``` json
{
  "success": true
}
```

------------------------------------------------------------------------

### Mark Task Pending

``` http
POST /api/tasks/{id}/pending
```

Behavior:

-   Set status='pending'
-   Set synced_at=NULL

Response:

``` json
{
  "success": true
}
```

------------------------------------------------------------------------

## Tests

The project includes tests for:

-   configuration loading and default config creation
-   SQLite schema/database creation
-   task create/read/status transitions
-   HTTP auth and validation behavior

Run tests with:

``` bash
go test ./...
```

------------------------------------------------------------------------

## Future Integration Notes

Do not implement yet, but design cleanly for:

### Alexa Skill

Future endpoint:

``` http
POST /api/alexa
```

Intent payloads will eventually create tasks.

### Apple Reminders Sync

A separate Mac service running on "Doc" will:

1.  Call:

``` http
GET /api/tasks/pending
```

2.  Create reminders in Apple Reminders list:

``` text
Home
```

3.  Call:

``` http
POST /api/tasks/{id}/synced
```

after successful creation.

------------------------------------------------------------------------

## Current Project Layout

The current project includes:

``` text
main.go
cmd/
internal/
    config/
    httpapi/
    models/
    store/
docs/
readme.md
revinder_bridge.json
revinder_bridge.sqlite
```

Documentation is maintained in `readme.md` and `docs/`.
