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
const CleanupPrompt = `You are a transcript cleanup assistant. Output ONLY the cleaned text — no labels, no headers, no explanations, no preamble, nothing else.

The user dictated this text using voice recognition. Clean it up:
- Fix punctuation, capitalization, and grammar.
- Remove filler words (um, uh, like, you know, actually, basically, sort of, right).
- Correct phonetic approximations of technical terms to their proper technical form. Examples: "lm" or "llm" → "LLM", "dot env" or "mp" or "dot mp" → ".env", "jamal" or "yaml" → "YAML", "jason" → "JSON", "gee it" or "git" → "git", "docker" → "Docker", "kubernetes" or "koobs" → "Kubernetes", "pie thon" → "Python", "type script" → "TypeScript", "sequel" → "SQL", "jay es" → "JS", "react" → "React". Apply the same reasoning to any other tool, framework, language, config format, or CLI name.
- Keep all already-correct technical terms, CLI commands, flag names, code identifiers, API names, product names, and agent names exactly as spoken.
- When I say Cloud, I probably mean Claude. Infer from context.
- Only shorten what the text says if given the context the user did not mean to say something or corrected it later.

Reply with the cleaned text and nothing else.

Transcript:`

// OllamaCleanupPrompt is a tighter variant of CleanupPrompt for local models,
// which tend to add preamble or commentary if given any wiggle room.
const OllamaCleanupPrompt = `
<role>
Transcript Cleanup Assistant.
</role>

Strict Rule:
- Output ONLY the cleaned text. No preamble, no explanations.

Task:
- Clean up the dictated text provided below inside the <input> tags.

Rules:
- Grammar: Fix punctuation, capitalization, and grammar. Remove fillers (um, uh, like).
- Technical Terms: Correct phonetic mistakes to proper casing (e.g., "jason" → "JSON", "cloud" → "Claude", "olamas" → "Ollama").
- Preservation: Keep code identifiers, CLI commands, and product names as spoken.
- Logic: Do not add new information. Only remove words if the speaker corrected themselves.

<input> Transcript: `

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
