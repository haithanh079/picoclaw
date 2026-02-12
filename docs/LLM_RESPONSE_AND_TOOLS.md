# LLM Response Format & Tool Execution

This document describes the expected response format from the LLM and how PicoClaw runs tools.

---

## 1. Expected LLM Response Format

PicoClaw uses the **OpenAI-style chat completions** format. The provider sends `POST` to `{api_base}/chat/completions` and expects this response shape:

### Raw API Response (what the provider receives)

```json
{
  "choices": [
    {
      "message": {
        "content": "I'll search for that for you.",
        "tool_calls": [
          {
            "id": "call_abc123xyz",
            "type": "function",
            "function": {
              "name": "web_search",
              "arguments": "{\"query\": \"current weather in Tokyo\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 1200,
    "completion_tokens": 85,
    "total_tokens": 1285
  }
}
```

### Parsed `LLMResponse` Structure

From `pkg/providers/types.go`:

```go
type LLMResponse struct {
    Content      string      // Optional text from the model
    ToolCalls    []ToolCall  // Tool calls to execute
    FinishReason string      // "stop" | "tool_calls"
    Usage        *UsageInfo
}

type ToolCall struct {
    ID        string
    Type      string                 // "function"
    Function  *FunctionCall
    Name      string                 // e.g. "web_search"
    Arguments map[string]interface{}  // Parsed JSON → map
}
```

### Example 1: Direct Answer (No Tools)

```json
{
  "choices": [{
    "message": {
      "content": "2 + 2 equals 4.",
      "tool_calls": null
    },
    "finish_reason": "stop"
  }]
}
```

### Example 2: Single Tool Call

```json
{
  "choices": [{
    "message": {
      "content": "Let me read that file for you.",
      "tool_calls": [{
        "id": "call_xyz789",
        "type": "function",
        "function": {
          "name": "read_file",
          "arguments": "{\"path\": \"/home/user/.picoclaw/workspace/AGENTS.md\"}"
        }
      }]
    },
    "finish_reason": "tool_calls"
  }]
}
```

### Example 3: Multiple Tool Calls

```json
{
  "choices": [{
    "message": {
      "content": "I'll search and then summarize what I find.",
      "tool_calls": [
        {
          "id": "call_001",
          "type": "function",
          "function": {
            "name": "web_search",
            "arguments": "{\"query\": \"PicoClaw AI assistant\"}"
          }
        },
        {
          "id": "call_002",
          "type": "function",
          "function": {
            "name": "cron",
            "arguments": "{\"name\": \"reminder\", \"message\": \"Check report\", \"every_seconds\": 3600}"
          }
        }
      ]
    },
    "finish_reason": "tool_calls"
  }]
}
```

### LLM Arguments Format

- `arguments` is a JSON string.
- PicoClaw parses it into `map[string]interface{}`.
- Keys must match the tool's parameter schema (e.g. `path`, `command`, `query`).

---

## 2. How PicoClaw Runs Tools

### Flow Overview

```
User message
    ↓
Build messages (system + history + user)
    ↓
┌─────────────────────────────────────────────────────────┐
│  LLM iteration loop (max 20 iterations)                  │
│                                                         │
│  1. Call LLM with messages + tool definitions             │
│  2. If LLM returns no tool_calls → break, return content  │
│  3. If LLM returns tool_calls:                           │
│     a. Append assistant message (with tool_calls)        │
│     b. For each tool call:                                │
│        - ExecuteWithContext(name, args, channel, chatID)  │
│        - Append tool result message                       │
│     c. Loop again (LLM sees tool results)                 │
└─────────────────────────────────────────────────────────┘
    ↓
Final content (last non-tool response)
```

### Code Path

1. **Request to LLM** (`pkg/providers/http_provider.go` ~line 70)

   - `POST {api_base}/chat/completions`
   - Body: `model`, `messages`, `tools`, `tool_choice: "auto"`

2. **Response handling** (`pkg/agent/loop.go` ~lines 355–364)

   - If `len(response.ToolCalls) == 0` → return `response.Content` as final answer.
   - Otherwise continue to tool execution.

3. **Tool execution** (`pkg/agent/loop.go` ~lines 401–424)

   ```go
   for _, tc := range response.ToolCalls {
       result, err := al.tools.ExecuteWithContext(ctx, tc.Name, tc.Arguments, opts.Channel, opts.ChatID)
       if err != nil {
           result = fmt.Sprintf("Error: %v", err)
       }
       toolResultMsg := providers.Message{
           Role:       "tool",
           Content:    result,
           ToolCallID: tc.ID,
       }
       messages = append(messages, toolResultMsg)
   }
   ```

4. **Registry execution** (`pkg/tools/registry.go` ~lines 40–83)

   - `tool, ok := r.Get(name)`
   - If tool implements `ContextualTool` interface: `SetContext(channel, chatID)`
   - `result, err := tool.Execute(ctx, args)`

5. **Next iteration**

   - Loop continues with messages + tool results.
   - LLM sees tool results and returns final answer or more tool calls.

### Example Message Sequence

```
[system]  ← Full system prompt
[user]    ← "What's in my AGENTS.md file?"
[assistant] ← content: "Let me read it for you.", tool_calls: [{id: "call_1", name: "read_file", args: {"path": "..."}}]
[tool]    ← tool_call_id: "call_1", content: "# Agent Instructions\n\nYou are a helpful AI assistant..."
[assistant] ← content: "Here's what's in your AGENTS.md file: ..."
```

### Tool Interface

Each tool implements (`pkg/tools/base.go`):

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}  // JSON schema for tool
    Execute(ctx context.Context, args map[string]interface{}) (string, error)
}
```

### Contextual Tools

Tools that need channel/chatID (e.g. `message`, `spawn`) implement `ContextualTool`; the agent calls `SetContext(channel, chatID)` before `Execute` so they can route responses correctly.

---

## Related

- [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md) – Architecture and key components
- [DEVELOPMENT.md](DEVELOPMENT.md) – Adding new tools
