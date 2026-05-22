package codemod

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

type mockCodemod struct {
	name      string
	ecosystem domain.Ecosystem
}

func (m *mockCodemod) Name() string                { return m.name }
func (m *mockCodemod) Ecosystem() domain.Ecosystem { return m.ecosystem }
func (m *mockCodemod) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	return nil
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	c := &mockCodemod{name: "test-mod", ecosystem: domain.EcosystemGo}
	r.Register(c)

	got, err := r.Get("test-mod")
	if err != nil {
		t.Fatalf("Get unexpected error: %v", err)
	}
	if got != c {
		t.Error("Get returned wrong codemod")
	}
}

func TestGetNotFound(t *testing.T) {
	r := NewRegistry()
	got, err := r.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent codemod, got %v", got)
	}
}

func TestApplyDelegatesToMatchingCodemod(t *testing.T) {
	r := NewRegistry()
	applied := false
	r.Register(&mockCodemod{
		name:      "go-mod",
		ecosystem: domain.EcosystemGo,
	})

	r.codemods["go-mod"] = &customApplyCodemod{name: "go-mod", ecosystem: domain.EcosystemGo, applyFn: func(ctx context.Context, workDir string, recipe *domain.Recipe) error {
		applied = true
		return nil
	}}

	recipe := &domain.Recipe{Ecosystem: domain.EcosystemGo}
	err := r.Apply(context.Background(), "/test", recipe)
	if err != nil {
		t.Fatalf("Apply unexpected error: %v", err)
	}
	if !applied {
		t.Error("expected Apply to call the matching codemod")
	}
}

type customApplyCodemod struct {
	name      string
	ecosystem domain.Ecosystem
	applyFn   func(ctx context.Context, workDir string, recipe *domain.Recipe) error
}

func (c *customApplyCodemod) Name() string                { return c.name }
func (c *customApplyCodemod) Ecosystem() domain.Ecosystem { return c.ecosystem }
func (c *customApplyCodemod) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	return c.applyFn(ctx, workDir, recipe)
}

func TestRegexModifierApplyStep(t *testing.T) {
	m := NewRegexModifier()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	step := domain.RecipeStep{
		Pattern:     "hello",
		Replacement: "hi",
	}

	if err := m.ApplyStep(context.Background(), filePath, step); err != nil {
		t.Fatalf("ApplyStep unexpected error: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "hi world" {
		t.Errorf("expected 'hi world', got %q", string(content))
	}
}

func TestRegexModifierApplyStepNoMatch(t *testing.T) {
	m := NewRegexModifier()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	step := domain.RecipeStep{
		Pattern:     "zzz",
		Replacement: "hi",
		FileGlobs:   []string{"*.txt"},
	}

	if err := m.ApplyStep(context.Background(), filePath, step); err != nil {
		t.Fatalf("ApplyStep unexpected error: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected content unchanged, got %q", string(content))
	}
}

func TestRegexModifierApplyStepEmptyPattern(t *testing.T) {
	m := NewRegexModifier()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	step := domain.RecipeStep{Pattern: ""}
	if err := m.ApplyStep(context.Background(), filePath, step); err != nil {
		t.Fatalf("ApplyStep with empty pattern should not error: %v", err)
	}
}

func TestRegexModifierApplyStepInvalidRegex(t *testing.T) {
	m := NewRegexModifier()
	step := domain.RecipeStep{Pattern: `\z\q`}
	err := m.ApplyStep(context.Background(), "/nonexistent/test.txt", step)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestRegexModifierNameAndEcosystem(t *testing.T) {
	m := NewRegexModifier()
	if m.Name() != "regex" {
		t.Errorf("expected name 'regex', got %q", m.Name())
	}
	if m.Ecosystem() != "" {
		t.Errorf("expected empty ecosystem for regex modifier, got %q", m.Ecosystem())
	}
}
