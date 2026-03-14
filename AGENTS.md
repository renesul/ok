# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build binary (runs generate first); output: build/ok
make test           # Run all tests
make lint           # Run golangci-lint
make fmt            # Format code
make check          # deps + fmt + vet + test (full CI check)
make install        # Build and install to ~/.local/bin
make generate       # Run go generate (required before build/test)
```

Run a single test:
```bash
CGO_ENABLED=0 go test -v -tags stdjson -run TestName ./app/orchestrator/
```

Build flags: `CGO_ENABLED=0`, Go flags: `-v -tags stdjson`. Version info injected via ldflags.

## Architecture

OK — Personal AI. Go-based framework that connects LLMs to messaging channels (Telegram, Discord, Slack, WhatsApp, WebSocket chat). Single binary, no CGO.

### Layered Architecture (strict dependency direction: top imports bottom)

```
main.go                  Entry point: flag parsing + gateway startup (no CLI framework)
internal/startup/        Gateway lifecycle: init, run loop, reload, graceful shutdown, onboarding
internal/appinfo/        Version info, config/home path helpers (ldflags target)
app/orchestrator/        Top-level coordinator: AgentLoop, AgentInstance, Registry
app/planning/            ReAct loop: LLM call → tool calls → observe → repeat
app/execution/           Tool registry (~20 tools: file ops, shell, web, subagent, MCP, skills)
app/output/              Summarization
app/memory/              JSONL session store, RAG (vector embeddings, flat-file)
app/context/             System prompt assembly from IDENTITY.md, SOUL.md, TOOLS.md + RAG
app/routing/             Route resolver, model router (complexity-based), classifier
app/input/               Channel adapters + message bus (bus/ subpackage for event types)
app/types/               Shared interfaces (Features, Classifier, ModelRouter, ThinkingLevel)
providers/               LLM backends: Anthropic (native SDK), OpenAI-compatible (13+ vendors)
internal/                Config, logger, auth, skills, webui, MCP, cron, devices, health, media
```

### Key Design Patterns

- **Provider factory** (`providers/factory_provider.go`): auto-detects vendor from model prefix (e.g., `openai/gpt-5.2`), with round-robin load balancing and structured failover classification
- **Tool interface** (`app/execution/`): `Name()`, `Description()`, `Schema()`, `Execute()` — tools self-register in the registry
- **Message bus** (`app/input/bus/`): typed events (`InboundMessage`, `OutboundMessage`, `OutboundMediaMessage`) decouple channels from orchestration
- **Persona system**: workspace markdown files (IDENTITY.md, SOUL.md, TOOLS.md, USER.md) define agent behavior; ContextBuilder assembles them with mtime-based cache invalidation
- **MCP integration** (`internal/mcp/`): stdio, http, sse transports for external tool servers

### Configuration

- Config file: `~/.ok/config.json` (or `$OK_CONFIG`)
- All fields overridable via `OK_` prefixed env vars (parsed by `caarlos0/env`)
- Workspace: `~/.ok/workspace/` (sessions, memory, state, cron, skills, persona files)
- Example config: `config/config.example.json`

### Gateway

The gateway command starts all channel adapters + web UI. Key ports:
- `18790`: Health check endpoint
- `18800`: Web UI (configurable)

### Testing

Tests use `*_test.go` convention with mock providers and table-driven patterns. Key test files:
- `app/orchestrator/agent_test.go` — core agent loop tests
- `app/routing/router_test.go` — model routing tests
- `providers/fallback_test.go` — provider failover tests
