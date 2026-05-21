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
	mistralAPIURL     = "https://api.mistral.ai/v1/chat/completions"
	mistralMaxRetries = 3
)

type MistralProvider struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralRequest struct {
	Model       string           `json:"model"`
	Messages    []mistralMessage `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

type mistralChoice struct {
	Index        int            `json:"index"`
	Message      mistralMessage `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type mistralUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type mistralResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []mistralChoice `json:"choices"`
	Usage   mistralUsage    `json:"usage"`
}

type mistralErrorResponse struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func NewMistral(cfg domain.LLMConfig) (*MistralProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("mistral: API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "mistral-large-latest"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &MistralProvider{
		apiKey:      cfg.APIKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *MistralProvider) Name() string {
	return "mistral"
}

func (p *MistralProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("mistral: messages cannot be empty")
	}

	reqMessages := make([]mistralMessage, 0, len(messages))
	for _, m := range messages {
		reqMessages = append(reqMessages, mistralMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	reqBody := mistralRequest{
		Model:     p.model,
		Messages:  reqMessages,
		MaxTokens: p.maxTokens,
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

func (p *MistralProvider) ChatWithJSON(ctx context.Context, system string, user string, schema any) error {
	if _, err := json.Marshal(schema); err != nil {
		return fmt.Errorf("mistral: failed to marshal schema: %w", err)
	}

	systemPrompt := fmt.Sprintf("%s\n\nRespond with valid JSON only. Do not include any other text or explanation.", system)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: user},
	}

	response, err := p.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("mistral: chat request failed: %w", err)
	}

	return json.Unmarshal([]byte(response), schema)
}

func (p *MistralProvider) SupportsStreaming() bool {
	return true
}

func (p *MistralProvider) doRequest(ctx context.Context, reqBody mistralRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("mistral: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("mistral: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("mistral: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("mistral: failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp mistralErrorResponse
		if parseErr := json.Unmarshal(body, &errResp); parseErr == nil && errResp.Error != nil {
			return "", fmt.Errorf("mistral: API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("mistral: HTTP error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp mistralResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("mistral: failed to unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("mistral: empty response choices")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func (p *MistralProvider) withRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt <= mistralMaxRetries; attempt++ {
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
	return fmt.Errorf("mistral: all %d retries exhausted: %w", mistralMaxRetries, lastErr)
}
