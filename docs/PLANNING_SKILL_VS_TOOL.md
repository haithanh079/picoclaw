# System prompt for extending capabilities

**Purpose:** Use this document as a seed when generating implementation plans for PicoClaw. Reference it when planning new features so the AI can decide whether to build a **skill** or a **tool** to achieve the goal.

When extending PicoClaw, first decide whether the goal requires a **skill** or a **tool**.

---

## Quick Decision

| Goal | Use | Reason |
|------|-----|--------|
| Teach agent how to use an API, CLI, or service | **Skill** | Instructions only; existing tools can do it |
| Add a new callable capability the agent doesn't have | **Tool** | Requires code; no existing tool can do it |

---

## Decision Flow

```
Can the agent accomplish this using ONLY:
  - exec (shell commands)
  - web_fetch (fetch URL content)
  - read_file / write_file / edit_file / list_dir
  - web_search (if configured)
  - message (send to channel)
  - spawn (background subagent)
  - cron (schedule)
    │
    ├─ YES → Implement a SKILL
    │        Add: skills/{name}/SKILL.md
    │        Content: instructions, curl/gh/CLI examples, API usage
    │
    └─ NO  → Implement a TOOL
             Add: pkg/tools/{name}.go
             Register in: pkg/agent/loop.go (NewAgentLoop)
             Implements: Tool interface (Name, Description, Parameters, Execute)
```

---

## When to Build a Skill

**Use a skill when the agent already has tools that can do the job, but needs guidance on how.**

### Criteria

- The task can be done with `exec` (running a CLI like `curl`, `gh`, `tmux`)
- Or with `web_fetch` (fetching a URL and parsing content)
- Or with existing file tools
- You are adding **knowledge** or **usage patterns**, not **new capabilities**

### What to create

- `skills/{name}/SKILL.md` with optional frontmatter:

```markdown
---
name: my_skill
description:One-line description for the agent
---

# Skill Name

Instructions for the agent. Include:
- CLI commands (curl, gh, etc.)
- API endpoints and usage
- Examples with expected output
```

### Examples

| Skill | Why Skill | Uses |
|-------|-----------|------|
| weather | wttr.in via curl | exec |
| github | gh CLI for issues/PRs | exec |
| tmux | tmux commands | exec |
| summarize | How to summarize text | read_file, write_file |

---

## When to Build a Tool

**Use a tool when the agent needs a new capability that no existing tool provides.**

### Criteria

- Requires a dedicated API integration (auth, signing, custom endpoints)
- Requires safe/sandboxed execution that `exec` shouldn't handle
- Requires structured input/output that doesn't fit shell commands
- The capability is a first-class action (e.g. schedule, send message, search)

### What to create

1. `pkg/tools/{name}.go` implementing `Tool`:

```go
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "..." }
func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{...},
        "required": []string{...},
    }
}
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) { ... }
```

2. Register in `pkg/agent/loop.go` inside `NewAgentLoop()`:

```go
toolsRegistry.Register(&tools.MyTool{})
```

### Examples

| Tool | Why Tool | No existing tool can |
|------|----------|----------------------|
| web_search | Brave Search API | Search the web |
| cron | Job scheduling | Schedule reminders/tasks |
| message | Send to channel | Route replies to user |
| spawn | Background subagent | Run async tasks |

---

## Boundary Cases

| Scenario | Recommendation |
|----------|----------------|
| Simple REST API, no auth | **Skill** – use `exec` with `curl` or `web_fetch` |
| API with API key in headers | **Skill** if key can be in env; **Tool** if config/signing is complex |
| Recurring logic (e.g. "every hour") | **Tool** – use `cron` |
| One-off command patterns | **Skill** |

---

## Summary for Plan Generation

When planning an implementation:

1. **Check the decision flow** – Can existing tools do it with instructions?
2. **Skill** → Add `skills/{name}/SKILL.md`; no code changes.
3. **Tool** → Add `pkg/tools/{name}.go` and register in `pkg/agent/loop.go`.
4. **Config** → If the tool needs API keys or options, add to `pkg/config/config.go` and `config.example.json`.
