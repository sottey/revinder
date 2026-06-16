# revinder consumers

Consumers are local workers that read pending items from `revinder_bridge`, perform one downstream action, then mark each handled item as processed or failed.

Each consumer is its own component with its own binary, config file, tests, and runtime responsibilities. Consumers should fetch only the item type they own so multiple consumers can run against the same bridge without failing each other's items.

## Current Consumers

| Consumer | Path | Item type | Destination |
| --- | --- | --- | --- |
| `revinder_task_consumer` | `./revinder_task_consumer` | `task` | Apple Reminders or JSONL file |
| `revinder_memory_consumer` | `./revinder_memory_consumer` | `memory` | JSONL file |

## Shared Contract

Consumers read pending items with a type filter:

```http
GET /api/items?status=pending&type=<type>
```

After successful handling, consumers mark the item processed:

```http
POST /api/items/{id}/processed
```

If the consumer owns the item type but cannot handle the item, it marks the item failed:

```http
POST /api/items/{id}/failed
```

Consumers should not mark unrelated item types failed.

## revinder_task_consumer

`revinder_task_consumer` handles `task` items.

It:

- supports a `reminders` target for Apple Reminders
- supports a `jsonl` target for file-based task capture
- marks successful items processed
- marks target failures failed

The default target is `reminders`.

The `reminders` target:

- runs on macOS
- verifies `osascript` is available
- verifies the target Reminders list exists
- creates reminders with title, notes, due date, and all-day due date support

The `jsonl` target:

- runs anywhere Go supports
- requires a configured `jsonl.path`
- writes one task JSON object per line

Run once:

```bash
cd revinder_task_consumer
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_task_consumer --once
```

Run once with the JSONL target:

```bash
cd revinder_task_consumer
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_task_consumer --target jsonl --jsonl-path ./revinder_tasks.jsonl --once
```

## revinder_memory_consumer

`revinder_memory_consumer` handles `memory` items and appends them to a configured JSONL file.

It:

- runs anywhere Go supports
- requires a configured `memory_file`
- writes one JSON object per line
- marks successful writes processed
- marks write failures failed

Run once:

```bash
cd revinder_memory_consumer
REVINDER_BRIDGE_TOKEN=some-long-random-secret ./revinder_memory_consumer --memory-file ./revinder_memory.jsonl --once
```

## Build and Test

From the repository root:

```bash
./test-all.sh
./build-all.sh
```
