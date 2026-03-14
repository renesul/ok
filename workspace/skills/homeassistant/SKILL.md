---
name: homeassistant
description: "Control Home Assistant: get states, turn on/off lights/switches, call services, and check entity history."
metadata: {"ok":{"emoji":"🏠"}}
---

# Home Assistant Skill

Use the `home_assistant` tool to control your smart home.

## Configuration (config.json)

```json
{
  "integrations": {
    "home_assistant": {
      "enabled": true,
      "url": "http://homeassistant.local:8123",
      "token": "eyJhbGc..."
    }
  }
}
```

**Get a Long-Lived Access Token**: Home Assistant → Profile → Long-Lived Access Tokens → Create Token.

The URL can be local (`http://homeassistant.local:8123`) or remote (Nabu Casa: `https://abc123.ui.nabu.casa`).

## Get entity state

```json
{"action": "get_state", "entity_id": "light.living_room"}
{"action": "get_state", "entity_id": "sensor.temperature"}
{"action": "get_state", "entity_id": "climate.bedroom"}
```

## List all entities

```json
{"action": "list_states"}
{"action": "list_states", "filter": "light."}
{"action": "list_states", "domain": "switch"}
```

## Call a service

Turn on/off lights:
```json
{"action": "call_service", "domain": "light", "service": "turn_on", "entity_id": "light.living_room"}
{"action": "call_service", "domain": "light", "service": "turn_off", "entity_id": "light.living_room"}
{"action": "call_service", "domain": "light", "service": "turn_on", "entity_id": "light.living_room", "service_data": {"brightness": 128, "color_temp": 4000}}
```

Toggle a switch:
```json
{"action": "call_service", "domain": "switch", "service": "toggle", "entity_id": "switch.fan"}
```

Set thermostat temperature:
```json
{"action": "call_service", "domain": "climate", "service": "set_temperature", "entity_id": "climate.bedroom", "service_data": {"temperature": 22}}
```

Play media:
```json
{"action": "call_service", "domain": "media_player", "service": "media_play", "entity_id": "media_player.living_room"}
```

## Get history

```json
{"action": "get_history", "entity_id": "sensor.temperature", "hours": 24}
{"action": "get_history", "hours": 6}
```

## Common domains and services

| Domain | Services |
|--------|----------|
| `light` | `turn_on`, `turn_off`, `toggle` |
| `switch` | `turn_on`, `turn_off`, `toggle` |
| `climate` | `set_temperature`, `set_hvac_mode` |
| `media_player` | `media_play`, `media_pause`, `volume_set` |
| `cover` | `open_cover`, `close_cover`, `stop_cover` |
| `automation` | `trigger`, `turn_on`, `turn_off` |
| `script` | `turn_on` |
| `scene` | `turn_on` |
