---
name: calendar
description: "Manage Google Calendar and Microsoft Outlook events: list upcoming, create, and delete events."
metadata: {"ok":{"emoji":"📅"}}
---

# Calendar Skill

Use the `calendar` tool to manage your calendar.

## Configuration (config.json)

### Google Calendar (API Key — read-only public calendars, or use OAuth token)
```json
{
  "integrations": {
    "calendar": {
      "enabled": true,
      "google_enabled": true,
      "google_api_key": "AIza...",
      "google_calendar_id": "primary"
    }
  }
}
```

> For full read/write access to private calendars, use a Google OAuth2 access token instead of API key. Set `google_api_key` to the Bearer token and the tool will use it.

### Microsoft Outlook
```json
{
  "integrations": {
    "calendar": {
      "enabled": true,
      "outlook_enabled": true,
      "outlook_access_token": "eyJ..."
    }
  }
}
```

Get an Outlook token via Azure AD: register an app with `Calendars.ReadWrite` scope and obtain a token via OAuth2 device flow.

## List upcoming events

```json
{"action": "list", "days": 7}
{"action": "list", "provider": "outlook", "days": 14, "max_results": 50}
```

## Create an event

```json
{
  "action": "create",
  "title": "Team standup",
  "start": "2026-03-15T10:00:00",
  "end": "2026-03-15T10:30:00",
  "timezone": "America/Sao_Paulo",
  "description": "Daily sync",
  "location": "Google Meet"
}
```

## Delete an event

```json
{"action": "delete", "event_id": "abc123..."}
```

The event ID comes from the `id` field in the list action response.

## Tips
- Use `list` first to find event IDs before deleting
- Timezone format: IANA names like `America/Sao_Paulo`, `Europe/London`, `UTC`
- For all-day events, use date-only format: `2026-03-15T00:00:00`
