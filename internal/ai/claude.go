package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/slucheninov/gmr/internal/ui"
)

// Claude implements Provider against Anthropic's Messages API.
type Claude struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewClaude builds a Claude provider; if model is empty, a sane default is used.
func NewClaude(apiKey, model string) *Claude {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &Claude{APIKey: apiKey, Model: model, BaseURL: "https://api.anthropic.com"}
}

// Name implements Provider.
func (c *Claude) Name() string { return "Claude" }

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type claudeReq struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}
type claudeResp struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate implements Provider.
func (c *Claude) Generate(ctx context.Context, diff string) (string, error) {
	if c.APIKey == "" {
		return "", ErrNoAPIKey
	}
	ui.Log("Generating commit message via Claude API...")

	body := claudeReq{
		Model:     c.Model,
		MaxTokens: 1024,
		Messages:  []claudeMessage{{Role: "user", Content: CommitPrompt + diff}},
	}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/messages", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := HTTPClient.Do(req)
	if err != nil {
		ui.Warn("Claude API error: %s", err)
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var parsed claudeResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		ui.Warn("Claude API error: invalid response")
		return "", err
	}
	if parsed.Error != nil {
		ui.Warn("Claude API error: %s", parsed.Error.Message)
		return "", fmt.Errorf("claude: %s", parsed.Error.Message)
	}
	if len(parsed.Content) == 0 {
		return "", fmt.Errorf("claude: empty response")
	}
	msg := strings.TrimSpace(parsed.Content[0].Text)
	if msg == "" {
		return "", fmt.Errorf("claude: empty response")
	}
	if parsed.StopReason == "max_tokens" {
		ui.Warn("Claude response truncated, using first line")
		msg = firstLine(msg)
		if msg == "" {
			return "", fmt.Errorf("claude: empty response")
		}
	}
	return msg, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
