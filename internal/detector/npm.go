package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type NPMDetector struct{}

func (d *NPMDetector) Ecosystem() domain.Ecosystem {
	return domain.EcosystemNPM
}

func (d *NPMDetector) CanHandle(filePath string) bool {
	return filepath.Base(filePath) == "package.json"
}

func (d *NPMDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	pkgJSONPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(pkgJSONPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	allDeps := make(map[string]bool)
	for name := range pkg.Dependencies {
		allDeps[name] = true
	}
	for name := range pkg.DevDependencies {
		allDeps[name] = true
	}

	var packages []domain.PackageInfo

	for name := range allDeps {
		current := ""
		if v, ok := pkg.Dependencies[name]; ok {
			current = cleanVersion(v)
		} else if v, ok := pkg.DevDependencies[name]; ok {
			current = cleanVersion(v)
		}

		latest, err := fetchLatestNPMVersion(ctx, name)
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

	return &domain.DetectResult{
		Ecosystem:    domain.EcosystemNPM,
		Dir:          dir,
		ManifestPath: pkgJSONPath,
		Packages:     packages,
		DetectedAt:   time.Now(),
	}, nil
}

func fetchLatestNPMVersion(ctx context.Context, pkgName string) (string, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", pkgName)

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
		return "", fmt.Errorf("npm registry returned %d for %s", resp.StatusCode, pkgName)
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

func cleanVersion(v string) string {
	for len(v) > 0 && (v[0] == '^' || v[0] == '~' || v[0] == '>' || v[0] == '=' || v[0] == '<') {
		v = v[1:]
	}
	return v
}
