package llm

import (
	"context"
	"fmt"

	"github.com/coder11125/patchwork/pkg/domain"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMProvider interface {
	Name() string
	Chat(ctx context.Context, messages []Message) (string, error)
	ChatWithJSON(ctx context.Context, system string, user string, schema any) error
	SupportsStreaming() bool
}

func NewProvider(cfg domain.LLMConfig) (LLMProvider, error) {
	switch cfg.Provider {
	case domain.ProviderAnthropic:
		return NewAnthropic(cfg)
	case domain.ProviderMistral:
		return NewMistral(cfg)
	case domain.ProviderGroq:
		return NewGroq(cfg)
	case domain.ProviderOllama:
		return NewOllama(cfg)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
