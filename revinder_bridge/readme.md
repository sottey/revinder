# revinder_bridge

revinder_bridge is a self-hosted task capture service. The first milestone is a Go API with a SQLite backend.

## Current State

revinder_bridge currently provides:

- A Cobra CLI with a `serve` command
- A Chi-based JSON HTTP API
- A SQLite backend using `github.com/mattn/go-sqlite3`
- Automatic config creation
- Automatic SQLite schema creation
- Bearer-token authentication for API routes
- Graceful HTTP shutdown
- Structured request logging

## Build

```bash
go build -o revinder_bridge .
```

## Run

```bash
HOME_TASKS_TOKEN=some-long-random-secret ./revinder_bridge serve
```

On startup, revinder_bridge creates these files beside the binary when needed:

- `revinder_bridge.json`
- `revinder_bridge.sqlite`

## Configuration

If `revinder_bridge.json` does not exist, it is created with:

```json
{
  "bind_address": "*",
  "port": 8080,
  "database_path": "revinder_bridge.sqlite"
}
```

Fields:

| Field | Default | Description |
| --- | --- | --- |
| `bind_address` | `*` | IP address to bind. `*` binds all interfaces. |
| `port` | `8080` | HTTP server port. |
| `database_path` | `revinder_bridge.sqlite` | SQLite database path. Relative paths are resolved beside the binary. |

## Authentication

All API endpoints except `GET /health` require:

```http
Authorization: Bearer <token>
```

The token comes from:

```bash
HOME_TASKS_TOKEN=some-long-random-secret
```

## API

All `/api/*` routes require:

```http
Authorization: Bearer <token>
```

### Health

```http
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

### Create Task

```http
POST /api/tasks
Authorization: Bearer <token>
Content-Type: application/json
```

Request:

```json
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

```json
{
  "id": 1,
  "status": "pending"
}
```

If `revinder_bridge_id` is supplied and already exists, the existing task id and status are returned instead of creating a duplicate task.

### Get Pending Tasks

```http
GET /api/tasks/pending
Authorization: Bearer <token>
```

Returns tasks with `status = "pending"`.

### Get Synced Tasks

```http
GET /api/tasks/synced
Authorization: Bearer <token>
```

Returns tasks with `status = "synced"`.

### Get One Task

```http
GET /api/tasks/{id}
Authorization: Bearer <token>
```

Returns one task by id.

### Mark Task Synced

```http
POST /api/tasks/{id}/synced
Authorization: Bearer <token>
```

Response:

```json
{
  "success": true
}
```

Invalid task IDs return:

```json
{
  "error": "invalid id"
}
```

### Mark Task Pending

```http
POST /api/tasks/{id}/pending
Authorization: Bearer <token>
```

Sets `status` back to `"pending"` and clears `synced_at`.

Response:

```json
{
  "success": true
}
```

## Test

```bash
go test ./...
```
