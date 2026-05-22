package codemod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coder11125/patchwork/pkg/domain"
)

type CargoModifier struct {
	regex *RegexModifier
}

func NewCargoModifier() *CargoModifier {
	return &CargoModifier{
		regex: &RegexModifier{},
	}
}

func (m *CargoModifier) Name() string {
	return "cargo"
}

func (m *CargoModifier) Ecosystem() domain.Ecosystem {
	return domain.EcosystemCargo
}

func (m *CargoModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	manifestPath := filepath.Join(workDir, "Cargo.toml")
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

func (m *CargoModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	return m.regex.ApplyStep(ctx, filePath, step)
}

func (m *CargoModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading Cargo.toml: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	updated := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed[0] == '[' {
			continue
		}

		eqIdx := strings.IndexByte(trimmed, '=')
		if eqIdx == -1 {
			continue
		}

		name := strings.TrimSpace(trimmed[:eqIdx])
		if !strings.EqualFold(name, packageName) {
			continue
		}

		value := strings.TrimSpace(trimmed[eqIdx+1:])

		if strings.HasPrefix(value, "\"") {
			lines[i] = fmt.Sprintf("%s = \"%s\"", name, targetVersion)
			updated = true
			break
		}

		if strings.HasPrefix(value, "{") {
			lines[i] = updateInlineTableVersion(line, name, targetVersion)
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("package %s not found in Cargo.toml", packageName)
	}

	out := strings.Join(lines, "\n")

	if err := os.WriteFile(manifestPath, []byte(out), 0644); err != nil {
		return fmt.Errorf("writing Cargo.toml: %w", err)
	}

	return nil
}

func updateInlineTableVersion(line, packageName, targetVersion string) string {
	eqIdx := strings.IndexByte(line, '=')
	value := strings.TrimSpace(line[eqIdx+1:])

	if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
		inner := value[1 : len(value)-1]
		parts := strings.Split(inner, ",")
		var found bool
		for j, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "version") {
				verEqIdx := strings.IndexByte(part, '=')
				if verEqIdx != -1 {
					parts[j] = fmt.Sprintf("version = \"%s\"", targetVersion)
					found = true
				}
			}
		}
		if found {
			newValue := "{ " + strings.Join(parts, ", ") + " }"
			return fmt.Sprintf("%s = %s", strings.TrimSpace(line[:eqIdx]), newValue)
		}
	}

	return line
}
