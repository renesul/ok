# OK

A single Go binary that turns any LLM into an autonomous personal assistant.

28 tools. 5 channels. 3 security tiers. Zero external dependencies at runtime.

---

## How It Works

You send a message. The agent decides the path:

1. **Direct answer** — plain conversation, no tools involved
2. **Single execution** — one tool, one result
3. **Autonomous plan** — multiple tools in sequence: OBSERVE, PLAN, ACT, REFLECT

Everything streams in real time over WebSocket. The `/agent` workspace is the central monitor — it shows processing from every channel simultaneously.

---

## Channels

| Channel | Integration |
|---------|-------------|
| **Web** | Chat + live workspace |
| **WhatsApp** | whatsmeow adapter |
| **Telegram** | telegram-bot-api |
| **Discord** | discordgo |
| **CLI** | Interactive terminal |

All channels feed the workspace via global event broadcast.

---

## Tools (28)

**Safe** — execute immediately

`echo` `math` `timestamp` `json_parse` `base64` `text_extract` `file_read` `search` `schedule` `learn_rule` `skill_loader`

**Restricted** — input validated, sandboxed

`http` `file_write` `folder_index` `browser` `delegate` `web_search` `skill_creator` `config_manager` `gcal_manager` `gmail_read` `gmail_send` `sql_inspector` `python_rpa`

**Dangerous** — requires human confirmation with preview

`shell` `file_edit` `repl` `docker_replicator`

Every dangerous tool passes through: Safety Gate → Rate Limiter → Confirmation → Audit Log.

---

## Skills System

Expandable at runtime via `.md` files. Teach the agent new capabilities without writing code.

- **skill_loader** — loads skill definitions on demand
- **skill_creator** — generates new skills from conversation context
- **Auto-learn / Auto-forget** — the agent creates and removes rules based on your feedback

---

## Browser

Headless browser with 7 action types:

`click` `fill` `js` `screenshot` `text` `analyze` `wait`

Panic recovery on every action. Runs sandboxed inside the restricted tier.

---

## LLM Configuration

Three independent LLM slots, each with its own provider:

| Slot | Purpose |
|------|---------|
| **Primary** | Planning, reflection, complex reasoning |
| **Fast** | Quick decisions, direct answers, triage |
| **Vision** | Image analysis, screenshots |

Set `USE_NATIVE_TOOLS=true` to enable native function calling instead of JSON-based tool dispatch.

---

## Security

**SecretScrubber** — strips AWS keys, OpenAI tokens, JWTs, RSA keys, Bearer tokens, GitHub/Slack secrets before anything reaches the LLM.

**Shell — 3 tiers:**
- **Tier 1** (blocked): `rm -rf /`, `dd if=/dev`, `mkfs`, fork bombs, writes to `/dev /etc /proc /sys`
- **Tier 2** (confirmation required): `sudo`, `rm -rf`, `chmod`, `chown`, `kill`, `shutdown`
- **Tier 3** (allowed): pipes, redirects, subshells

**Network** — `http` tool blocks localhost, private IPs (10.x, 192.168.x, 172.16.x).

**Rate limits** — shell 5/min, http 20/min, file_write 10/min.

**Defenses** — panic recovery on all adapters, 3min/5min timeouts, 100MB zip bomb protection.

---

## Quick Start

```bash
# Web server
go run cmd/ok/main.go

# Interactive CLI
go run cmd/cli/main.go
```

### Required (`data/.env`)

```env
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4.1-mini
AUTH_PASSWORD=your-password
```

### Optional

```env
# Fast LLM
LLM_FAST_BASE_URL=https://api.openai.com/v1
LLM_FAST_API_KEY=sk-...
LLM_FAST_MODEL=gpt-4.1-nano

# Vision LLM
VISION_BASE_URL=https://api.openai.com/v1
VISION_API_KEY=sk-...
VISION_MODEL=gpt-4.1-mini

# Embeddings
EMBED_PROVIDER=openai
EMBED_BASE_URL=https://api.openai.com/v1
EMBED_API_KEY=sk-...
EMBED_MODEL=text-embedding-3-small

# Native function calling
USE_NATIVE_TOOLS=true

# Adapters
WHATSAPP_OWNER_NUMBER=5511999999999
WHATSAPP_DB_PATH=data/whatsapp.db
TELEGRAM_BOT_TOKEN=...
TELEGRAM_OWNER_ID=123456789
DISCORD_BOT_TOKEN=...
DISCORD_OWNER_ID=...
```

---

## Tests

500+ tests across 18 packages. 122 stress scenarios with real LLMs.

```bash
# Unit tests (no LLM needed)
go test ./domain/... ./application/... ./adapters/... ./infrastructure/... ./interfaces/... -count=1

# Integration tests (needs SQLite)
go test ./tests/integration/... -count=1

# Full E2E stress (needs configured LLM)
go test ./tests/integration/... -run TestStressE2E -timeout=25m -count=1
```

### E2E Batteries (18)

| Battery | Scenarios | Requires |
|---------|-----------|----------|
| FullBattery | 15 | LLM |
| ToolChaining | 8 | LLM |
| Security | 10 | LLM |
| Resilience | 10 | LLM |
| DirectMode | 8 | LLM |
| Memory | 8 | LLM |
| FTS5 | 5 | — |
| ChatFlow | 6 | LLM |
| Concurrent | 2 | LLM |
| Offline | 13 | — |
| Delegate | 1 | LLM |
| SemanticSearch | 1 | Embedding |
| ConcurrentImports | 1 | — |
| SchedulerCRUD | 8 | — |
| Confirmation | 4 | LLM |
| SkillBattery | 8 | LLM |
| BrowserActions | 8 | LLM |
| ConfigTool | 6 | — |

---

## API

### Auth
```
POST /api/auth/login         { password }
POST /api/auth/logout
```

### Conversations
```
GET    /api/conversations
POST   /api/conversations                { title }
GET    /api/conversations/search?q=term
GET    /api/conversations/:id/messages
POST   /api/conversations/:id/messages   { content }  → SSE stream
DELETE /api/conversations/:id
```

### Agent
```
POST /api/agent/run                      { input }    → sync
POST /api/agent/stream                   { input }    → SSE stream
GET  /api/agent/status
GET  /api/agent/tools
GET  /api/agent/skills
GET  /api/agent/executions
GET  /api/agent/executions/:id
GET  /api/agent/metrics
GET  /api/agent/limits
PUT  /api/agent/limits                   { max_steps, max_attempts, timeout_ms }
GET  /api/agent/config/:key
PUT  /api/agent/config/:key              { value }
POST /api/agent/confirm/:id              { approved }
POST /api/agent/cancel
```

### Scheduler
```
GET    /api/scheduler/jobs
POST   /api/scheduler/jobs               { name, task_type, input, interval_seconds }
PUT    /api/scheduler/jobs/:id           { enabled, interval_seconds }
DELETE /api/scheduler/jobs/:id
```

### Import
```
POST /api/import/chatgpt                 multipart/form-data (zip, sharded)
```

### WebSocket
```
GET /ws/agent                            bidirectional, JSON
```

### Health
```
GET /health
```

---

## Architecture

```
cmd/
  ok/              web server
  cli/             interactive terminal
domain/            entities and interfaces
application/       services + agent engine
  engine/          OBSERVE / PLAN / ACT / REFLECT loop
infrastructure/
  agent/           planner, executor, safety, memory, tools (28)
  database/        pure SQLite (database/sql)
  llm/             OpenAI-compatible client
  embedding/       OpenAI / Ollama embeddings
  scheduler/       background job runner
  security/        SecretScrubber
  repository/      conversation, message, session
  bootstrap/       dependency wiring
interfaces/
  http/            Fiber server, handlers, middlewares
adapters/          WhatsApp, Telegram, Discord, CLI
web/               HTML templates, CSS, vanilla JS
```

### Stack

| Layer | Technology |
|-------|------------|
| Language | Go (CGO_ENABLED=0, single binary) |
| Database | SQLite WAL mode (modernc, pure Go) |
| HTTP | Fiber |
| DB access | database/sql (no ORM) |
| Config | Viper |
| Logging | Zap (structured) |
| Vector search | chromem-go (in-memory semantic search) |
| Frontend | HTML + CSS + JS vanilla (zero frameworks) |

---

## License

MIT
