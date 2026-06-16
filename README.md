# revinder

`revinder` is a small framework for capturing natural-language voice input through Alexa and exposing it as structured items for downstream consumers.

The ecosystem is split into separate components:

| Component | Path | Purpose |
| --- | --- | --- |
| `revinder_bridge` | `./revinder_bridge` | Self-hosted Go API and SQLite capture store. |
| `revinder_alexa_skill` | `./revinder_alexa_skill` | Alexa Skill and Lambda backend that creates bridge items. |
| `revinder_reminders_consumer` | `./consumers/revinder_reminders_consumer` | Local Mac consumer that reads pending bridge items, creates Apple Reminders for task items, then marks items processed. |
| `revinder_memory_consumer` | `./consumers/revinder_memory_consumer` | Local consumer that writes memory items to a configured JSONL file, then marks items processed. |

Planned components:

| Component | Path | Purpose |
| --- | --- | --- |
| `revinder_ops` | Not created yet | Optional deployment/runtime notes, scripts, or service definitions for running the framework. |

## Flow

```text
Alexa Skill: revinder
    -> AWS Lambda
    -> Cloudflare Tunnel
    -> revinder_bridge
    -> SQLite pending items
    -> revinder_reminders_consumer
    -> Apple Reminders

revinder_bridge
    -> SQLite pending memory items
    -> revinder_memory_consumer
    -> JSONL memory file
```

The Alexa Skill creates generic items in `revinder_bridge`. `revinder_reminders_consumer` reads pending task items from the bridge, creates Apple Reminders, then marks those items as processed. `revinder_memory_consumer` reads pending memory items from the bridge, writes them to a configured JSONL file, then marks those items as processed.

## Current Components

## revinder_bridge

`revinder_bridge` is the central capture store.

It provides:

- `POST /api/items`
- `GET /api/items`
- `GET /api/items/pending`
- `GET /api/items/{id}`
- `POST /api/items/{id}/processed`
- `POST /api/items/{id}/failed`
- `DELETE /api/items/{id}`
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
Alexa, ask revinder bridge to add a task on Tuesday at 8pm do that one thing with tags home and cottage
Alexa, tell revinder bridge that my dog's name is Minnie
```

The Lambda sends this item shape to `revinder_bridge`:

```json
{
  "revinder_id": "<Alexa request id>",
  "source": "alexa",
  "type": "task",
  "text": "do that one thing",
  "title": "do that one thing",
  "list_name": "Home",
  "due_at": "2026-06-16T20:00:00-07:00",
  "notes": null,
  "tags": ["home", "cottage"],
  "metadata": {
    "due_date": "2026-06-16",
    "due_time": "20:00",
    "all_day": false
  }
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

## revinder_reminders_consumer

`revinder_reminders_consumer` is the Apple Reminders sync worker and first downstream consumer.

Responsibilities:

- Run on a Mac with access to Apple Reminders.
- Poll `revinder_bridge` for pending task items.
- Create reminders in the configured Apple Reminders list for items where `type = "task"`.
- Preserve task fields where Apple Reminders supports them:
  - title
  - due date/time
  - notes
  - tags, if supported by the sync implementation
- Mark successfully created bridge items as processed.

Bridge calls:

```http
GET /api/items?status=pending&type=task
POST /api/items/{id}/processed
POST /api/items/{id}/failed
```

Build and test:

```bash
cd consumers/revinder_reminders_consumer
go test ./...
go build -o revinder_reminders_consumer ./cmd/revinder_reminders_consumer
```

Run once:

```bash
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_reminders_consumer --once
```

Run continuously:

```bash
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_reminders_consumer
```

Configuration can also be loaded from a JSON config file:

```bash
./revinder_reminders_consumer --config revinder_reminders_consumer.example.json
```

## revinder_memory_consumer

`revinder_memory_consumer` writes memory items to a configured JSONL file.

Responsibilities:

- Poll `revinder_bridge` for pending memory items.
- Append each memory item to the configured output file.
- Mark successfully written bridge items as processed.
- Mark write failures as failed.

Bridge calls:

```http
GET /api/items?status=pending&type=memory
POST /api/items/{id}/processed
POST /api/items/{id}/failed
```

Build and test:

```bash
cd consumers/revinder_memory_consumer
go test ./...
go build -o revinder_memory_consumer ./cmd/revinder_memory_consumer
```

Run once:

```bash
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_memory_consumer --memory-file ./revinder_memory.jsonl --once
```

Run continuously:

```bash
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_memory_consumer --memory-file ./revinder_memory.jsonl
```

Configuration can also be loaded from a JSON config file:

```bash
./revinder_memory_consumer --config revinder_memory_consumer.example.json
```

## Future Components

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
- `consumers/revinder_reminders_consumer/revinder_reminders_consumer`
- `consumers/revinder_memory_consumer/revinder_memory_consumer`
- `.DS_Store`
