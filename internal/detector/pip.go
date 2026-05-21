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

type PipDetector struct{}

func (d *PipDetector) Ecosystem() domain.Ecosystem {
	return domain.EcosystemPip
}

func (d *PipDetector) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	return base == "requirements.txt" || base == "requirements-dev.txt" || strings.HasPrefix(base, "requirements")
}

func (d *PipDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	reqPath := filepath.Join(dir, "requirements.txt")
	if _, err := os.Stat(reqPath); os.IsNotExist(err) {
		return nil, nil
	}

	f, err := os.Open(reqPath)
	if err != nil {
		return nil, fmt.Errorf("open requirements.txt: %w", err)
	}
	defer f.Close()

	var packages []domain.PackageInfo
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		name, current := parseRequirementLine(line)
		if name == "" {
			continue
		}

		latest, err := fetchLatestPipVersion(ctx, name)
		if err != nil {
			continue
		}

		packages = append(packages, domain.PackageInfo{
			Name:       name,
			Current:    current,
			Latest:     latest,
			IsOutdated: current != latest && current != "",
			IsDirect:   true,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan requirements.txt: %w", err)
	}

	return &domain.DetectResult{
		Ecosystem:    domain.EcosystemPip,
		Dir:          dir,
		ManifestPath: reqPath,
		Packages:     packages,
		DetectedAt:   time.Now(),
	}, nil
}

func parseRequirementLine(line string) (string, string) {
	line = strings.TrimSpace(line)

	for _, sep := range []string{">=", "<=", "==", "!=", "~=", ">", "<"} {
		if idx := strings.Index(line, sep); idx != -1 {
			name := strings.TrimSpace(line[:idx])
			version := strings.TrimSpace(line[idx+len(sep):])
			version = strings.SplitN(version, ",", 2)[0]
			version = strings.TrimSpace(version)
			return normalizePipName(name), version
		}
	}

	if strings.Contains(line, "[") {
		parts := strings.SplitN(line, "[", 2)
		name := strings.TrimSpace(parts[0])
		return normalizePipName(name), ""
	}

	return normalizePipName(line), ""
}

func normalizePipName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func fetchLatestPipVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", pkgName)

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
		return "", fmt.Errorf("pypi returned %d for %s", resp.StatusCode, pkgName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response for %s: %w", pkgName, err)
	}

	var info struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("unmarshal response for %s: %w", pkgName, err)
	}

	return info.Info.Version, nil
}
