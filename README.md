<div align="center">

```
  ██████╗ ██╗  ██╗
 ██╔═══██╗██║ ██╔╝
 ██║   ██║█████╔╝
 ██║   ██║██╔═██╗
 ╚██████╔╝██║  ██╗
  ╚═════╝ ╚═╝  ╚═╝
```

### **Personal AI**

Multi-channel AI assistant — one binary, zero config files to edit, everything through the web UI.

[![Go 1.25+](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Platform](https://img.shields.io/badge/Platform-Linux%20amd64%20%7C%20arm64-lightgrey?style=for-the-badge)](/)
[![CGO](https://img.shields.io/badge/CGO-disabled-green?style=for-the-badge)](/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

</div>

---

## ✨ Features

- 🤖 **13+ LLM vendors** — OpenAI, Anthropic, Google Gemini, DeepSeek, Groq, Ollama, OpenRouter, and more
- 💬 **5 chat channels** — Telegram, Discord, WhatsApp, Slack, built-in web chat
- 🖼️ **Vision / image analysis** — dedicated image model routing (Gemini) on any channel that sends photos
- 🎙️ **Voice transcription** — automatic audio-to-text via Groq/Whisper on all channels
- 🖥️ **Web UI** — responsive config editor with i18n (EN/PT-BR/ES), real-time logs, test chat
- 🧠 **RAG** — semantic long-term memory via vector embeddings, flat-file storage
- 🔌 **MCP** — Model Context Protocol support (stdio + HTTP/SSE)
- 🛠️ **Skills** — extensible skill system with built-in defaults
- ⚡ **Agent loop** — ReAct planner + parallel tool execution + sub-agent spawning
- 🔄 **Smart fallback** — automatic load-balancing, cooldown, and failover across providers
- ⏰ **Heartbeat & Cron** — scheduled tasks and periodic agent check-ins
- 🧬 **Persona files** — customize identity, personality, and behavior via markdown
- 🔗 **Integrations** — Email (IMAP/SMTP), Google Calendar, Home Assistant
- 📦 **Single binary** — no CGO, no external dependencies

---

## 🚀 Quick Start

```bash
git clone https://github.com/renesul/OK.git && cd OK
make build && make install

ok              # starts gateway + web UI on http://localhost:18800
ok -version     # show version info
ok -debug       # verbose logging
```

1. Open **http://localhost:18800**
2. Add your LLM API key
3. Enable a channel (Telegram, Discord, WhatsApp, Slack) or use the built-in web chat
4. Done — start chatting 🎉

On first run, OK creates `~/.ok/` with a default config and workspace.

---

## 📋 Requirements

| Requirement | Details |
|---|---|
| **Go** | 1.25+ |
| **CGO** | Disabled (pure Go) |
| **OS** | Linux amd64/arm64 |

---

## ⚙️ Configuration

Config file: `~/.ok/config.json` — edit via web UI or directly.

`OK_HOME` sets the base directory (default `~/.ok`):

```bash
OK_HOME=/srv/ok ok
```

> **Everything else** (models, channels, agents, skills, MCP servers, RAG) is configured through the web UI.

### Minimal Config

```json
{
  "model_list": [
    { "model_name": "gpt-5.2", "model": "openai/gpt-5.2", "api_key": "sk-..." }
  ],
  "agents": { "defaults": { "model": "gpt-5.2" } },
  "channels": {
    "telegram": { "enabled": true, "token": "BOT_TOKEN", "allow_from": ["USER_ID"] }
  }
}
```

### Supported Vendors

All vendors use the OpenAI-compatible HTTP protocol.

**Work out of the box** (just set `api_key`):

| Vendor | Prefix |
|---|---|
| OpenAI | `openai/` |
| Anthropic | `anthropic/` |
| Google Gemini | `google/` |
| DeepSeek | `deepseek/` |
| Groq | `groq/` |
| Mistral | `mistral/` |
| xAI | `xai/` |
| OpenRouter | `openrouter/` |
| NVIDIA | `nvidia/` |
| Cerebras | `cerebras/` |
| Together | `together/` |
| Qwen | `qwen/` |
| Ollama | `ollama/` |

**Any other OpenAI-compatible provider** — set `api_base` in the model config:

```json
{ "model_name": "my-model", "model": "custom/model-id", "api_key": "sk-...", "api_base": "https://my-provider.com/v1" }
```

Multiple entries with the same `model_name` are automatically load-balanced (round-robin).

---

## 🏗️ Architecture

### Message Flow

```
Incoming Message (Telegram, Discord, WhatsApp, Slack, Web Chat)
    │
    ▼
 Channel Adapter (app/input/)
    │  ├─ audio → Voice Transcription (Groq/Whisper)
    │  └─ image → Media Store (downloaded + base64)
    ▼
 Route Resolver (app/routing/)
    │
    ▼
 Agent Instance (app/orchestrator/)
    │
    ▼
 Context Assembly (persona files + RAG)
    │
    ▼
 Model Router
    │  ├─ image detected → Image Provider (Gemini)
    │  ├─ light message  → Light Model (optional)
    │  └─ default        → Primary Provider
    ▼
 ReAct Loop (LLM → tool calls → observe → repeat)
    │
    ▼
 Response → Channel
```

### Project Structure

```
main.go                  Entry point: flag parsing + gateway startup

app/                     Business logic
  orchestrator/          AgentLoop, AgentInstance, Registry
  planning/              ReAct loop: LLM → tool calls → observe → repeat
  execution/             Tool registry (~20 tools)
  memory/                JSONL sessions, RAG (vector embeddings)
  context/               System prompt assembly from persona files + RAG
  routing/               Route resolver, model router
  input/                 Channel adapters + message bus

providers/               LLM backends (Anthropic native + OpenAI-compatible)

internal/                Infrastructure
  startup/               Gateway lifecycle, onboarding, graceful shutdown
  config/                Config loader + hot-reload
  logger/                Structured logging
  auth/                  Authentication
  voice/                 Audio transcription (Groq/Whisper)
  media/                 Media store with TTL cleanup
  heartbeat/             Periodic agent check-ins
  cron/                  Scheduled job execution
  skills/                Skill system
  webui/                 Web UI (embedded SPA)
  mcp/                   MCP client + server
```

### Workspace

```
~/.ok/workspace/
├── sessions/            Conversation history
├── memory/              Long-term memory (RAG-indexed)
├── skills/              Skill packages
├── IDENTITY.md          Agent identity and capabilities
├── SOUL.md              Agent personality and behavior rules
├── USER.md              User preferences
├── AGENTS.md            Multi-agent configuration
├── HEARTBEAT.md         Periodic task checklist
└── TOOLS.md             Tool usage guidelines
```

---

## 🐳 Docker

```bash
docker build -t ok .
docker run -d --name ok -v ~/.ok:/home/ok/.ok -p 18800:18800 ok
```

---

## 🧑‍💻 Development

```bash
make build          # Build binary (output: build/ok)
make test           # Run all tests
make lint           # Run golangci-lint
make fmt            # Format code
make check          # deps + fmt + vet + test (full CI check)
make install        # Build and install to ~/.local/bin
make generate       # Run go generate (required before build/test)
```

---

## 📦 Tech Stack

| Component | Technology |
|---|---|
| **Language** | Go (pure, no CGO) |
| **Storage** | Flat files (JSON, JSONL) |
| **Embeddings** | Vector similarity (flat-file) |
| **LLM Clients** | Anthropic native + OpenAI-compatible |
| **MCP** | stdio + HTTP/SSE transport |
| **Web UI** | Vanilla JS (embedded SPA) |

---

## 📄 Credits

Fork of [PicoClaw](https://github.com/pico-claw/picoclaw), based on [OpenClaw](https://github.com/claw-project/openclaw).

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).

---

<div align="center">

Built with ❤️ using **Go**

</div>
