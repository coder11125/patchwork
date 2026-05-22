package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder11125/patchwork/internal/llm"
	"github.com/coder11125/patchwork/pkg/domain"
)

const llmAnalysisPrompt = `You are an expert software engineer analyzing changelogs for breaking changes.

Analyze the following changelog content for package %s upgrading from version %s to %s.

Extract all breaking changes, API modifications, deprecated features, and removed functionality.

Return ONLY a valid JSON array with this exact schema:
[
  {
    "id": "unique-id",
    "package_name": "%s",
    "version": "%s",
    "description": "clear description of the breaking change",
    "severity": "critical|high|medium|low",
    "affected_apis": ["affected.API", "another.Function"],
    "migration_hint": "how to migrate away from this breaking change",
    "source_url": ""
  }
]

If there are no breaking changes, return an empty array: []

Changelog content:
---
%s
---`

type LLMAnalyzer struct {
	provider       llm.LLMProvider
	semverAnalyzer *SemverAnalyzer
	ecosystem      domain.Ecosystem
	httpClient     *http.Client
}

func NewLLMAnalyzer(provider llm.LLMProvider) *LLMAnalyzer {
	return &LLMAnalyzer{
		provider:       provider,
		semverAnalyzer: NewSemverAnalyzer(),
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func NewLLMAnalyzerForEcosystem(provider llm.LLMProvider, eco domain.Ecosystem) *LLMAnalyzer {
	return &LLMAnalyzer{
		provider:       provider,
		semverAnalyzer: NewSemverAnalyzer(),
		ecosystem:      eco,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *LLMAnalyzer) Ecosystem() domain.Ecosystem {
	return a.ecosystem
}

func (a *LLMAnalyzer) Analyze(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error) {
	semChanges, _, _ := a.semverAnalyzer.AnalyzeVersion(currentVersion, latestVersion, pkg.Name)

	entry := &domain.ChangelogEntry{
		Version:         latestVersion,
		BreakingChanges: semChanges,
	}

	if a.provider == nil {
		return entry, nil
	}

	releaseBody, releaseDate := a.fetchGitHubRelease(ctx, pkg.Name, latestVersion)
	if releaseBody != "" {
		entry.RawContent = releaseBody
		entry.ReleaseDate = releaseDate

		llmChanges, _, _ := a.AnalyzeWithFallback(ctx, pkg.Name, currentVersion, latestVersion, releaseBody)
		if len(llmChanges) > 0 {
			entry.BreakingChanges = llmChanges
		}
	}

	return entry, nil
}

func (a *LLMAnalyzer) fetchGitHubRelease(ctx context.Context, packageName, version string) (body string, date string) {
	repo := inferRepoFromPackage(packageName)
	if repo == "" {
		return "", ""
	}

	u := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=10", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "patchwork-analyzer")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	var releases []struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.Unmarshal(bodyBytes, &releases); err != nil {
		return "", ""
	}

	cleanVersion := strings.TrimPrefix(version, "v")
	for _, rel := range releases {
		tagClean := strings.TrimPrefix(rel.TagName, "v")
		if tagClean == cleanVersion || rel.TagName == version {
			if t, err := time.Parse(time.RFC3339, rel.PublishedAt); err == nil {
				return rel.Body, t.Format("2006-01-02")
			}
			return rel.Body, ""
		}
	}

	if len(releases) > 0 {
		if t, err := time.Parse(time.RFC3339, releases[0].PublishedAt); err == nil {
			return releases[0].Body, t.Format("2006-01-02")
		}
		return releases[0].Body, ""
	}

	return "", ""
}

func (a *LLMAnalyzer) AnalyzeChangelog(ctx context.Context, packageName, currentVersion, latestVersion, changelogContent string) ([]domain.BreakingChange, error) {
	if changelogContent == "" {
		return nil, fmt.Errorf("empty changelog content")
	}

	prompt := fmt.Sprintf(llmAnalysisPrompt,
		packageName, currentVersion, latestVersion,
		packageName, latestVersion,
		changelogContent,
	)

	messages := []llm.Message{
		{Role: "system", Content: "You are a changelog analyzer. Return only valid JSON arrays."},
		{Role: "user", Content: prompt},
	}

	response, err := a.provider.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM chat failed: %w", err)
	}

	changes, err := parseLLMResponse(response, packageName, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w", err)
	}

	return changes, nil
}

func (a *LLMAnalyzer) AnalyzeWithFallback(ctx context.Context, packageName, currentVersion, latestVersion, changelogContent string) ([]domain.BreakingChange, domain.RiskLevel, bool) {
	changes, err := a.AnalyzeChangelog(ctx, packageName, currentVersion, latestVersion, changelogContent)
	if err != nil {
		slog.Warn("LLM analysis failed, falling back to semver",
			"package", packageName,
			"error", err,
		)
		semChanges, risk, isBreaking := a.semverAnalyzer.AnalyzeVersion(currentVersion, latestVersion, packageName)
		return semChanges, risk, isBreaking
	}

	if len(changes) > 0 {
		return changes, domain.RiskHigh, true
	}

	semChanges, risk, isBreaking := a.semverAnalyzer.AnalyzeVersion(currentVersion, latestVersion, packageName)
	return semChanges, risk, isBreaking
}

func parseLLMResponse(response, packageName, version string) ([]domain.BreakingChange, error) {
	trimmed := response
	start := -1
	end := -1

	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == '[' {
			start = i
			break
		}
	}
	for i := len(trimmed) - 1; i >= 0; i-- {
		if trimmed[i] == ']' {
			end = i + 1
			break
		}
	}

	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := trimmed[start:end]

	var rawChanges []struct {
		ID            string   `json:"id"`
		PackageName   string   `json:"package_name"`
		Version       string   `json:"version"`
		Description   string   `json:"description"`
		Severity      string   `json:"severity"`
		AffectedAPIs  []string `json:"affected_apis"`
		MigrationHint string   `json:"migration_hint"`
		SourceURL     string   `json:"source_url"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawChanges); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	var changes []domain.BreakingChange
	for i, rc := range rawChanges {
		change := domain.BreakingChange{
			ID:            rc.ID,
			PackageName:   rc.PackageName,
			Version:       rc.Version,
			Description:   rc.Description,
			Severity:      rc.Severity,
			AffectedAPIs:  rc.AffectedAPIs,
			MigrationHint: rc.MigrationHint,
			SourceURL:     rc.SourceURL,
		}

		if change.ID == "" {
			change.ID = fmt.Sprintf("%s-%s-llm-%d", packageName, version, i)
		}
		if change.PackageName == "" {
			change.PackageName = packageName
		}
		if change.Version == "" {
			change.Version = version
		}

		changes = append(changes, change)
	}

	return changes, nil
}
