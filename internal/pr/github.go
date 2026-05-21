package pr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type githubCreator struct {
	token string
	owner string
	repo  string
}

func NewGitHub(cfg domain.GitConfig) (*githubCreator, error) {
	if cfg.Token == "" {
		return nil, errors.New("github token is required")
	}
	return &githubCreator{
		token: cfg.Token,
		owner: cfg.Owner,
		repo:  cfg.Repo,
	}, nil
}

func (g *githubCreator) Platform() string {
	return "github"
}

func (g *githubCreator) CreatePR(ctx context.Context, req PRRequest) (*PRResponse, error) {
	log := slog.With("platform", "github", "owner", req.Owner, "repo", req.Repo, "head", req.Head, "base", req.Base)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", req.Owner, req.Repo)

	payload := map[string]string{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.Head,
		"base":  req.Base,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal github pr request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create github pr request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/vnd.github.v3+json")
	httpReq.Header.Set("Authorization", "Bearer "+g.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute github pr request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read github pr response: %w", err)
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		var apiErr struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			if apiErr.Message == "Validation Failed" || containsMessage(respBody, "already exists") {
				log.Warn("pull request already exists", "status", resp.StatusCode, "response", string(respBody))
				return nil, &PRExistsError{Message: "pull request already exists"}
			}
		}
		return nil, fmt.Errorf("github pr validation failed (422): %s", string(respBody))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github pr request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal github pr response: %w", err)
	}

	log.Info("pull request created", "url", result.HTMLURL, "number", result.Number, "state", result.State)

	return &PRResponse{
		URL:    result.HTMLURL,
		Number: result.Number,
		State:  result.State,
	}, nil
}

type PRExistsError struct {
	Message string
}

func (e *PRExistsError) Error() string {
	return e.Message
}

func containsMessage(body []byte, substr string) bool {
	return bytes.Contains(bytes.ToLower(body), []byte(substr))
}
