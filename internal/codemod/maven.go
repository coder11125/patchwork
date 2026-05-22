package codemod

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coder11125/patchwork/pkg/domain"
)

type MavenModifier struct {
	regex *RegexModifier
}

func NewMavenModifier() *MavenModifier {
	return &MavenModifier{
		regex: &RegexModifier{},
	}
}

func (m *MavenModifier) Name() string {
	return "maven"
}

func (m *MavenModifier) Ecosystem() domain.Ecosystem {
	return domain.EcosystemMaven
}

func (m *MavenModifier) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	manifestPath := filepath.Join(workDir, "pom.xml")
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

func (m *MavenModifier) ApplyStep(ctx context.Context, filePath string, step domain.RecipeStep) error {
	return m.regex.ApplyStep(ctx, filePath, step)
}

func (m *MavenModifier) UpdateManifest(ctx context.Context, manifestPath string, packageName, currentVersion, targetVersion string) error {
	parts := strings.SplitN(packageName, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid maven package name %q, expected groupId:artifactId", packageName)
	}
	groupID := parts[0]
	artifactID := parts[1]

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading pom.xml: %w", err)
	}

	text := string(content)

	updated, err := tryUpdateInline(text, groupID, artifactID, targetVersion)
	if err != nil {
		return err
	}
	if updated != text {
		return writeXML(manifestPath, updated)
	}

	updated, err = tryUpdatePropertyRef(text, groupID, artifactID, targetVersion)
	if err != nil {
		return err
	}
	if updated != text {
		return writeXML(manifestPath, updated)
	}

	updated, err = tryUpdateParent(text, groupID, artifactID, targetVersion)
	if err != nil {
		return err
	}
	if updated != text {
		return writeXML(manifestPath, updated)
	}

	return fmt.Errorf("package %s not found in pom.xml", packageName)
}

func tryUpdateInline(text, groupID, artifactID, targetVersion string) (string, error) {
	re := regexp.MustCompile(`(?s)(<groupId>\s*` + regexp.QuoteMeta(groupID) + `\s*</groupId>\s*\n\s*<artifactId>\s*` + regexp.QuoteMeta(artifactID) + `\s*</artifactId>\s*\n\s*)<version>\s*([^<]+)\s*</version>`)
	matches := re.FindStringSubmatch(text)
	if matches == nil {
		re = regexp.MustCompile(`(?s)(<artifactId>\s*` + regexp.QuoteMeta(artifactID) + `\s*</artifactId>\s*\n\s*<groupId>\s*` + regexp.QuoteMeta(groupID) + `\s*</groupId>\s*\n\s*)<version>\s*([^<]+)\s*</version>`)
		matches = re.FindStringSubmatch(text)
	}
	if matches == nil {
		return text, nil
	}

	oldVersion := matches[2]
	if isPropertyRef(oldVersion) {
		return text, nil
	}

	return re.ReplaceAllString(text, "${1}<version>"+targetVersion+"</version>"), nil
}

func tryUpdatePropertyRef(text, groupID, artifactID, targetVersion string) (string, error) {
	propRefRe := regexp.MustCompile(`(?s)(<groupId>\s*` + regexp.QuoteMeta(groupID) + `\s*</groupId>\s*\n\s*<artifactId>\s*` + regexp.QuoteMeta(artifactID) + `\s*</artifactId>\s*\n\s*)<version>\s*\$\{([^}]+)\}\s*</version>`)
	matches := propRefRe.FindStringSubmatch(text)
	if matches == nil {
		propRefRe = regexp.MustCompile(`(?s)(<artifactId>\s*` + regexp.QuoteMeta(artifactID) + `\s*</artifactId>\s*\n\s*<groupId>\s*` + regexp.QuoteMeta(groupID) + `\s*</groupId>\s*\n\s*)<version>\s*\$\{([^}]+)\}\s*</version>`)
		matches = propRefRe.FindStringSubmatch(text)
	}
	if matches == nil {
		return text, nil
	}

	propName := matches[2]
	return updateProperty(text, propName, targetVersion)
}

func tryUpdateParent(text, groupID, artifactID, targetVersion string) (string, error) {
	parentRe := regexp.MustCompile(`(?s)(<parent>\s*\n\s*<groupId>\s*` + regexp.QuoteMeta(groupID) + `\s*</groupId>\s*\n\s*<artifactId>\s*` + regexp.QuoteMeta(artifactID) + `\s*</artifactId>\s*\n\s*)<version>\s*([^<]+)\s*</version>`)
	matches := parentRe.FindStringSubmatch(text)
	if matches == nil {
		return text, nil
	}

	oldVersion := matches[2]
	if isPropertyRef(oldVersion) {
		propName := oldVersion[2 : len(oldVersion)-1]
		return updateProperty(text, propName, targetVersion)
	}

	return parentRe.ReplaceAllString(text, "${1}<version>"+targetVersion+"</version>"), nil
}

func updateProperty(text, propName, targetVersion string) (string, error) {
	propRe := regexp.MustCompile(`<` + regexp.QuoteMeta(propName) + `\s*>\s*([^<]+)\s*</` + regexp.QuoteMeta(propName) + `\s*>`)
	matches := propRe.FindStringSubmatch(text)
	if matches == nil {
		return text, nil
	}
	return propRe.ReplaceAllString(text, "<"+propName+">"+targetVersion+"</"+propName+">"), nil
}

func isPropertyRef(version string) bool {
	return strings.HasPrefix(version, "${") && strings.HasSuffix(version, "}")
}

func writeXML(path, content string) error {
	data := []byte(content)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing pom.xml: %w", err)
	}
	return nil
}

func (m *MavenModifier) CanHandle(filePath string) bool {
	base := filepath.Base(filePath)
	return strings.EqualFold(base, "pom.xml") || strings.HasSuffix(filePath, ".java")
}
