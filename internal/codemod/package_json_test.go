package codemod

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageJSONModifierName(t *testing.T) {
	m := NewPackageJSONModifier()
	if m.Name() != "package-json" {
		t.Errorf("expected name 'package-json', got %q", m.Name())
	}
}

func TestPackageJSONModifierEcosystem(t *testing.T) {
	m := NewPackageJSONModifier()
	if m.Ecosystem().String() != "npm" {
		t.Errorf("expected ecosystem 'npm', got %q", m.Ecosystem())
	}
}

func TestPackageJSONUpdateManifest(t *testing.T) {
	m := NewPackageJSONModifier()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	content := []byte(`{
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`)
	if err := os.WriteFile(pkgPath, content, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	err := m.UpdateManifest(context.Background(), pkgPath, "lodash", "^4.17.21", "^5.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	if !strings.Contains(string(data), "^5.0.0") {
		t.Errorf("expected package.json to contain '^5.0.0', got:\n%s", string(data))
	}
}

func TestPackageJSONUpdateDevDependencies(t *testing.T) {
	m := NewPackageJSONModifier()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	content := []byte(`{
  "devDependencies": {
    "jest": "^29.0.0"
  }
}`)
	if err := os.WriteFile(pkgPath, content, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	err := m.UpdateManifest(context.Background(), pkgPath, "jest", "^29.0.0", "^30.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	if !strings.Contains(string(data), "^30.0.0") {
		t.Errorf("expected package.json to contain '^30.0.0', got:\n%s", string(data))
	}
}

func TestPackageJSONUpdatePeerDependencies(t *testing.T) {
	m := NewPackageJSONModifier()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	content := []byte(`{
  "peerDependencies": {
    "react": "^18.0.0"
  }
}`)
	if err := os.WriteFile(pkgPath, content, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	err := m.UpdateManifest(context.Background(), pkgPath, "react", "^18.0.0", "^19.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}
	if !strings.Contains(string(data), "^19.0.0") {
		t.Errorf("expected package.json to contain '^19.0.0', got:\n%s", string(data))
	}
}

func TestPackageJSONPackageNotFound(t *testing.T) {
	m := NewPackageJSONModifier()
	dir := t.TempDir()
	pkgPath := filepath.Join(dir, "package.json")

	content := []byte(`{"dependencies": {"express": "^4.0.0"}}`)
	if err := os.WriteFile(pkgPath, content, 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	err := m.UpdateManifest(context.Background(), pkgPath, "nonexistent", "1.0.0", "2.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}
}

func TestCanHandlePackageJSON(t *testing.T) {
	m := NewPackageJSONModifier()
	if !m.CanHandle("package.json") {
		t.Error("expected CanHandle('package.json') to be true")
	}
	if m.CanHandle("go.mod") {
		t.Error("expected CanHandle('go.mod') to be false")
	}
}
