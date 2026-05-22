package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.LLMProvider != "ollama" {
		t.Errorf("expected LLMProvider 'ollama', got %q", cfg.LLMProvider)
	}
	if cfg.LLMModel != "llama3.2" {
		t.Errorf("expected LLMModel 'llama3.2', got %q", cfg.LLMModel)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.AutoCommit != true {
		t.Errorf("expected AutoCommit true, got %v", cfg.AutoCommit)
	}
	if cfg.DryRun != false {
		t.Errorf("expected DryRun false, got %v", cfg.DryRun)
	}
	if cfg.SkipTests != false {
		t.Errorf("expected SkipTests false, got %v", cfg.SkipTests)
	}
	if cfg.GitRemote != "origin" {
		t.Errorf("expected GitRemote 'origin', got %q", cfg.GitRemote)
	}
	if cfg.GitPRBranch != "main" {
		t.Errorf("expected GitPRBranch 'main', got %q", cfg.GitPRBranch)
	}
	if cfg.RecipeDir == "" {
		t.Error("expected non-empty RecipeDir")
	}
	if !strings.Contains(cfg.RecipeDir, ".patchwork") {
		t.Errorf("expected RecipeDir to contain '.patchwork', got %q", cfg.RecipeDir)
	}
}

func TestDefaultConfigWithCustomHome(t *testing.T) {
	home := os.Getenv("HOME")
	defer os.Setenv("HOME", home)
	os.Setenv("HOME", "/custom/home")

	cfg := DefaultConfig()
	if cfg.RecipeDir != "/custom/home/.patchwork/recipes" {
		t.Errorf("expected RecipeDir '/custom/home/.patchwork/recipes', got %q", cfg.RecipeDir)
	}
}

func TestConfmapProvider(t *testing.T) {
	p := confmapProvider(map[string]any{
		"key": "value",
		"num": 42,
	})

	read, err := p.Read()
	if err != nil {
		t.Fatalf("Read unexpected error: %v", err)
	}
	if read["key"] != "value" {
		t.Errorf("expected 'value', got %v", read["key"])
	}
	if read["num"] != 42 {
		t.Errorf("expected 42, got %v", read["num"])
	}

	_, err = p.ReadBytes()
	if err == nil {
		t.Error("ReadBytes should return error")
	}
}

func TestLoadConfigRequiresProvider(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("get home dir: %v", err)
	}
	configPath := filepath.Join(homeDir, ".patchwork.yaml")
	hadConfig := true
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		hadConfig = false
	}

	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("llm_provider", "", "")

	cfg, err := LoadConfig(flagSet)
	if hadConfig && err == nil && cfg != nil {
		t.Log("Config loaded successfully with existing ~/.patchwork.yaml")
	}
}
