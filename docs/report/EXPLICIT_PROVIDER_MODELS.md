# Enhancement Report: Explicit Provider Models

**Feature:** Explicit model-to-provider mapping via `models` array in provider config  
**Date:** February 2026  
**Status:** Implemented

---

## Overview

PicoClaw now supports explicit model-to-provider mapping through an optional `models` array on each provider in `config.json`. This replaces brittle name-based inference for users who need custom endpoints or models whose names don't follow recognized patterns.

---

## Problem Statement

Previously, PicoClaw inferred the API provider from the model name using hardcoded patterns:

- `groq` or `groq/` prefix → Groq
- `gpt` or `openai/` prefix → OpenAI
- `claude` or `anthropic/` prefix → Anthropic
- `glm`, `zhipu`, `zai` → Zhipu
- etc.

This approach failed when:

1. **Third-party models on Groq**: Groq hosts models like `gemma-7b-it` and `llama2-70b-4096`, which contain no "groq" in the name.
2. **Custom endpoints**: Users wanting to route `gpt-4` through a proxy or custom API base had no way to specify that mapping.
3. **New or non-standard model names**: Any model name that didn't match the patterns could not be used reliably.

---

## Solution

Add an optional `models` array to each provider in `config.json`. PicoClaw checks this list first (explicit mapping) before falling back to name-based inference.

### Resolution Order

1. **Phase 1 – Explicit**: For each provider, if the configured model is in that provider's `models` list and credentials are valid, use that provider.
2. **Phase 2 – Inference**: If no explicit match, use the existing name-based inference logic.

---

## Configuration

### Schema

Each provider can now include a `models` array:

```json
{
  "providers": {
    "groq": {
      "api_key": "gsk_xxx",
      "api_base": "",
      "models": ["gemma-7b-it", "llama2-70b-4096"]
    },
    "openai": {
      "api_key": "your_openai_api_key",
      "api_base": "https://your-custom-endpoint.com/v1",
      "models": ["gpt-4", "gpt-3.5-turbo"]
    },
    "zhipu": {
      "api_key": "YOUR_ZHIPU_API_KEY",
      "api_base": "",
      "models": ["glm-4", "glm-4.7"]
    }
  }
}
```

### Example: Groq with Non-Groq Model Names

```json
{
  "agents": {
    "defaults": {
      "model": "gemma-7b-it"
    }
  },
  "providers": {
    "groq": {
      "api_key": "gsk_xxx",
      "api_base": "",
      "models": ["gemma-7b-it", "llama2-70b-4096"]
    }
  }
}
```

### Example: Custom OpenAI Endpoint

```json
{
  "agents": {
    "defaults": {
      "model": "gpt-4"
    }
  },
  "providers": {
    "openai": {
      "api_key": "your_proxy_key",
      "api_base": "https://your-proxy.com/v1",
      "models": ["gpt-4", "gpt-3.5-turbo"]
    }
  }
}
```

---

## Backward Compatibility

| Config state | Behavior |
|--------------|----------|
| No `models` field | Inference only (unchanged) |
| `models: []` (empty) | Inference only |
| `models: ["gpt-4", ...]` | Explicit mapping used when model matches |

Existing configs without `models` continue to work exactly as before.

---

## Error Handling

If a model is explicitly listed for a provider but that provider lacks credentials:

- **Missing `api_key`**: `"model X is mapped to provider but api_key is not configured"`
- **Missing `api_base`** (for providers like VLLM with no default): `"model X is mapped to provider but api_base is not configured"`

---

## Implementation Details

### Files Modified

| File | Change |
|------|--------|
| `pkg/config/config.go` | Added `Models []string` to `ProviderConfig` |
| `pkg/providers/http_provider.go` | Two-phase lookup: explicit first, then inference |
| `config.example.json` | Added `models` arrays to all providers |
| `README.md` | Documented `models` field in Providers section |
| `docs/PROJECT_OVERVIEW.md` | Updated provider routing and config reference |

### Provider Order for Explicit Lookup

Providers are checked in this order; first match wins:

1. Anthropic  
2. OpenAI  
3. OpenRouter  
4. Groq  
5. Zhipu  
6. VLLM  
7. Gemini  

---

## Related Documentation

- [README.md](../README.md) – User-facing providers section
- [PROJECT_OVERVIEW.md](../PROJECT_OVERVIEW.md) – Architecture and config reference
- [config.example.json](../../config.example.json) – Full config template
