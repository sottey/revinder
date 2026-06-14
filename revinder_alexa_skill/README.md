# revinder Alexa Skill

Custom Alexa Skill for creating tasks in `revinder_bridge`.

## Required Lambda Environment

```text
REVINDER_BRIDGE_BASE_URL=https://your-cloudflare-host.example
REVINDER_BRIDGE_TOKEN=your-home-tasks-token
DEFAULT_TIME_ZONE=America/Los_Angeles
```

`DEFAULT_TIME_ZONE` is optional.

## Example Utterances

```text
Alexa, use revinder to add a task on Tuesday at 8pm do that one thing with tags home and cottage
Alexa, use revinder to add a task do that one thing
Alexa, use revinder to add a task do that one thing with tags home and cottage
```

## Bridge Payload

The Lambda posts to `POST /api/tasks`:

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

