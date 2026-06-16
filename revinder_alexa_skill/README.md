# revinder Alexa Skill

Custom Alexa Skill for creating captured items in `revinder_bridge`.

## Required Lambda Environment

```text
REVINDER_BRIDGE_BASE_URL=https://your-cloudflare-host.example
REVINDER_BRIDGE_TOKEN=your-home-tasks-token
DEFAULT_TIME_ZONE=America/Los_Angeles
```

`DEFAULT_TIME_ZONE` is optional.

## Example Utterances

```text
Alexa, ask revinder bridge to add a task on Tuesday at 8pm do that one thing with tags home and cottage
Alexa, ask revinder bridge to add a task do that one thing
Alexa, ask revinder bridge to add a task do that one thing with tags home and cottage
Alexa, tell revinder bridge that my dog's name is Minnie
```

## Bridge Payload

The Lambda posts to `POST /api/items`:

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
