# PicoClaw Development Guide

Practical tips and workflows for developers contributing to or extending PicoClaw.

---

## Local Development Setup

### 1. Clone & Build

```bash
git clone https://github.com/sipeed/picoclaw.git
cd picoclaw
make deps    # Update/go get dependencies
make build   # Build to build/picoclaw
```

### 2. Use Local Binary During Development

Instead of `make install`, run directly:

```bash
./build/picoclaw agent -m "Hello"
# or
PICOCLAW_HOME=/tmp/picoclaw-test ./build/picoclaw onboard
PICOCLAW_HOME=/tmp/picoclaw-test ./build/picoclaw agent -m "Test"
```

### 3. Workspace for Development

Use a separate workspace to avoid polluting your main config:

```bash
# In config, set:
# "workspace": "/tmp/picoclaw-dev/workspace"

# Or override via env (need to check if supported)
```

---

## Code Organization Conventions

### Package Responsibilities

| Package | Responsibility |
|---------|----------------|
| `agent` | Agent loop, context building, summarization, session flow |
| `bus` | Message passing (inbound/outbound) |
| `channels` | External integrations (Telegram, Discord, etc.) |
| `config` | Load/save config, env overrides |
| `cron` | Job storage and scheduling |
| `providers` | LLM API calls, model routing |
| `session` | Conversation history, summaries |
| `skills` | Load SKILL.md from workspace/global/builtin |
| `tools` | Tool implementations and registry |

### Adding New Code

- **Tools**: `pkg/tools/` + register in `agent/loop.go`
- **Channels**: `pkg/channels/` + wire in `manager.go`
- **Providers**: extend `providers.CreateProvider()` and config
- **Skills**: add `skills/{name}/SKILL.md` (no code)

---

## Testing Before Submitting

```bash
# Format
make fmt

# Run tests
go test ./...

# Build for all platforms
make build-all

# Sanity check
./build/picoclaw status
./build/picoclaw agent -m "What is 2+2?"
```

---

## Debugging

### Enable Debug Logging

```bash
./build/picoclaw agent --debug -m "Your message"
./build/picoclaw gateway --debug
```

### Inspect System Prompt

Add temporary logging in `pkg/agent/context.go`:

```go
// In BuildSystemPrompt() or BuildMessages()
logger.DebugCF("agent", "System prompt", map[string]interface{}{
    "content": systemPrompt,
})
```

### Trace Tool Calls

Tool execution is logged in `pkg/tools/registry.go` with `logger.InfoCF` / `logger.ErrorCF`.

---

## Cross-Compilation

```bash
# Linux ARM64 (e.g. Raspberry Pi, LicheeRV)
GOOS=linux GOARCH=arm64 go build -o picoclaw-linux-arm64 ./cmd/picoclaw

# Linux RISC-V (e.g. LicheeRV-Nano)
GOOS=linux GOARCH=riscv64 go build -o picoclaw-linux-riscv64 ./cmd/picoclaw
```

Or use `make build-all` for predefined targets.

---

## Dependency Management

```bash
go get -u ./...
go mod tidy
```

---

## Skill Development

### Skill Structure

```
skills/my_skill/
├── SKILL.md    # Required: instructions for the AI
└── (optional)  # Scripts, configs the AI can read
```

### SKILL.md Frontmatter

```markdown
---
name: my_skill
description: One-line description
---

# My Skill

Full instructions...
```

### Testing a Skill

1. Add skill to `skills/` or `~/.picoclaw/workspace/skills/`
2. Run `picoclaw skills list` to verify it's loaded
3. Ask the agent something that requires the skill
4. Agent will use `read_file` to load SKILL.md when needed

---

## Common Extension Patterns

### Pattern: Tool with Channel Context

For tools that need to send replies (e.g. `message`, `spawn`):

1. Implement `ContextualTool` interface
2. Agent calls `SetContext(channel, chatID)` before execution
3. Tool uses context to route responses

### Pattern: Cron Job with Delivery

When a cron job should reply to a channel:

- Job stores `deliver: true`, `channel`, `to`
- Cron tool executes job, gets response
- Response published to bus with correct channel/chatID

---

## File Locations

| Item | Default Path |
|------|--------------|
| Config | `~/.picoclaw/config.json` |
| Workspace | `~/.picoclaw/workspace` |
| Sessions | `{workspace}/sessions/` |
| Memory | `{workspace}/memory/MEMORY.md` |
| Cron jobs | `{workspace}/cron/jobs.json` |
| Skills | `{workspace}/skills/`, `~/.picoclaw/skills/`, `{repo}/skills/` |

---

## Semantic Memory with mem0

PicoClaw supports [mem0](https://github.com/mem0ai/mem0) as an optional semantic memory layer. When enabled, the agent automatically:

1. **Searches** mem0 for relevant memories before each LLM call (based on the user's message)
2. **Captures** new memories after each conversation turn (user message + assistant response)

This works alongside the existing file-based memory (`MEMORY.md` and daily notes), providing vector-based semantic search and automatic fact extraction from conversations.

### Starting the mem0 Server

The easiest way is via the provided Docker Compose file:

```bash
cd docker
OPENAI_API_KEY=sk-... docker compose -f docker-compose.mem0.yml up -d
```

This starts:
- **Qdrant** vector store on port 6333
- **mem0 REST API** on port 8080 (API docs at http://localhost:8080/docs)

mem0 uses OpenAI for embeddings and memory extraction by default. Set your `OPENAI_API_KEY` environment variable accordingly.

### Configuration

Add the `mem0` section to your `~/.picoclaw/config.json`:

```json
{
  "mem0": {
    "enabled": true,
    "api_url": "http://localhost:8080"
  }
}
```

Or use environment variables:

```bash
export PICOCLAW_MEM0_ENABLED=true
export PICOCLAW_MEM0_API_URL=http://localhost:8080
```

### How It Works

- **Search**: Before building the LLM context, PicoClaw queries mem0 with the user's message and retrieves up to 5 relevant memories. These are injected into the system prompt under "Semantic Memory (mem0)".
- **Capture**: After the assistant responds, the conversation (user message + assistant reply) is sent to mem0 asynchronously. mem0 extracts facts and preferences automatically.
- **User ID**: Each chat session maps to a mem0 user ID (derived from the session key, e.g. `telegram_12345`).
- **Fallback**: If mem0 is disabled or unreachable, the agent falls back to file-based memory only. All mem0 errors are non-fatal.

### Verifying mem0 Is Working

Run the agent with `--debug` and look for log messages:
- `mem0 semantic memory enabled` at startup
- `mem0 memories retrieved` when memories are found
- `mem0 memories captured` after each turn

---

## Getting Help

- **Issues**: https://github.com/sipeed/picoclaw/issues
- **Discord**: https://discord.gg/V4sAZ9XWpN
