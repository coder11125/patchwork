package codemod

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCargoModifierName(t *testing.T) {
	m := NewCargoModifier()
	if m.Name() != "cargo" {
		t.Errorf("expected name 'cargo', got %q", m.Name())
	}
}

func TestCargoModifierEcosystem(t *testing.T) {
	m := NewCargoModifier()
	if m.Ecosystem().String() != "cargo" {
		t.Errorf("expected ecosystem 'cargo', got %q", m.Ecosystem())
	}
}

func TestCargoUpdateSimpleVersion(t *testing.T) {
	m := NewCargoModifier()
	dir := t.TempDir()
	cargoPath := filepath.Join(dir, "Cargo.toml")

	content := []byte("[dependencies]\nserde = \"1.0.0\"\ntokio = \"1.0.0\"\n")
	if err := os.WriteFile(cargoPath, content, 0644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}

	err := m.UpdateManifest(context.Background(), cargoPath, "serde", "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(cargoPath)
	if err != nil {
		t.Fatalf("failed to read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(data), "serde = \"2.0.0\"") {
		t.Errorf("expected Cargo.toml to contain 'serde = \"2.0.0\"', got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "tokio = \"1.0.0\"") {
		t.Errorf("expected other deps to remain, got:\n%s", string(data))
	}
}

func TestCargoUpdateInlineTableVersion(t *testing.T) {
	m := NewCargoModifier()
	dir := t.TempDir()
	cargoPath := filepath.Join(dir, "Cargo.toml")

	content := []byte("[dependencies]\nserde = { version = \"1.0.0\", features = [\"derive\"] }\n")
	if err := os.WriteFile(cargoPath, content, 0644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}

	err := m.UpdateManifest(context.Background(), cargoPath, "serde", "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}

	data, err := os.ReadFile(cargoPath)
	if err != nil {
		t.Fatalf("failed to read Cargo.toml: %v", err)
	}
	if !strings.Contains(string(data), "version = \"2.0.0\"") {
		t.Errorf("expected Cargo.toml to contain 'version = \"2.0.0\"', got:\n%s", string(data))
	}
}

func TestCargoPackageNotFound(t *testing.T) {
	m := NewCargoModifier()
	dir := t.TempDir()
	cargoPath := filepath.Join(dir, "Cargo.toml")

	content := []byte("[dependencies]\nserde = \"1.0.0\"\n")
	if err := os.WriteFile(cargoPath, content, 0644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}

	err := m.UpdateManifest(context.Background(), cargoPath, "nonexistent", "1.0.0", "2.0.0")
	if err == nil {
		t.Error("expected error for non-existent package")
	}
}

func TestCargoSkipsSections(t *testing.T) {
	m := NewCargoModifier()
	dir := t.TempDir()
	cargoPath := filepath.Join(dir, "Cargo.toml")

	content := []byte("[package]\nname = \"test\"\n\n[dependencies]\nserde = \"1.0.0\"\n")
	if err := os.WriteFile(cargoPath, content, 0644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}

	err := m.UpdateManifest(context.Background(), cargoPath, "serde", "1.0.0", "2.0.0")
	if err != nil {
		t.Fatalf("UpdateManifest unexpected error: %v", err)
	}
}

func TestUpdateInlineTableVersion(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		pkgName     string
		targetVer   string
		wantContain string
	}{
		{
			name:        "simple inline table",
			line:        `serde = { version = "1.0.0", features = ["derive"] }`,
			pkgName:     "serde",
			targetVer:   "2.0.0",
			wantContain: `version = "2.0.0"`,
		},
		{
			name:        "no version field",
			line:        `serde = { features = ["derive"] }`,
			pkgName:     "serde",
			targetVer:   "2.0.0",
			wantContain: `features = ["derive"]`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := updateInlineTableVersion(tc.line, tc.pkgName, tc.targetVer)
			if !strings.Contains(result, tc.wantContain) {
				t.Errorf("expected result to contain %q, got %q", tc.wantContain, result)
			}
		})
	}
}


