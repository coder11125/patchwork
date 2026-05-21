package pr

import (
	"context"
	"fmt"

	"github.com/coder11125/patchwork/pkg/domain"
)

type PRRequest struct {
	Owner string
	Repo  string
	Head  string
	Base  string
	Title string
	Body  string
}

type PRResponse struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
	State  string `json:"state"`
}

type PRCreator interface {
	Platform() string
	CreatePR(ctx context.Context, req PRRequest) (*PRResponse, error)
}

func NewCreator(cfg domain.GitConfig) (PRCreator, error) {
	switch cfg.Platform {
	case "github":
		return NewGitHub(cfg)
	case "gitlab":
		return NewGitLab(cfg)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", cfg.Platform)
	}
}
