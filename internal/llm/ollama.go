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
	ollamaAPIURL     = "http://localhost:11434/api/chat"
	ollamaMaxRetries = 3
)

type OllamaProvider struct {
	baseURL     string
	model       string
	temperature float64
	httpClient  *http.Client
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature float64         `json:"temperature,omitempty"`
}

type ollamaResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            ollamaMessage `json:"message"`
	Done               bool          `json:"done"`
	TotalDuration      int64         `json:"total_duration,omitempty"`
	LoadDuration       int64         `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64         `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       int64         `json:"eval_duration,omitempty"`
}

type ollamaErrorResponse struct {
	Error string `json:"error,omitempty"`
}

func NewOllama(cfg domain.LLMConfig) (*OllamaProvider, error) {
	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &OllamaProvider{
		baseURL:     cfg.BaseURL,
		model:       model,
		temperature: cfg.Temperature,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (p *OllamaProvider) apiURL() string {
	if p.baseURL != "" {
		return p.baseURL
	}
	return ollamaAPIURL
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("ollama: messages cannot be empty")
	}

	reqMessages := make([]ollamaMessage, 0, len(messages))
	for _, m := range messages {
		reqMessages = append(reqMessages, ollamaMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	reqBody := ollamaRequest{
		Model:    p.model,
		Messages: reqMessages,
		Stream:   false,
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

func (p *OllamaProvider) ChatWithJSON(ctx context.Context, system string, user string, schema any) error {
	if _, err := json.Marshal(schema); err != nil {
		return fmt.Errorf("ollama: failed to marshal schema: %w", err)
	}

	systemPrompt := fmt.Sprintf("%s\n\nRespond with valid JSON only. Do not include any other text or explanation.", system)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: user},
	}

	response, err := p.Chat(ctx, messages)
	if err != nil {
		return fmt.Errorf("ollama: chat request failed: %w", err)
	}

	return json.Unmarshal([]byte(response), schema)
}

func (p *OllamaProvider) SupportsStreaming() bool {
	return false
}

func (p *OllamaProvider) doRequest(ctx context.Context, reqBody ollamaRequest) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ollama: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL(), bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("ollama: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ollama: failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ollamaErrorResponse
		if parseErr := json.Unmarshal(body, &errResp); parseErr == nil && errResp.Error != "" {
			return "", fmt.Errorf("ollama: API error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return "", fmt.Errorf("ollama: HTTP error (status %d): %s", resp.StatusCode, sanitizeBody(body, ""))
	}

	var apiResp ollamaResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("ollama: failed to unmarshal response: %w", err)
	}

	return apiResp.Message.Content, nil
}

func (p *OllamaProvider) withRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < ollamaMaxRetries; attempt++ {
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
	return fmt.Errorf("ollama: all %d retries exhausted: %w", ollamaMaxRetries, lastErr)
}
