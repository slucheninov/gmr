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

// Gemini implements Provider against Google's Gemini API.
type Gemini struct {
	APIKey  string
	Model   string
	BaseURL string
}

// NewGemini builds a Gemini provider; if model is empty, a sane default is used.
func NewGemini(apiKey, model string) *Gemini {
	if model == "" {
		model = "gemini-flash-latest"
	}
	return &Gemini{APIKey: apiKey, Model: model, BaseURL: "https://generativelanguage.googleapis.com/v1beta"}
}

// Name implements Provider.
func (g *Gemini) Name() string { return "Gemini" }

type geminiPart struct {
	Text string `json:"text"`
}
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}
type geminiReq struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}
type geminiResp struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate implements Provider.
func (g *Gemini) Generate(ctx context.Context, diff string) (string, error) {
	if g.APIKey == "" {
		return "", ErrNoAPIKey
	}
	ui.Log("Generating commit message via Gemini API...")

	body := geminiReq{Contents: []geminiContent{{Parts: []geminiPart{{Text: CommitPrompt + diff}}}}}
	body.GenerationConfig.MaxOutputTokens = 1024
	buf, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/models/%s:generateContent", g.BaseURL, g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-goog-api-key", g.APIKey)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		ui.Warn("Gemini API error: %s", err)
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var parsed geminiResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		ui.Warn("Gemini API error: invalid response")
		return "", err
	}
	if parsed.Error != nil {
		ui.Warn("Gemini API error: %s", parsed.Error.Message)
		return "", fmt.Errorf("gemini: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	msg := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	if msg == "" {
		return "", fmt.Errorf("gemini: empty response")
	}
	if parsed.Candidates[0].FinishReason == "MAX_TOKENS" {
		ui.Warn("Gemini response truncated, using first line")
		msg = firstLine(msg)
		if msg == "" {
			return "", fmt.Errorf("gemini: empty response")
		}
	}
	return msg, nil
}
