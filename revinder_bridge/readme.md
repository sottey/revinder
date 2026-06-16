# revinder_bridge

revinder_bridge is a self-hosted natural-language capture service with a Go API and SQLite backend.

## Current State

revinder_bridge currently provides:

- A Cobra CLI with a `serve` command
- A Chi-based JSON HTTP API
- Generic item capture endpoints
- Legacy task endpoints
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

### Create Item

```http
POST /api/items
Authorization: Bearer <token>
Content-Type: application/json
```

Request:

```json
{
  "revinder_id": "alexa-request-1",
  "source": "alexa",
  "type": "task",
  "text": "on Tuesday at 8pm do that one thing",
  "title": "do that one thing",
  "notes": null,
  "due_at": "2026-06-16T20:00:00-07:00",
  "priority": null,
  "list_name": "Home",
  "tags": ["home", "cottage"],
  "metadata": {
    "due_date": "2026-06-16",
    "due_time": "20:00",
    "all_day": false
  }
}
```

Response:

```json
{
  "id": 1,
  "status": "pending"
}
```

If `revinder_id` is supplied and already exists, the existing item id and status are returned instead of creating a duplicate item.

### Get Items

```http
GET /api/items
Authorization: Bearer <token>
```

Optional status filter:

```http
GET /api/items?status=pending
```

Optional type filter:

```http
GET /api/items?status=pending&type=memory
```

### Get Pending Items

```http
GET /api/items/pending
Authorization: Bearer <token>
```

Returns items with `status = "pending"`.

### Get One Item

```http
GET /api/items/{id}
Authorization: Bearer <token>
```

Returns one item by id.

### Mark Item Processed

```http
POST /api/items/{id}/processed
Authorization: Bearer <token>
```

Sets `status` to `"processed"` and sets `processed_at`.

Response:

```json
{
  "success": true
}
```

### Mark Item Failed

```http
POST /api/items/{id}/failed
Authorization: Bearer <token>
```

Sets `status` to `"failed"`.

Response:

```json
{
  "success": true
}
```

### Delete Item

```http
DELETE /api/items/{id}
Authorization: Bearer <token>
```

Response:

```json
{
  "success": true
}
```

### Legacy Create Task

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
