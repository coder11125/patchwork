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
	groqAPIURL     = "https://api.groq.com/openai/v1/chat/completions"
	groqMaxRetries = 3
)

type GroqProvider struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model       string        `json:"model"`
	Messages    []groqMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type groqChoice struct {
	Index        int         `json:"index"`
	Message      groqMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type groqUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type groqResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []groqChoice `json:"choices"`
	Usage   groqUsage    `json:"usage"`
}

type groqErrorResponse struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func NewGroq(cfg domain.LLMConfig) (*GroqProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("groq: API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &GroqProvider{
		apiKey:      cfg.APIKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *GroqProvider) Name() string {
	return "groq"
}

func (p *GroqProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("groq: messages cannot be empty")
	}

	reqMessages := make([]groqMessage, 0, len(messages))
	for _, m := range messages {
		reqMessages = append(reqMessages, groqMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	reqBody := groqRequest{
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

func (p *GroqProvider) ChatWithJSON(ctx context.Context, system string, user string, schema any) error {
	if _, err := json.Marshal(schema); err != nil {
		return fmt.Errorf("groq: failed to marshal schema: %w", err)
	}

	systemPrompt := fmt.Sprintf("%s\n\nRespond with valid JSON only. Do not include any other text or explanation.", system)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: user},
	}

	response, err := p.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("groq: chat request failed: %w", err)
	}

	return json.Unmarshal([]byte(response), schema)
}

func (p *GroqProvider) SupportsStreaming() bool {
	return true
}

func (p *GroqProvider) doRequest(ctx context.Context, reqBody groqRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("groq: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("groq: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("groq: failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp groqErrorResponse
		if parseErr := json.Unmarshal(body, &errResp); parseErr == nil && errResp.Error != nil {
			return "", fmt.Errorf("groq: API error (status %d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("groq: HTTP error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp groqResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("groq: failed to unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("groq: empty response choices")
	}

	return apiResp.Choices[0].Message.Content, nil
}

func (p *GroqProvider) withRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt <= groqMaxRetries; attempt++ {
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
	return fmt.Errorf("groq: all %d retries exhausted: %w", groqMaxRetries, lastErr)
}
