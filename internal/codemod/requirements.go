package codemod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coder11125/patchwork/pkg/domain"
)

type RequirementsModifier struct {
	regex *RegexModifier
}

func NewRequirementsModifier() *RequirementsModifier {
	return &RequirementsModifier{
		regex: &RegexModifier{},
	}
}

func (m *RequirementsModifier) Name() string {
	return "requirements"
}

func (m *RequirementsModifier) Ecosystem() domain.Ecosystem {
	return domain.EcosystemPip
}

func (m *RequirementsModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	manifestPath := filepath.Join(workDir, "requirements.txt")
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

func (m *RequirementsModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	return m.regex.ApplyStep(ctx, filePath, step)
}

func (m *RequirementsModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading requirements.txt: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	updated := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		var pkgName string
		for _, sep := range []string{"==", ">=", "<=", "!=", "~=", ">", "<"} {
			if idx := strings.Index(trimmed, sep); idx != -1 {
				pkgName = strings.TrimSpace(trimmed[:idx])
				break
			}
		}

		if pkgName == "" {
			pkgName = strings.TrimSpace(trimmed)
		}

		if !strings.EqualFold(pkgName, packageName) {
			continue
		}

		lines[i] = fmt.Sprintf("%s==%s", pkgName, targetVersion)
		updated = true
		break
	}

	if !updated {
		return fmt.Errorf("package %s not found in requirements.txt", packageName)
	}

	out := strings.Join(lines, "\n")

	if err := os.WriteFile(manifestPath, []byte(out), 0644); err != nil {
		return fmt.Errorf("writing requirements.txt: %w", err)
	}

	return nil
}

func (m *RequirementsModifier) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.EqualFold(base, "requirements.txt") || strings.EqualFold(base, "setup.py") || strings.EqualFold(base, "pyproject.toml") || strings.HasSuffix(filePath, ".py")
}
