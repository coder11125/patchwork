package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
}

func NewLLMAnalyzer(provider llm.LLMProvider) *LLMAnalyzer {
	return &LLMAnalyzer{
		provider:       provider,
		semverAnalyzer: NewSemverAnalyzer(),
	}
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
