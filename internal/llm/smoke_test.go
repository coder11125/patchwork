package llm

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

func smokeConfig(provider domain.LLMProviderType, keyEnv string) domain.LLMConfig {
	key := os.Getenv(keyEnv)
	if key == "" {
		return domain.LLMConfig{}
	}
	return domain.LLMConfig{
		Provider:    provider,
		APIKey:      key,
		Model:       "",
		MaxTokens:   50,
		Temperature: 0,
		TimeoutSec:  15,
	}
}

func TestSmokeAnthropic(t *testing.T) {
	cfg := smokeConfig(domain.ProviderAnthropic, "ANTHROPIC_API_KEY")
	if cfg.APIKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := p.Chat(ctx, []Message{{Role: "user", Content: "Say hello in one word."}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp == "" {
		t.Fatal("empty response")
	}
	t.Logf("response: %s", resp)
}

func TestSmokeMistral(t *testing.T) {
	cfg := smokeConfig(domain.ProviderMistral, "MISTRAL_API_KEY")
	if cfg.APIKey == "" {
		t.Skip("MISTRAL_API_KEY not set")
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := p.Chat(ctx, []Message{{Role: "user", Content: "Say hello in one word."}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp == "" {
		t.Fatal("empty response")
	}
	t.Logf("response: %s", resp)
}

func TestSmokeGroq(t *testing.T) {
	cfg := smokeConfig(domain.ProviderGroq, "GROQ_API_KEY")
	if cfg.APIKey == "" {
		t.Skip("GROQ_API_KEY not set")
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := p.Chat(ctx, []Message{{Role: "user", Content: "Say hello in one word."}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp == "" {
		t.Fatal("empty response")
	}
	t.Logf("response: %s", resp)
}

func TestSmokeOllama(t *testing.T) {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	cfg := domain.LLMConfig{
		Provider:    domain.ProviderOllama,
		BaseURL:     baseURL,
		Model:       "",
		MaxTokens:   50,
		Temperature: 0,
		TimeoutSec:  30,
	}
	p, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := p.Chat(ctx, []Message{{Role: "user", Content: "Say hello in one word."}})
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
	}
	if resp == "" {
		t.Fatal("empty response")
	}
	t.Logf("response: %s", resp)
}
