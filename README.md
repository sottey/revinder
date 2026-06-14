# revinder

`revinder` is a small framework for capturing tasks through Alexa and syncing them into Apple Reminders.

The ecosystem is split into separate components:

| Component | Path | Purpose |
| --- | --- | --- |
| `revinder_bridge` | `./revinder_bridge` | Self-hosted Go API and SQLite task queue. |
| `revinder_alexa_skill` | `./revinder_alexa_skill` | Alexa Skill and Lambda backend that creates bridge tasks. |

Planned components:

| Component | Path | Purpose |
| --- | --- | --- |
| `revinder_sync` | Not created yet | Local Mac service that reads pending bridge tasks, creates Apple Reminders, then marks tasks synced. |
| `revinder_ops` | Not created yet | Optional deployment/runtime notes, scripts, or service definitions for running the framework. |

## Flow

```text
Alexa Skill: revinder
    -> AWS Lambda
    -> Cloudflare Tunnel
    -> revinder_bridge
    -> SQLite pending tasks
    -> Apple Reminders sync service
    -> Apple Reminders
```

The Alexa Skill creates tasks in `revinder_bridge`. A separate sync service reads pending tasks from the bridge, creates Apple Reminders, then marks those tasks as synced.

## Current Components

## revinder_bridge

`revinder_bridge` is the central task queue.

It provides:

- `POST /api/tasks`
- `GET /api/tasks/pending`
- `GET /api/tasks/synced`
- `GET /api/tasks/{id}`
- `POST /api/tasks/{id}/synced`
- `POST /api/tasks/{id}/pending`

Build and test:

```bash
cd revinder_bridge
go test ./...
go build -o revinder_bridge .
```

Run:

```bash
HOME_TASKS_TOKEN=some-long-random-secret ./revinder_bridge serve
```

Configuration is loaded from `revinder_bridge.json`.

Example local config:

```json
{
  "bind_address": "*",
  "port": 9120,
  "database_path": "revinder_bridge.sqlite"
}
```

## revinder_alexa_skill

`revinder_alexa_skill` contains the Alexa interaction model and Lambda handler.

Example utterance:

```text
Alexa, use revinder to add a task on Tuesday at 8pm do that one thing with tags home and cottage
```

The Lambda sends this shape to `revinder_bridge`:

```json
{
  "revinder_bridge_id": "<Alexa request id>",
  "title": "do that one thing",
  "source": "alexa",
  "list_name": "Home",
  "due_at": "2026-06-16T20:00:00-07:00",
  "all_day": false,
  "notes": null,
  "tags": ["home", "cottage"]
}
```

Package Lambda:

```bash
cd revinder_alexa_skill/lambda
zip -r ../lambda.zip index.js package.json
```

Required Lambda environment:

```text
REVINDER_BRIDGE_BASE_URL=https://your-cloudflare-host.example
REVINDER_BRIDGE_TOKEN=your-home-tasks-token
DEFAULT_TIME_ZONE=America/Los_Angeles
```

`DEFAULT_TIME_ZONE` is optional.

## Future Components

### revinder_sync

`revinder_sync` will be the Apple Reminders sync worker.

Expected responsibilities:

- Run on a Mac with access to Apple Reminders.
- Poll `revinder_bridge` for pending tasks.
- Create reminders in the configured Apple Reminders list.
- Preserve task fields where Apple Reminders supports them:
  - title
  - due date/time
  - notes
  - tags, if supported by the sync implementation
- Mark successfully created bridge tasks as synced.

Expected bridge calls:

```http
GET /api/tasks/pending
POST /api/tasks/{id}/synced
```

Failure handling is not designed yet.

### revinder_ops

`revinder_ops` may hold operational files later.

Possible contents:

- launchd service definitions
- Cloudflare Tunnel notes
- deployment scripts
- local backup notes
- production runbooks

Do not place secrets in this component.

## Public Access

`revinder_bridge` is intended to run locally and be exposed through Cloudflare Tunnel.

The public bridge URL should be used as:

```text
REVINDER_BRIDGE_BASE_URL=https://your-cloudflare-host.example
```

Do not commit real tokens, tunnel credentials, SQLite databases, built binaries, or Lambda deployment zips.

## Repository Notes

Local runtime files are intentionally ignored:

- `configuration_notes.txt`
- `codex.resume`
- `revinder_alexa_skill/lambda.zip`
- `revinder_bridge/revinder_bridge`
- `revinder_bridge/revinder_bridge.sqlite`
- `.DS_Store`
