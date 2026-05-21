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
	"net/url"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type gitlabCreator struct {
	token string
	owner string
	repo  string
}

func NewGitLab(cfg domain.GitConfig) (*gitlabCreator, error) {
	if cfg.Token == "" {
		return nil, errors.New("gitlab token is required")
	}
	return &gitlabCreator{
		token: cfg.Token,
		owner: cfg.Owner,
		repo:  cfg.Repo,
	}, nil
}

func (g *gitlabCreator) Platform() string {
	return "gitlab"
}

func (g *gitlabCreator) CreatePR(ctx context.Context, req PRRequest) (*PRResponse, error) {
	log := slog.With("platform", "gitlab", "owner", req.Owner, "repo", req.Repo, "source_branch", req.Head, "target_branch", req.Base)

	projectPath := url.PathEscape(req.Owner + "/" + req.Repo)
	apiURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/merge_requests", projectPath)

	payload := map[string]string{
		"title":         req.Title,
		"description":   req.Body,
		"source_branch": req.Head,
		"target_branch": req.Base,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal gitlab merge request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create gitlab merge request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("PRIVATE-TOKEN", g.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute gitlab merge request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gitlab merge request response: %w", err)
	}

	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusUnprocessableEntity {
		var apiErr struct {
			Message string   `json:"message"`
			Errors  []string `json:"errors"`
		}
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			if containsMessage(respBody, "already exists") || containsMessage(respBody, "open merge request") {
				log.Warn("merge request already exists", "status", resp.StatusCode, "response", string(respBody))
				return nil, &PRExistsError{Message: "merge request already exists"}
			}
		}
		return nil, fmt.Errorf("gitlab merge request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitlab merge request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		WebURL string `json:"web_url"`
		IID    int    `json:"iid"`
		State  string `json:"state"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal gitlab merge request response: %w", err)
	}

	log.Info("merge request created", "url", result.WebURL, "iid", result.IID, "state", result.State)

	return &PRResponse{
		URL:    result.WebURL,
		Number: result.IID,
		State:  result.State,
	}, nil
}
