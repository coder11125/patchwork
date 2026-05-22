package detector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type BundlerDetector struct{}

func (d *BundlerDetector) Ecosystem() domain.Ecosystem {
	return domain.EcosystemRuby
}

func (d *BundlerDetector) CanHandle(filePath string) bool {
	return filepath.Base(filePath) == "Gemfile"
}

func (d *BundlerDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	gemfilePath := filepath.Join(dir, "Gemfile")
	if _, err := os.Stat(gemfilePath); os.IsNotExist(err) {
		return nil, nil
	}

	f, err := os.Open(gemfilePath)
	if err != nil {
		return nil, fmt.Errorf("open Gemfile: %w", err)
	}
	defer f.Close()

	var packages []domain.PackageInfo
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "source ") || strings.HasPrefix(line, "ruby ") || strings.HasPrefix(line, "git_source") {
			continue
		}

		name, version := parseGemfileLine(line)
		if name == "" {
			continue
		}

		latest, err := fetchLatestRubyGemsVersion(ctx, name)
		if err != nil {
			continue
		}

		packages = append(packages, domain.PackageInfo{
			Name:       name,
			Current:    version,
			Latest:     latest,
			IsOutdated: isVersionOutdated(version, latest),
			IsDirect:   true,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan Gemfile: %w", err)
	}

	return &domain.DetectResult{
		Ecosystem:    domain.EcosystemRuby,
		Dir:          dir,
		ManifestPath: gemfilePath,
		Packages:     packages,
		DetectedAt:   time.Now(),
	}, nil
}

func parseGemfileLine(line string) (string, string) {
	trimmed := strings.TrimSpace(line)

	if !strings.HasPrefix(trimmed, "gem ") {
		return "", ""
	}

	rest := strings.TrimSpace(trimmed[4:])

	if len(rest) < 2 {
		return "", ""
	}

	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return "", ""
	}

	endIdx := strings.IndexByte(rest[1:], byte(quote))
	if endIdx == -1 {
		return "", ""
	}
	name := rest[1 : endIdx+1]

	afterName := strings.TrimSpace(rest[endIdx+2:])
	if afterName == "" || afterName[0] != ',' {
		return name, ""
	}

	versionPart := strings.TrimSpace(afterName[1:])
	if versionPart == "" {
		return name, ""
	}

	if versionPart[0] != '"' && versionPart[0] != '\'' {
		return name, ""
	}
	vQuote := versionPart[0]
	vEnd := strings.IndexByte(versionPart[1:], byte(vQuote))
	if vEnd == -1 {
		return name, ""
	}
	version := versionPart[1 : vEnd+1]

	version = strings.TrimSpace(version)
	for _, prefix := range []string{"~> ", ">= ", "<= ", "> ", "< ", "= "} {
		if strings.HasPrefix(version, prefix) {
			version = strings.TrimSpace(version[len(prefix):])
			break
		}
	}

	return name, version
}

func fetchLatestRubyGemsVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("https://rubygems.org/api/v1/gems/%s.json", pkgName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request for %s: %w", pkgName, err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest for %s: %w", pkgName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rubygems returned %d for %s", resp.StatusCode, pkgName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response for %s: %w", pkgName, err)
	}

	var info struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("unmarshal response for %s: %w", pkgName, err)
	}

	return info.Version, nil
}
