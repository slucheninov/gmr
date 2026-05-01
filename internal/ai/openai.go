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

// OpenAI implements Provider against the OpenAI Chat Completions API.
type OpenAI struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewOpenAI builds an OpenAI provider; if model is empty, a sane default is used.
func NewOpenAI(apiKey, model string) *OpenAI {
	return NewOpenAIWithBaseURL(apiKey, model, "")
}

// NewOpenAIWithBaseURL builds an OpenAI provider with an optional API base URL override.
func NewOpenAIWithBaseURL(apiKey, model, baseURL string) *OpenAI {
	if model == "" {
		model = "gpt-4o-mini"
	}
	baseURL = normalizeBaseURL(baseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAI{APIKey: apiKey, Model: model, BaseURL: baseURL}
}

// Name implements Provider.
func (o *OpenAI) Name() string { return "OpenAI" }

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type openaiReq struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openaiMessage `json:"messages"`
}
type openaiResp struct {
	Choices []struct {
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate implements Provider.
func (o *OpenAI) Generate(ctx context.Context, diff string) (string, error) {
	if o.APIKey == "" {
		return "", ErrNoAPIKey
	}
	ui.Log("Generating commit message via OpenAI API...")

	body := openaiReq{
		Model:     o.Model,
		MaxTokens: 1024,
		Messages:  []openaiMessage{{Role: "user", Content: CommitPrompt + diff}},
	}
	buf, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/v1/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		ui.Warn("OpenAI API error: %s", err)
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var parsed openaiResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		ui.Warn("OpenAI API error: invalid response")
		return "", err
	}
	if parsed.Error != nil {
		ui.Warn("OpenAI API error: %s", parsed.Error.Message)
		return "", fmt.Errorf("openai: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}
	msg := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if msg == "" {
		return "", fmt.Errorf("openai: empty response")
	}
	if parsed.Choices[0].FinishReason == "length" {
		ui.Warn("OpenAI response truncated, using first line")
		msg = firstLine(msg)
		if msg == "" {
			return "", fmt.Errorf("openai: empty response")
		}
	}
	return msg, nil
}
