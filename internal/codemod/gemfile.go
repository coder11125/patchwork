package codemod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coder11125/patchwork/pkg/domain"
)

type GemfileModifier struct {
	regex *RegexModifier
}

func NewGemfileModifier() *GemfileModifier {
	return &GemfileModifier{
		regex: &RegexModifier{},
	}
}

func (m *GemfileModifier) Name() string {
	return "gemfile"
}

func (m *GemfileModifier) Ecosystem() domain.Ecosystem {
	return domain.EcosystemRuby
}

func (m *GemfileModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	manifestPath := filepath.Join(workDir, "Gemfile")
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

func (m *GemfileModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	return m.regex.ApplyStep(ctx, filePath, step)
}

func (m *GemfileModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading Gemfile: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	updated := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !strings.HasPrefix(trimmed, "gem ") {
			continue
		}

		rest := strings.TrimSpace(trimmed[4:])
		if len(rest) < 2 {
			continue
		}

		quote := rest[0]
		if quote != '"' && quote != '\'' {
			continue
		}

		endIdx := strings.IndexByte(rest[1:], byte(quote))
		if endIdx == -1 {
			continue
		}
		name := rest[1 : endIdx+1]

		if !strings.EqualFold(name, packageName) {
			continue
		}

		afterName := strings.TrimSpace(rest[endIdx+2:])
		if afterName == "" || afterName[0] != ',' {
			continue
		}

		versionPart := strings.TrimSpace(afterName[1:])
		if len(versionPart) < 2 {
			continue
		}

		vQuote := versionPart[0]
		if vQuote != '"' && vQuote != '\'' {
			continue
		}

		vEnd := strings.IndexByte(versionPart[1:], byte(vQuote))
		if vEnd == -1 {
			continue
		}

		var operator string
		versionExpr := versionPart[1 : vEnd+1]
		for _, op := range []string{"~> ", ">= ", "<= ", "> ", "< ", "= "} {
			if strings.HasPrefix(versionExpr, op) {
				operator = op
				break
			}
		}

		oldVersionStr := string(vQuote) + versionExpr + string(vQuote)
		newVersionStr := string(vQuote) + operator + targetVersion + string(vQuote)
		lines[i] = strings.Replace(line, oldVersionStr, newVersionStr, 1)
		updated = true
		break
	}

	if !updated {
		return fmt.Errorf("package %s not found in Gemfile", packageName)
	}

	out := strings.Join(lines, "\n")
	if err := os.WriteFile(manifestPath, []byte(out), 0644); err != nil {
		return fmt.Errorf("writing Gemfile: %w", err)
	}

	return nil
}

func (m *GemfileModifier) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.EqualFold(base, "Gemfile") || strings.EqualFold(base, "Gemfile.lock") || strings.HasSuffix(filePath, ".rb") || strings.HasSuffix(filePath, ".rake")
}
