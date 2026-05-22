package detector

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type MavenDetector struct{}

func (d *MavenDetector) Ecosystem() domain.Ecosystem {
	return domain.EcosystemMaven
}

func (d *MavenDetector) CanHandle(filePath string) bool {
	return filepath.Base(filePath) == "pom.xml"
}

type mavenProject struct {
	XMLName              xml.Name            `xml:"project"`
	GroupID              string              `xml:"groupId"`
	ArtifactID           string              `xml:"artifactId"`
	Version              string              `xml:"version"`
	Parent               *mavenParent        `xml:"parent"`
	Properties           *mavenProperties    `xml:"properties"`
	Dependencies         *mavenDependencies  `xml:"dependencies"`
	DependencyManagement *mavenDepManagement `xml:"dependencyManagement"`
}

type mavenParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type mavenProperties struct {
	Entries []mavenProperty `xml:",any"`
}

type mavenProperty struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type mavenDepManagement struct {
	Dependencies *mavenDependencies `xml:"dependencies"`
}

type mavenDependencies struct {
	Dependencies []mavenDependency `xml:"dependency"`
}

type mavenDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
}

func (d *MavenDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	pomPath := filepath.Join(dir, "pom.xml")
	if _, err := os.Stat(pomPath); os.IsNotExist(err) {
		return nil, nil
	}

	project, err := parsePOM(pomPath)
	if err != nil {
		return nil, fmt.Errorf("parse pom.xml: %w", err)
	}

	props := buildPropertyMap(project)

	managed := make(map[string]string)
	if project.DependencyManagement != nil && project.DependencyManagement.Dependencies != nil {
		for _, dep := range project.DependencyManagement.Dependencies.Dependencies {
			key := dep.GroupID + ":" + dep.ArtifactID
			managed[key] = resolveVersion(dep.Version, props)
		}
	}

	seen := make(map[string]bool)
	var packages []domain.PackageInfo

	if project.Dependencies != nil {
		for _, dep := range project.Dependencies.Dependencies {
			key := dep.GroupID + ":" + dep.ArtifactID
			if seen[key] {
				continue
			}
			seen[key] = true

			version := resolveVersion(dep.Version, props)
			if version == "" {
				version = managed[key]
			}
			if version == "" {
				continue
			}

			latest, err := fetchLatestMavenVersion(ctx, dep.GroupID, dep.ArtifactID)
			if err != nil {
				continue
			}

			packages = append(packages, domain.PackageInfo{
				Name:       key,
				Current:    version,
				Latest:     latest,
				IsOutdated: isVersionOutdated(version, latest),
				IsDirect:   true,
			})
		}
	}

	if project.Parent != nil && project.Parent.Version != "" {
		version := resolveVersion(project.Parent.Version, props)
		if version != "" {
			latest, err := fetchLatestMavenVersion(ctx, project.Parent.GroupID, project.Parent.ArtifactID)
			if err == nil {
				key := project.Parent.GroupID + ":" + project.Parent.ArtifactID
				packages = append(packages, domain.PackageInfo{
					Name:       key,
					Current:    version,
					Latest:     latest,
			IsOutdated: isVersionOutdated(version, latest),
					IsDirect:   true,
				})
			}
		}
	}

	return &domain.DetectResult{
		Ecosystem:    domain.EcosystemMaven,
		Dir:          dir,
		ManifestPath: pomPath,
		Packages:     packages,
		DetectedAt:   time.Now(),
	}, nil
}

func parsePOM(path string) (*mavenProject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var project mavenProject
	if err := xml.Unmarshal(data, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func buildPropertyMap(project *mavenProject) map[string]string {
	props := make(map[string]string)
	if project.Properties != nil {
		for _, entry := range project.Properties.Entries {
			name := entry.XMLName.Local
			if name != "" {
				props[name] = entry.Value
			}
		}
	}
	if project.Version != "" {
		props["project.version"] = project.Version
	}
	if project.GroupID != "" {
		props["project.groupId"] = project.GroupID
	}
	return props
}

func resolveVersion(version string, props map[string]string) string {
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "${") && strings.HasSuffix(version, "}") {
		key := version[2 : len(version)-1]
		if v, ok := props[key]; ok {
			return v
		}
		return ""
	}
	return version
}

func fetchLatestMavenVersion(ctx context.Context, groupID, artifactID string) (string, error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("g:%s+AND+a:%s", groupID, artifactID))
	query.Set("rows", "1")
	query.Set("wt", "json")

	url := fmt.Sprintf("https://search.maven.org/solrsearch/select?%s", query.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request for %s:%s: %w", groupID, artifactID, err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest for %s:%s: %w", groupID, artifactID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("maven search returned %d for %s:%s", resp.StatusCode, groupID, artifactID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response for %s:%s: %w", groupID, artifactID, err)
	}

	var result struct {
		Response struct {
			Docs []struct {
				LatestVersion string `json:"latestVersion"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response for %s:%s: %w", groupID, artifactID, err)
	}

	if len(result.Response.Docs) == 0 {
		return "", fmt.Errorf("no results for %s:%s", groupID, artifactID)
	}

	return result.Response.Docs[0].LatestVersion, nil
}
