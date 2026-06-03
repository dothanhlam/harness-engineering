package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Mem0Client is a simple REST client for interacting with a Mem0 server.
type Mem0Client struct {
	BaseURL    string
	AgentID    string
	HTTPClient *http.Client
}

// NewMem0Client creates a new Mem0 client.
func NewMem0Client(baseURL, agentID string) *Mem0Client {
	return &Mem0Client{
		BaseURL: baseURL,
		AgentID: agentID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MemoryMessage represents a single message for memory extraction.
type MemoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AddMemoryRequest represents the payload for POST /memories.
type AddMemoryRequest struct {
	Messages []MemoryMessage `json:"messages"`
	AgentID  string          `json:"agent_id,omitempty"`
}

// SearchRequest represents the payload for POST /search.
type SearchRequest struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters,omitempty"`
	TopK    int                    `json:"top_k,omitempty"`
}

// SearchResult represents a single search result from Mem0.
type SearchResult struct {
	ID       string                 `json:"id"`
	Memory   string                 `json:"memory"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AddMemory sends a new memory to the Mem0 server.
func (c *Mem0Client) AddMemory(text string) error {
	reqBody := AddMemoryRequest{
		Messages: []MemoryMessage{
			{
				Role:    "user",
				Content: text,
			},
		},
		AgentID: c.AgentID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal add memory request: %w", err)
	}

	url := fmt.Sprintf("%s/memories", c.BaseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create add memory request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute add memory request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mem0 add memory failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SearchMemories searches the Mem0 server for relevant memories.
func (c *Mem0Client) SearchMemories(query string, topK int) ([]string, error) {
	reqBody := SearchRequest{
		Query: query,
		Filters: map[string]interface{}{
			"agent_id": c.AgentID,
		},
		TopK: topK,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/search", c.BaseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mem0 search failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var results []SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	var memories []string
	for _, res := range results {
		if res.Memory != "" {
			memories = append(memories, res.Memory)
		}
	}

	return memories, nil
}
