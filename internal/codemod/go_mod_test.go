package codemod

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoModModifierName(t *testing.T) {
	m := NewGoModModifier()
	if m.Name() != "go-mod" {
		t.Errorf("expected name 'go-mod', got %q", m.Name())
	}
}

func TestGoModModifierEcosystem(t *testing.T) {
	m := NewGoModModifier()
	if m.Ecosystem().String() != "go" {
		t.Errorf("expected ecosystem 'go', got %q", m.Ecosystem())
	}
}

func TestUpdateManifestAddsRequire(t *testing.T) {
	m := NewGoModModifier()
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")

	content := []byte("module example.com/test\n\ngo 1.21\n")
	if err := os.WriteFile(modPath, content, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	err := m.UpdateManifest(context.Background(), modPath, "example.com/pkg", "v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(data), "example.com/pkg") {
		t.Errorf("expected go.mod to contain 'example.com/pkg', got:\n%s", string(data))
	}
}

func TestUpdateManifestUpdatesExisting(t *testing.T) {
	m := NewGoModModifier()
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")

	content := []byte("module example.com/test\n\ngo 1.21\n\nrequire example.com/pkg v1.0.0\n")
	if err := os.WriteFile(modPath, content, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	err := m.UpdateManifest(context.Background(), modPath, "example.com/pkg", "v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(data), "v2.0.0") {
		t.Errorf("expected go.mod to contain 'v2.0.0', got:\n%s", string(data))
	}
}

func TestUpdateManifestAlreadyCurrent(t *testing.T) {
	m := NewGoModModifier()
	dir := t.TempDir()
	modPath := filepath.Join(dir, "go.mod")

	content := []byte("module example.com/test\n\ngo 1.21\n\nrequire example.com/pkg v1.1.0\n")
	if err := os.WriteFile(modPath, content, 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	err := m.UpdateManifest(context.Background(), modPath, "example.com/pkg", "v1.0.0", "v1.1.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}
}

func TestCanHandleGoMod(t *testing.T) {
	m := NewGoModModifier()
	if !m.CanHandle("go.mod") {
		t.Error("expected CanHandle('go.mod') to be true")
	}
	if m.CanHandle("package.json") {
		t.Error("expected CanHandle('package.json') to be false")
	}
}
