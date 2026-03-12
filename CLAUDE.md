# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

OK is an lightweight personal AI assistant framework in Go. It connects LLMs to messaging platforms (Telegram, Discord, WhatsApp, Slack, etc.) via autonomous agents with tools. Single binary, no CGO.

**Repo:** `github.com/renesul/ok` · **License:** MIT · **Go:** 1.25.7 · **Platforms:** linux/amd64, linux/arm64 only

## Build & Development Commands

```bash
make build          # Build for current platform (output: build/ok)
make build-all      # Build linux/amd64 + linux/arm64
make install        # Build + install to ~/.local/bin
make test           # Run all tests (includes go generate)
make lint           # Run golangci-lint
make fmt            # Format code
make vet            # Static analysis
make check          # deps + fmt + vet + test
```

**Run a single test:**
```bash
CGO_ENABLED=0 go test -tags stdjson -run TestName ./app/orchestrator/...
```

**Build flags:** Always use `CGO_ENABLED=0` and `-tags stdjson`. The Makefile sets these via `GO?=CGO_ENABLED=0 go` and `GOFLAGS?=-v -tags stdjson`.

**Go generate:** `make build` automatically runs `go generate ./...` which copies `workspace/` into `cmd/ok/internal/onboard/` for embedding builtin skills into the binary.

## Architecture

### Data Flow

```
                USER
                  │
                  ▼
           AgentLoop.Run()
                  │
     ┌────────────┼────────────┐
     ▼            ▼            ▼
ContextBuilder   Memory      ToolRegistry
     │
     ▼
 Prompt Pipeline
     │
     ├─ System
     ├─ Identity
     ├─ Runtime
     ├─ Summary
     ├─ RAG
     ├─ History
     └─ User message
                  │
                  ▼
             Planner
         (ReAct loop)
                  │
        ┌─────────┼─────────┐
        ▼                   ▼
      LLM              ToolCalls
        │                   │
        │           ToolExecutor
        │                   │
        │            Observations
        │                   │
        └───── loop ◄───────┘
                  │
                  ▼
            Final Answer
                  │
        ┌─────────┼─────────┐
        ▼                   ▼
    Memory Save         Summary
        │
        ▼
     RAG Index
                  │
                  ▼
              ResponseBus
                  │
                  ▼
                USER
```

### Layered Architecture (`app/`)

The pipeline is organized as layered packages under `app/`. Each layer has strict dependency rules (no cycles):

```
app/types/         →  providers/                         (foundation)
app/input/bus/     →  (no app/ deps)                     (foundation)
app/memory/        →  providers/                         (layer 1)
app/output/        →  providers/                         (layer 1)
app/routing/       →  app/types                          (layer 2)
app/execution/     →  app/input/bus                      (layer 2)
app/planning/      →  app/input/bus, app/types           (layer 2)
app/context/       →  app/memory                         (layer 3)
app/input/         →  app/input/bus                      (layer 3)
app/orchestrator/  →  all above                          (top)
```

- **`app/types/`** — Shared interfaces and types (Features, Classifier, ModelRouter, ThinkingLevel, RoutePeer, RouteInput, ResolvedRoute).
- **`app/input/`** — Channel adapters + message bus. Plugin architecture via `init()` → `RegisterFactory()`. Subpackages: bus/, telegram/, discord/, slack/, whatsapp/.
- **`app/routing/`** — Route resolver (7-level priority cascade), model router, classifier, agent ID normalization, session key building.
- **`app/context/`** — Prompt assembly: ContextBuilder, Persona/PersonaLoader, RAGContextCache, MemoryStore.
- **`app/planning/`** — ReAct loop: Planner calls LLM → tool calls → observe → loop. Handles model selection, fallback chain, context recovery.
- **`app/execution/`** — Agent tools (file ops, exec, web, skills, subagent, MCP). Singletons; channel/chatID injected via `context.Value()`.
- **`app/memory/`** — Persistence: JSONL store, session manager, RAG (vector embeddings, retriever, embedder).
- **`app/output/`** — Summarization.
- **`app/orchestrator/`** — Top-level coordinator: AgentLoop, AgentInstance, AgentRegistry, ToolExecutor, MemoryManager, Planner wrapper. Imports all layers above.
- **`providers/`** — LLM backend abstraction. `LLMProvider` interface with `Chat()` method. Subpackages: anthropic/, openai_compat/.
- **`internal/`** — Auxiliary packages (not importable outside this module): config/, logger/, auth/, identity/, media/, mcp/, skills/, health/, heartbeat/, devices/, cron/, voice/, webui/, migrate/, commands/, utils/.

### Key Patterns

**Channel registration:** Each channel has `init.go` calling `channels.RegisterFactory("name", factory)`. The `Manager` instantiates enabled channels from config.

**Tool context injection:** Tools are singletons — per-request state (channel, chatID) comes from `context.Value()`. Access via `ToolChannel(ctx)` and `ToolChatID(ctx)` helpers. This avoids data races.

**Provider selection:** `model_list` entries use `vendor/model` format (e.g. `openai/gpt-5.2`). The vendor prefix determines the protocol (OpenAI-compatible or Anthropic). Multiple entries with the same `model_name` are round-robined for load balancing.

**Tool definitions are sorted** alphabetically when sent to LLMs for deterministic KV cache behavior.

### Entry Points

- **`cmd/ok/main.go`** — CLI entry point (cobra commands: agent, gateway, onboard, cron, skills, etc.)

### Config

- Config dir: `~/.ok/` (override with `OK_HOME`)
- Config file: `~/.ok/config.json` (override with `OK_CONFIG`)
- Workspace: `~/.ok/workspace/` — contains IDENTITY.md, SOUL.md, TOOLS.md, sessions/, memory/, skills/
- Builtin skills root: overridable via `OK_BUILTIN_SKILLS`

### Dependencies (notable)

- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- `spf13/cobra` for CLI
- `openai/openai-go`, `anthropics/anthropic-sdk-go` for LLM providers
- Channel SDKs: `telego` (Telegram), `discordgo`, `slack-go/slack`, etc.
