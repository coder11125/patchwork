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

type CargoDetector struct{}

func (d *CargoDetector) Ecosystem() domain.Ecosystem {
	return domain.EcosystemCargo
}

func (d *CargoDetector) CanHandle(filePath string) bool {
	return filepath.Base(filePath) == "Cargo.toml"
}

func (d *CargoDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	cargoPath := filepath.Join(dir, "Cargo.toml")
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		return nil, nil
	}

	f, err := os.Open(cargoPath)
	if err != nil {
		return nil, fmt.Errorf("open Cargo.toml: %w", err)
	}
	defer f.Close()

	deps, err := parseCargoDependencies(f)
	if err != nil {
		return nil, fmt.Errorf("parse Cargo.toml: %w", err)
	}

	var packages []domain.PackageInfo

	for _, dep := range deps {
		latest, err := fetchLatestCargoVersion(ctx, dep.name)
		if err != nil {
			continue
		}

		packages = append(packages, domain.PackageInfo{
			Name:       dep.name,
			Current:    dep.version,
			Latest:     latest,
			IsOutdated: dep.version != latest && dep.version != "",
			IsDirect:   true,
		})
	}

	return &domain.DetectResult{
		Ecosystem:    domain.EcosystemCargo,
		Dir:          dir,
		ManifestPath: cargoPath,
		Packages:     packages,
		DetectedAt:   time.Now(),
	}, nil
}

type cargoDep struct {
	name    string
	version string
}

func parseCargoDependencies(r io.Reader) ([]cargoDep, error) {
	var deps []cargoDep
	scanner := bufio.NewScanner(r)
	inDeps := false
	inDevDeps := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			section := strings.TrimSpace(line[1 : len(line)-1])
			inDeps = section == "dependencies"
			inDevDeps = section == "dev-dependencies"
			continue
		}

		if !inDeps && !inDevDeps {
			continue
		}

		dep, ok := parseCargoDepLine(line)
		if ok && dep.version != "" {
			deps = append(deps, dep)
		}
	}

	return deps, scanner.Err()
}

func parseCargoDepLine(line string) (cargoDep, bool) {
	eqIdx := strings.IndexByte(line, '=')
	if eqIdx == -1 {
		return cargoDep{}, false
	}

	name := strings.TrimSpace(line[:eqIdx])
	value := strings.TrimSpace(line[eqIdx+1:])

	if name == "" {
		return cargoDep{}, false
	}

	value = strings.TrimSpace(value)

	if strings.HasPrefix(value, "\"") {
		version := strings.Trim(value, "\"")
		return cargoDep{name: name, version: version}, true
	}

	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		inner := value[1 : len(value)-1]
		return cargoDep{name: name, version: extractVersionFromInline(inner)}, true
	}

	if strings.HasPrefix(value, "{") {
		return cargoDep{name: name, version: ""}, true
	}

	return cargoDep{}, false
}

func extractVersionFromInline(inner string) string {
	inner = strings.TrimSpace(inner)
	parts := strings.Split(inner, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "version") {
			eqIdx := strings.IndexByte(part, '=')
			if eqIdx == -1 {
				continue
			}
			ver := strings.TrimSpace(part[eqIdx+1:])
			ver = strings.Trim(ver, "\"")
			return ver
		}
	}
	return ""
}

func fetchLatestCargoVersion(ctx context.Context, crateName string) (string, error) {
	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s", crateName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request for %s: %w", crateName, err)
	}
	req.Header.Set("User-Agent", "patchwork/1.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest for %s: %w", crateName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("crates.io returned %d for %s", resp.StatusCode, crateName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response for %s: %w", crateName, err)
	}

	var info struct {
		Crate struct {
			MaxVersion       string `json:"max_version"`
			MaxStableVersion string `json:"max_stable_version"`
		} `json:"crate"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("unmarshal response for %s: %w", crateName, err)
	}

	if info.Crate.MaxStableVersion != "" {
		return info.Crate.MaxStableVersion, nil
	}
	return info.Crate.MaxVersion, nil
}
