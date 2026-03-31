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

const endpoint = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"

// CleanupPrompt is the system prompt used for all transcript cleanup calls.
// It preserves technical terms, CLI flags, code identifiers, and product names
// exactly as spoken — tuned for dictation to AI agents, Claude Code, and Slack.
const CleanupPrompt = `You are a transcript cleanup assistant. The user dictated this text using voice recognition.
Clean it up: fix punctuation, capitalization, and grammar. Remove filler words (um, uh, like, you know, actually, basically, sort of, right).
Keep all technical terms, CLI commands, flag names, code identifiers, API names, product names, agent names, and proper nouns exactly as spoken — do not translate or paraphrase them.
Return only the cleaned text with no explanation or preamble.`

// Client sends transcripts to the Claude API for cleanup.
type Client struct {
	apiKey  string
	model   string
	timeout time.Duration
	http    *http.Client
}

// New creates a Client. timeoutSeconds is applied per-call via context.
func New(apiKey, model string, timeoutSeconds int) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		timeout: time.Duration(timeoutSeconds) * time.Second,
		http:    &http.Client{},
	}
}

// Process sends text through Claude using the given system prompt.
// Returns the cleaned text, or an error — the caller is responsible for fallback.
func (c *Client) Process(systemPrompt, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	reqBody, err := json.Marshal(map[string]any{
		"model":      c.model,
		"max_tokens": 1024,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": "Transcript:\n\n" + text},
		},
	})
	if err != nil {
		return "", fmt.Errorf("llm: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: API error %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("llm: parse response: %w", err)
	}
	if len(result.Content) == 0 || result.Content[0].Text == "" {
		return "", fmt.Errorf("llm: empty response")
	}
	return result.Content[0].Text, nil
}
