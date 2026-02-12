// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Mem0Memory represents a single memory entry returned by the mem0 API.
type Mem0Memory struct {
	ID        string  `json:"id"`
	Memory    string  `json:"memory"`
	UserID    string  `json:"user_id"`
	Score     float64 `json:"score,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
}

// Mem0Message represents a chat message sent to the mem0 API for memory extraction.
type Mem0Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Mem0Client is an HTTP client for the mem0 self-hosted REST API.
type Mem0Client struct {
	apiURL     string
	httpClient *http.Client
}

// NewMem0Client creates a new Mem0Client pointing at the given API URL.
// The URL should be the base URL of the mem0 server (e.g. "http://localhost:8080").
func NewMem0Client(apiURL string) *Mem0Client {
	// Trim trailing slash
	apiURL = strings.TrimRight(apiURL, "/")

	return &Mem0Client{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ping checks if the mem0 server is reachable.
func (c *Mem0Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/", nil)
	if err != nil {
		return fmt.Errorf("mem0 ping: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mem0 ping: server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("mem0 ping: server returned status %d", resp.StatusCode)
	}

	return nil
}

// searchResponse wraps the mem0 search endpoint response.
type searchResponse struct {
	Results []Mem0Memory `json:"results"`
}

// Search queries mem0 for memories relevant to the given query for a specific user.
// Returns up to `limit` results ordered by relevance score.
func (c *Mem0Client) Search(ctx context.Context, query, userID string, limit int) ([]Mem0Memory, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("user_id", userID)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	reqURL := fmt.Sprintf("%s/memories/search?%s", c.apiURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("mem0 search: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mem0 search: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mem0 search: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mem0 search: server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Try parsing as {"results": [...]} first, then as bare array
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err == nil && sr.Results != nil {
		return sr.Results, nil
	}

	var memories []Mem0Memory
	if err := json.Unmarshal(body, &memories); err != nil {
		return nil, fmt.Errorf("mem0 search: failed to parse response: %w", err)
	}

	return memories, nil
}

// addRequest is the request body for the mem0 add endpoint.
type addRequest struct {
	Messages []Mem0Message `json:"messages"`
	UserID   string        `json:"user_id"`
}

// addResponse wraps the mem0 add endpoint response.
type addResponse struct {
	Results []struct {
		ID    string `json:"id"`
		Event string `json:"event"`
		Data  struct {
			Memory string `json:"memory"`
		} `json:"data"`
	} `json:"results"`
}

// Add sends conversation messages to mem0 for automatic memory extraction.
// mem0 will process the messages and extract relevant facts/preferences.
func (c *Mem0Client) Add(ctx context.Context, messages []Mem0Message, userID string) error {
	reqBody := addRequest{
		Messages: messages,
		UserID:   userID,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("mem0 add: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/memories", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("mem0 add: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mem0 add: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mem0 add: server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAll retrieves all memories for a specific user.
func (c *Mem0Client) GetAll(ctx context.Context, userID string) ([]Mem0Memory, error) {
	params := url.Values{}
	params.Set("user_id", userID)

	reqURL := fmt.Sprintf("%s/memories?%s", c.apiURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("mem0 get all: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mem0 get all: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mem0 get all: failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mem0 get all: server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Try parsing as {"results": [...]} first, then as bare array
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err == nil && sr.Results != nil {
		return sr.Results, nil
	}

	var memories []Mem0Memory
	if err := json.Unmarshal(body, &memories); err != nil {
		return nil, fmt.Errorf("mem0 get all: failed to parse response: %w", err)
	}

	return memories, nil
}

// FormatMemoriesForContext formats a list of mem0 memories into a string
// suitable for injection into the agent's system prompt.
func FormatMemoriesForContext(memories []Mem0Memory) string {
	if len(memories) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Semantic Memory (mem0)\n\n")
	sb.WriteString("The following are relevant memories retrieved based on the current conversation:\n\n")
	for _, m := range memories {
		sb.WriteString(fmt.Sprintf("- %s\n", m.Memory))
	}

	return sb.String()
}
