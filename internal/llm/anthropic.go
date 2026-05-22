package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
	anthropicMaxRetries = 3
)

type AnthropicProvider struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Error      *anthropicError         `json:"error,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func NewAnthropic(cfg domain.LLMConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic: API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &AnthropicProvider{
		apiKey:      cfg.APIKey,
		baseURL:     cfg.BaseURL,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *AnthropicProvider) apiURL() string {
	if p.baseURL != "" {
		return p.baseURL
	}
	return anthropicAPIURL
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("anthropic: messages cannot be empty")
	}

	var systemMsg string
	var chatMessages []anthropicMessage

	for _, m := range messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			chatMessages = append(chatMessages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	reqBody := anthropicRequest{
		Model:     p.model,
		MaxTokens: p.maxTokens,
		System:    systemMsg,
		Messages:  chatMessages,
	}

	if p.temperature > 0 {
		reqBody.Temperature = p.temperature
	}

	var result string
	err := p.withRetry(ctx, func(ctx context.Context) error {
		resp, err := p.doRequest(ctx, reqBody)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})

	return result, err
}

func (p *AnthropicProvider) ChatWithJSON(ctx context.Context, system string, user string, schema any) error {
	if _, err := json.Marshal(schema); err != nil {
		return fmt.Errorf("anthropic: failed to marshal schema: %w", err)
	}

	systemPrompt := fmt.Sprintf("%s\n\nRespond with valid JSON only. Do not include any other text or explanation.", system)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: user},
	}

	response, err := p.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("anthropic: chat request failed: %w", err)
	}

	return json.Unmarshal([]byte(response), schema)
}

func (p *AnthropicProvider) SupportsStreaming() bool {
	return true
}

func (p *AnthropicProvider) doRequest(ctx context.Context, reqBody anthropicRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL(), bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr anthropicResponse
		if parseErr := json.Unmarshal(body, &apiErr); parseErr == nil && apiErr.Error != nil {
			return "", fmt.Errorf("anthropic: API error (status %d): %s - %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return "", fmt.Errorf("anthropic: HTTP error (status %d): %s", resp.StatusCode, sanitizeBody(body, p.apiKey))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("anthropic: failed to unmarshal response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response content")
	}

	return apiResp.Content[0].Text, nil
}

func (p *AnthropicProvider) withRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < anthropicMaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
	}
	return fmt.Errorf("anthropic: all %d retries exhausted: %w", anthropicMaxRetries, lastErr)
}
