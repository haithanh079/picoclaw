# PicoClaw Project Overview

This document provides a comprehensive overview of the PicoClaw project for developers who need to understand, extend, or contribute to the codebase.

---

## Table of Contents

1. [What is PicoClaw?](#what-is-picoclaw)
2. [Architecture](#architecture)
3. [Project Structure](#project-structure)
4. [Key Components](#key-components)
5. [Data Flow](#data-flow)
6. [Development Guide](#development-guide)
7. [Extending the System](#extending-the-system)
8. [Configuration Reference](#configuration-reference)
9. [Testing & Debugging](#testing--debugging)

---

## What is PicoClaw?

PicoClaw is an **ultra-lightweight personal AI assistant** written in Go, designed to run on minimal hardware ($10 boards, <10MB RAM). It was inspired by [nanobot](https://github.com/HKUDS/nanobot) and refactored from the ground up in Go.

### Key Characteristics

- **Language**: Go 1.24+
- **Memory**: <10MB footprint
- **Startup**: ~1 second on 0.6GHz single core
- **Deployment**: Single binary for RISC-V, ARM64, x86_64
- **LLM Support**: OpenRouter, OpenAI, Anthropic, Zhipu, Groq, Gemini, vLLM (local)

### Core Capabilities

- CLI chat (`picoclaw agent`)
- Multi-channel gateway (Telegram, Discord, QQ, DingTalk, Feishu, WhatsApp, MaixCam)
- Web search (Brave Search API)
- File operations (read, write, edit, list)
- Shell execution
- Scheduled tasks (cron tool)
- Skills system (extensible knowledge via SKILL.md files)
- Voice transcription (Groq Whisper for Telegram/Discord)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           CLI / Gateway Entry                             │
│                    (cmd/picoclaw/main.go)                                 │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Message Bus (pkg/bus)                            │
│  Inbound: Channels → Agent    │    Outbound: Agent → Channels             │
└─────────────────────────────────────────────────────────────────────────┘
         │                                          ▲
         │                                          │
         ▼                                          │
┌─────────────────────┐    ┌─────────────────────────────────────────────┐
│  Channel Manager    │    │            Agent Loop (pkg/agent)              │
│  (pkg/channels)     │    │  • ContextBuilder (system prompt, memory)     │
│  • Telegram        │    │  • SessionManager (history, summarization)    │
│  • Discord         │    │  • ToolRegistry (read_file, exec, web, etc.)   │
│  • QQ, DingTalk    │    │  • LLM Provider (HTTP API)                     │
│  • Feishu, etc.    │    │                                               │
└─────────────────────┘    └─────────────────────────────────────────────┘
         │                                          │
         │                                          │
         └──────────────────────────────────────────┘
```

### Design Principles

1. **Message Bus**: All communication flows through `MessageBus`. Channels publish inbound messages; the agent consumes them and publishes outbound responses. This decouples channels from the agent.
2. **Tool-based reasoning**: The agent uses OpenAI-compatible function calling. Tools are registered in `ToolRegistry` and exposed to the LLM.
3. **Workspace-centric**: All persistent data (sessions, memory, skills, cron) lives under a configurable workspace directory.

---

## Project Structure

```
picoclaw/
├── cmd/picoclaw/          # Entry point, CLI commands
│   └── main.go
├── pkg/                   # Core packages
│   ├── agent/             # Agent loop, context building, summarization
│   ├── bus/               # Message bus (inbound/outbound)
│   ├── channels/          # Channel implementations (Telegram, Discord, etc.)
│   ├── config/            # Configuration loading
│   ├── cron/              # Scheduled jobs
│   ├── heartbeat/         # Heartbeat service
│   ├── logger/            # Logging
│   ├── providers/         # LLM providers (HTTP, model routing)
│   ├── session/           # Session & history management
│   ├── skills/            # Skills loader & installer
│   ├── tools/             # Tool implementations
│   ├── utils/             # Utilities
│   └── voice/             # Voice transcription (Groq Whisper)
├── skills/                # Built-in skills (SKILL.md per skill)
├── assets/                # Images, GIFs for README
├── config.example.json    # Example config
├── Makefile               # Build, install, cross-compile
├── go.mod / go.sum
└── docs/                  # This documentation
```

---

## Key Components

### 1. Agent Loop (`pkg/agent/loop.go`)

The central orchestration layer. It:

- Consumes messages from `MessageBus`
- Builds context (system prompt + history + summary + memory)
- Calls the LLM with tool definitions
- Executes tool calls and feeds results back to the LLM
- Triggers summarization when history exceeds ~75% of context window
- Publishes responses to the bus

**Key types**: `AgentLoop`, `processOptions`, `ContextBuilder`

### 2. Message Bus (`pkg/bus/`)

- `InboundMessage`: channel, sender_id, chat_id, content, session_key, media
- `OutboundMessage`: channel, chat_id, content
- Channels publish inbound; agent publishes outbound
- `ChannelManager` subscribes to outbound and dispatches to channels

### 3. Context Builder (`pkg/agent/context.go`)

Builds the system prompt from:

- Identity (time, runtime, workspace paths)
- Bootstrap files: `AGENTS.md`, `SOUL.md`, `USER.md`, `IDENTITY.md`
- Skills summary (list of skills; agent reads full SKILL.md via `read_file`)
- Memory context (`memory/MEMORY.md`, daily notes)
- Tool summaries

### 4. Tools (`pkg/tools/`)

| Tool          | Purpose                          |
|---------------|----------------------------------|
| `read_file`   | Read files (workspace-scoped)    |
| `write_file`  | Write files                      |
| `edit_file`   | Search-replace in files          |
| `list_dir`    | List directory contents          |
| `exec`        | Run shell commands               |
| `web_search`  | Brave Search API                 |
| `web_fetch`   | Fetch URL content                |
| `message`     | Send message to channel          |
| `spawn`       | Spawn sub-agent for background task |
| `cron`        | Schedule reminders/jobs          |

Tools implement `Tool` interface; optional `ContextualTool` for channel/chatID context.

### 5. Providers (`pkg/providers/`)

- `HTTPProvider`: Calls OpenAI-compatible `/chat/completions` API
- `CreateProvider()`: Routes model names to API keys (OpenRouter, Zhipu, etc.). Explicit mapping via `providers.*.models` takes precedence; falls back to name-based inference.

### 6. Skills (`pkg/skills/`)

Skills are **markdown files** (`SKILL.md`) containing instructions or knowledge. Load order:

1. **Workspace** `{workspace}/skills/{name}/SKILL.md`
2. **Global** `~/.picoclaw/skills/{name}/SKILL.md`
3. **Built-in** `{project}/skills/{name}/SKILL.md`

Workspace overrides global overrides built-in.

### 7. Channels (`pkg/channels/`)

Each channel implements `Channel` interface: `Start`, `Stop`, `Send`, `IsRunning`. Channels subscribe to inbound (user messages) and receive outbound (agent responses) via the bus.

---

## Data Flow

### Gateway Mode

1. User sends message via Telegram/Discord/etc.
2. Channel receives message → `bus.PublishInbound(InboundMessage)`
3. Agent loop: `bus.ConsumeInbound()` → `processMessage()` → `runAgentLoop()`
4. Context built → LLM called → tool calls executed → response
5. Agent → `bus.PublishOutbound(OutboundMessage)`
6. Channel manager dispatches to correct channel → user sees reply

### CLI Mode

1. `picoclaw agent -m "Hello"` → `ProcessDirect()` → same `runAgentLoop()` path
2. No bus; response returned directly to stdout

### System Messages ( Background Tasks )

1. Cron job fires → `bus.PublishInbound` with `channel: "system"`, `chat_id: "origin_channel:origin_chat_id"`
2. Agent processes as system message → `SendResponse: true` → reply sent back to origin channel

---

## Development Guide

### Prerequisites

- Go 1.24+
- Git

### Build

```bash
make build          # Current platform
make build-all      # Linux amd64/arm64/riscv64, Windows amd64
make install        # Install to ~/.local/bin + copy builtin skills
```

### Run

```bash
# After make build
./build/picoclaw onboard
# Edit ~/.picoclaw/config.json with API keys
./build/picoclaw agent -m "Hello"
./build/picoclaw gateway   # Start channels
```

### Config Path

- Config: `~/.picoclaw/config.json`
- Workspace: `~/.picoclaw/workspace` (or `agents.defaults.workspace`)

### Debug Mode

```bash
picoclaw agent --debug -m "Hello"
picoclaw gateway --debug
```

---

## Extending the System

### Adding a New Tool

1. Create `pkg/tools/my_tool.go` implementing `Tool`:

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "Does something useful." }
func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "arg1": map[string]interface{}{"type": "string", "description": "..."},
        },
        "required": []string{"arg1"},
    }
}
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // implementation
}
```

2. Register in `pkg/agent/loop.go` inside `NewAgentLoop()`:

```go
toolsRegistry.Register(&tools.MyTool{})
```

### Adding a New Channel

1. Create `pkg/channels/my_channel.go` implementing `Channel`
2. Add config struct in `pkg/config/config.go`
3. Add to `ChannelsConfig` and `DefaultConfig()`
4. Add init logic in `pkg/channels/manager.go` `initChannels()`

### Adding a New LLM Provider

1. Add provider config in `pkg/config/config.go`
2. Extend `CreateProvider()` in `pkg/providers/http_provider.go` with model routing logic (or add the provider to the explicit `models` lookup phase)

### Adding a Skill

1. Create `skills/{name}/SKILL.md` with optional frontmatter:

```markdown
---
name: my_skill
description: What this skill does
---

# My Skill

Instructions for the AI...
```

2. Skills are auto-discovered; no code changes needed.

---

## Configuration Reference

| Section | Key | Description |
|---------|-----|-------------|
| `agents.defaults` | `workspace` | Workspace path |
| | `model` | LLM model (e.g. `glm-4-7`, `openrouter/...`) |
| | `max_tokens` | Max response tokens |
| | `max_tool_iterations` | Max LLM→tool loop iterations |
| `providers` | `openrouter`, `zhipu`, etc. | `api_key`, `api_base`, `models` (optional, explicit model list) |
| `channels` | `telegram`, etc. | Per-channel config |
| `tools.web.search` | `api_key` | Brave Search API key |
| `gateway` | `host`, `port` | Gateway bind address |

Environment variables override JSON: `PICOCLAW_PROVIDERS_OPENROUTER_API_KEY`, etc.

---

## Testing & Debugging

### Run Tests

```bash
go test ./...
```

### Logging

- `logger.SetLevel(logger.DEBUG)` for verbose logs
- `--debug` flag on agent/gateway commands

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| "no API key configured" | No provider set for model | Set `providers.openrouter.api_key` (or matching provider) |
| Web search "API 配置问题" | No Brave Search key | Add `tools.web.search.api_key` |
| Telegram "Conflict: terminated" | Multiple gateways | Stop other `picoclaw gateway` instances |
| Content filtering errors | Provider policy | Rephrase or use different model |

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `picoclaw onboard` | Initialize config & workspace |
| `picoclaw agent -m "..."` | One-shot message |
| `picoclaw agent` | Interactive chat |
| `picoclaw gateway` | Start channels + cron + heartbeat |
| `picoclaw status` | Show config status |
| `picoclaw cron list/add/remove` | Manage scheduled jobs |
| `picoclaw skills list/install/show` | Manage skills |

---

## Further Reading

- [README.md](../README.md) – User-facing docs, setup, chat apps
- [config.example.json](../config.example.json) – Full config template
- [skills/](../skills/) – Example skill implementations
