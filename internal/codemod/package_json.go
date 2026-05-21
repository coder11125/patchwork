package codemod

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coder11125/patchwork/pkg/domain"
)

type PackageJSONModifier struct {
	regex *RegexModifier
}

func NewPackageJSONModifier() *PackageJSONModifier {
	return &PackageJSONModifier{
		regex: &RegexModifier{},
	}
}

func (m *PackageJSONModifier) Name() string {
	return "package-json"
}

func (m *PackageJSONModifier) Ecosystem() domain.Ecosystem {
	return domain.EcosystemNPM
}

func (m *PackageJSONModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	manifestPath := filepath.Join(workDir, "package.json")
	if _, err := os.Stat(manifestPath); err == nil {
		if err := m.UpdateManifest(ctx, manifestPath, recipe.PackageName, recipe.FromVersion, recipe.ToVersion); err != nil {
			return err
		}
	}
	for _, step := range recipe.Steps {
		if step.Type == "manual" {
			continue
		}
		if len(step.FileGlobs) == 0 {
			continue
		}
		for _, glob := range step.FileGlobs {
			matches, err := filepath.Glob(filepath.Join(workDir, glob))
			if err != nil {
				return fmt.Errorf("glob pattern %q: %w", glob, err)
			}
			for _, match := range matches {
				if err := m.ApplyStep(ctx, match, step); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (m *PackageJSONModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	return m.regex.ApplyStep(ctx, filePath, step)
}

func (m *PackageJSONModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading package.json: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("parsing package.json: %w", err)
	}

	depKeys := []string{"dependencies", "devDependencies", "peerDependencies"}
	updated := false

	for _, key := range depKeys {
		deps, ok := data[key]
		if !ok {
			continue
		}
		depMap, ok := deps.(map[string]interface{})
		if !ok {
			continue
		}
		if _, exists := depMap[packageName]; exists {
			depMap[packageName] = targetVersion
			data[key] = depMap
			updated = true
		}
	}

	if !updated {
		return fmt.Errorf("package %s not found in any dependency section", packageName)
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling package.json: %w", err)
	}

	out = append(out, '\n')

	if err := os.WriteFile(manifestPath, out, 0644); err != nil {
		return fmt.Errorf("writing package.json: %w", err)
	}

	return nil
}

func (m *PackageJSONModifier) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.EqualFold(base, "package.json") || strings.EqualFold(base, "package-lock.json") || strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".jsx") || strings.HasSuffix(filePath, ".tsx")
}
