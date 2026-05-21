package codemod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/coder11125/patchwork/pkg/domain"
)

type GoModModifier struct {
	regex *RegexModifier
}

func NewGoModModifier() *GoModModifier {
	return &GoModModifier{
		regex: &RegexModifier{},
	}
}

func (m *GoModModifier) Name() string {
	return "go-mod"
}

func (m *GoModModifier) Ecosystem() domain.Ecosystem {
	return domain.EcosystemGo
}

func (m *GoModModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	manifestPath := filepath.Join(workDir, "go.mod")
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

func (m *GoModModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	return m.regex.ApplyStep(ctx, filePath, step)
}

func (m *GoModModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	f, err := modfile.Parse(manifestPath, content, nil)
	if err != nil {
		return fmt.Errorf("parsing go.mod: %w", err)
	}

	updated := false
	for _, req := range f.Require {
		if req.Mod.Path == packageName {
			if req.Mod.Version == targetVersion {
				return nil
			}
			if err := f.AddRequire(req.Mod.Path, targetVersion); err != nil {
				return fmt.Errorf("updating require directive for %s: %w", packageName, err)
			}
			updated = true
			break
		}
	}

	if !updated {
		if err := f.AddRequire(packageName, targetVersion); err != nil {
			return fmt.Errorf("adding require directive for %s: %w", packageName, err)
		}
	}

	out, err := f.Format()
	if err != nil {
		return fmt.Errorf("formatting go.mod: %w", err)
	}

	if err := os.WriteFile(manifestPath, out, 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	return nil
}

func (m *GoModModifier) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.EqualFold(base, "go.mod") || strings.EqualFold(base, "go.sum") || strings.HasSuffix(filePath, ".go")
}
