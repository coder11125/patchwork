package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

var breakingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)BREAKING`),
	regexp.MustCompile(`(?i)MAJOR\s*CHANGE`),
	regexp.MustCompile(`(?i)BC\b`),
	regexp.MustCompile(`(?i)deprecated`),
	regexp.MustCompile(`(?i)removed`),
	regexp.MustCompile(`(?i)dropped\s+support`),
	regexp.MustCompile(`(?i)no\s+longer\s+supports`),
	regexp.MustCompile(`(?i)incompatible`),
	regexp.MustCompile(`(?i)migration\s+(required|needed)`),
	regexp.MustCompile(`(?i)API\s+change`),
	regexp.MustCompile(`(?i)signature\s+change`),
	regexp.MustCompile(`(?i)breaking\s+change`),
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
}

type GitHubAnalyzer struct {
	httpClient *http.Client
	token      string
}

func NewGitHubAnalyzer(token string) *GitHubAnalyzer {
	return &GitHubAnalyzer{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
	}
}

func (a *GitHubAnalyzer) Ecosystem() domain.Ecosystem {
	return domain.EcosystemGo
}

func (a *GitHubAnalyzer) Analyze(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error) {
	repo := inferRepoFromPackage(pkg.Name)
	if repo == "" {
		return a.fallbackChangelog(pkg, currentVersion, latestVersion), nil
	}

	release, err := a.fetchRelease(ctx, repo, latestVersion)
	if err != nil {
		slog.Warn("failed to fetch GitHub release, falling back",
			"package", pkg.Name,
			"repo", repo,
			"error", err,
		)
		return a.fallbackChangelog(pkg, currentVersion, latestVersion), nil
	}

	entry := &domain.ChangelogEntry{
		Version:    latestVersion,
		RawContent: release.Body,
	}

	if release.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, release.PublishedAt); err == nil {
			entry.ReleaseDate = t.Format("2006-01-02")
		}
	}

	entry.BreakingChanges = a.extractBreakingChanges(release.Body, pkg.Name, latestVersion, release.HTMLURL)

	return entry, nil
}

func (a *GitHubAnalyzer) fetchRelease(ctx context.Context, repo, version string) (*githubRelease, error) {
	releases, err := a.fetchReleases(ctx, repo)
	if err != nil {
		return nil, err
	}

	cleanVersion := strings.TrimPrefix(version, "v")

	for _, rel := range releases {
		tagClean := strings.TrimPrefix(rel.TagName, "v")
		if tagClean == cleanVersion || rel.TagName == version {
			return &rel, nil
		}
	}

	if len(releases) > 0 {
		return &releases[0], nil
	}

	return nil, fmt.Errorf("no release found for version %s in %s", version, repo)
}

func (a *GitHubAnalyzer) fetchReleases(ctx context.Context, repo string) ([]githubRelease, error) {
	u := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=10", repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "patchwork-analyzer")

	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decoding releases: %w", err)
	}

	return releases, nil
}

func (a *GitHubAnalyzer) fetchChangelog(ctx context.Context, repo, version string) (string, error) {
	u := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/CHANGELOG.md", repo, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "patchwork-analyzer")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching changelog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("changelog not found: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading changelog: %w", err)
	}

	return string(body), nil
}

func (a *GitHubAnalyzer) extractBreakingChanges(body, packageName, version, sourceURL string) []domain.BreakingChange {
	if body == "" {
		return nil
	}

	var changes []domain.BreakingChange
	lines := strings.Split(body, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !isBreakingLine(trimmed) {
			continue
		}

		description := trimmed
		if strings.HasPrefix(description, "-") || strings.HasPrefix(description, "*") {
			description = strings.TrimSpace(description[1:])
		}

		if description == "" {
			continue
		}

		change := domain.BreakingChange{
			ID:          fmt.Sprintf("%s-%s-bc-%d", packageName, version, len(changes)),
			PackageName: packageName,
			Version:     version,
			Description: description,
			Severity:    "high",
			SourceURL:   sourceURL,
		}

		if i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && !isBreakingLine(next) && !strings.HasPrefix(next, "#") {
				change.Description += " " + next
			}
		}

		changes = append(changes, change)
	}

	return changes
}

func isBreakingLine(line string) bool {
	for _, pat := range breakingPatterns {
		if pat.MatchString(line) {
			return true
		}
	}
	return false
}

func (a *GitHubAnalyzer) fallbackChangelog(pkg domain.PackageInfo, currentVersion, latestVersion string) *domain.ChangelogEntry {
	return &domain.ChangelogEntry{
		Version:         latestVersion,
		RawContent:      fmt.Sprintf("No changelog available for %s %s", pkg.Name, latestVersion),
		BreakingChanges: nil,
	}
}

func inferRepoFromPackage(name string) string {
	if strings.HasPrefix(name, "github.com/") {
		parts := strings.SplitN(name, "/", 4)
		if len(parts) >= 3 {
			return parts[1] + "/" + parts[2]
		}
	}

	parsed, err := url.Parse(name)
	if err == nil && parsed.Host == "github.com" {
		parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	return ""
}
