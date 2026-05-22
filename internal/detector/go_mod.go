package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/modfile"

	"github.com/coder11125/patchwork/pkg/domain"
)

type GoModDetector struct{}

func (d *GoModDetector) Ecosystem() domain.Ecosystem {
	return domain.EcosystemGo
}

func (d *GoModDetector) CanHandle(filePath string) bool {
	return filepath.Base(filePath) == "go.mod"
}

func (d *GoModDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	goModPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}

	modFile, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	var packages []domain.PackageInfo

	for _, req := range modFile.Require {
		if req.Indirect {
			continue
		}

		latest, err := fetchLatestGoVersion(ctx, req.Mod.Path)
		if err != nil {
			continue
		}

		packages = append(packages, domain.PackageInfo{
			Name:       req.Mod.Path,
			Current:    req.Mod.Version,
			Latest:     latest,
			IsOutdated: isVersionOutdated(req.Mod.Version, latest),
			IsDirect:   true,
		})
	}

	return &domain.DetectResult{
		Ecosystem:    domain.EcosystemGo,
		Dir:          dir,
		ManifestPath: goModPath,
		Packages:     packages,
		DetectedAt:   time.Now(),
	}, nil
}

func fetchLatestGoVersion(ctx context.Context, modulePath string) (string, error) {
	escaped := moduleEscape(modulePath)
	url := fmt.Sprintf("https://proxy.golang.org/%s/@latest", escaped)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request for %s: %w", modulePath, err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest for %s: %w", modulePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy returned %d for %s", resp.StatusCode, modulePath)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response for %s: %w", modulePath, err)
	}

	var info struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("unmarshal response for %s: %w", modulePath, err)
	}

	return info.Version, nil
}

func moduleEscape(path string) string {
	var b strings.Builder
	for _, r := range path {
		if isUpper(r) {
			b.WriteRune('!')
			b.WriteRune(toLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func toLower(r rune) rune {
	return r + ('a' - 'A')
}
