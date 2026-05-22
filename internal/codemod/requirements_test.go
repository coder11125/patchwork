package codemod

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequirementsModifierName(t *testing.T) {
	m := NewRequirementsModifier()
	if m.Name() != "requirements" {
		t.Errorf("expected name 'requirements', got %q", m.Name())
	}
}

func TestRequirementsModifierEcosystem(t *testing.T) {
	m := NewRequirementsModifier()
	if m.Ecosystem().String() != "pip" {
		t.Errorf("expected ecosystem 'pip', got %q", m.Ecosystem())
	}
}

func TestRequirementsUpdateManifest(t *testing.T) {
	m := NewRequirementsModifier()
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "requirements.txt")

	content := []byte("requests==2.28.0\nnumpy==1.24.0\n")
	if err := os.WriteFile(reqPath, content, 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	err := m.UpdateManifest(context.Background(), reqPath, "requests", "2.28.0", "2.31.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("failed to read requirements.txt: %v", err)
	}
	if !strings.Contains(string(data), "requests==2.31.0") {
		t.Errorf("expected requirements.txt to contain 'requests==2.31.0', got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "numpy==1.24.0") {
		t.Errorf("expected other packages to remain, got:\n%s", string(data))
	}
}

func TestRequirementsUpdateManifestWithWhitespace(t *testing.T) {
	m := NewRequirementsModifier()
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "requirements.txt")

	content := []byte("requests==2.28.0\n# comment\n\nnumpy==1.24.0\n")
	if err := os.WriteFile(reqPath, content, 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	err := m.UpdateManifest(context.Background(), reqPath, "numpy", "1.24.0", "1.26.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatalf("failed to read requirements.txt: %v", err)
	}
	if !strings.Contains(string(data), "numpy==1.26.0") {
		t.Errorf("expected numpy version to be updated, got:\n%s", string(data))
	}
}

func TestRequirementsPackageNotFound(t *testing.T) {
	m := NewRequirementsModifier()
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "requirements.txt")

	content := []byte("requests==2.28.0\n")
	if err := os.WriteFile(reqPath, content, 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	err := m.UpdateManifest(context.Background(), reqPath, "nonexistent", "1.0.0", "2.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}
}

func TestRequirementsCaseInsensitiveMatch(t *testing.T) {
	m := NewRequirementsModifier()
	dir := t.TempDir()
	reqPath := filepath.Join(dir, "requirements.txt")

	content := []byte("Django==4.2.0\n")
	if err := os.WriteFile(reqPath, content, 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	err := m.UpdateManifest(context.Background(), reqPath, "django", "4.2.0", "5.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}
}

func TestCanHandleRequirements(t *testing.T) {
	m := NewRequirementsModifier()
	if !m.CanHandle("requirements.txt") {
		t.Error("expected CanHandle('requirements.txt') to be true")
	}
	if m.CanHandle("go.mod") {
		t.Error("expected CanHandle('go.mod') to be false")
	}
}
