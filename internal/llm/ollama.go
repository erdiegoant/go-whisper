package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient sends transcripts to a local Ollama instance for cleanup.
type OllamaClient struct {
	model   string
	host    string
	timeout time.Duration
	http    *http.Client
}

// NewOllama creates an OllamaClient. host should be "http://localhost:11434".
func NewOllama(model, host string, timeoutSeconds int) *OllamaClient {
	return &OllamaClient{
		model:   model,
		host:    host,
		timeout: time.Duration(timeoutSeconds) * time.Second,
		http:    &http.Client{},
	}
}

// Process sends text to Ollama using the given system prompt and returns the cleaned result.
func (c *OllamaClient) Process(systemPrompt, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	reqBody, err := json.Marshal(map[string]any{
		"model":  c.model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "Transcript:\n\n" + text},
		},
	})
	if err != nil {
		return "", fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/chat", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("ollama: build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: request failed (is Ollama running?): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ollama: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: API error %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("ollama: parse response: %w", err)
	}
	if result.Message.Content == "" {
		return "", fmt.Errorf("ollama: empty response")
	}
	return result.Message.Content, nil
}
