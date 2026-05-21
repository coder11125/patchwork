package codemod

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/coder11125/patchwork/pkg/domain"
)

type RegexModifier struct{}

func NewRegexModifier() *RegexModifier {
	return &RegexModifier{}
}

func (m *RegexModifier) Name() string {
	return "regex"
}

func (m *RegexModifier) Ecosystem() domain.Ecosystem {
	return ""
}

func (m *RegexModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
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

func (m *RegexModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	if step.Pattern == "" {
		return nil
	}

	if len(step.FileGlobs) > 0 {
		matched := false
		for _, glob := range step.FileGlobs {
			ok, err := path.Match(glob, filePath)
			if err != nil {
				return fmt.Errorf("invalid glob pattern %q: %w", glob, err)
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
	}

	re, err := regexp.Compile(step.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", step.Pattern, err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filePath, err)
	}

	updated := re.ReplaceAllString(string(content), step.Replacement)

	if string(content) == updated {
		return nil
	}

	if err := os.WriteFile(filePath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", filePath, err)
	}

	return nil
}

func (m *RegexModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	return nil
}

func (m *RegexModifier) CanHandle(filePath string) bool {
	return true
}
