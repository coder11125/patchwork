package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coder11125/patchwork/internal/keyring"
	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

func LoadConfig(flagSet *pflag.FlagSet) (*domain.Config, error) {
	k := koanf.New(".")

	defaults := DefaultConfig()

	if err := k.Load(confmapProvider(map[string]any{
		"llm_provider":  defaults.LLMProvider,
		"llm_model":     defaults.LLMModel,
		"llm_base_url":  defaults.LLMBaseURL,
		"llm_api_key":   defaults.LLMAPIKey,
		"recipe_dir":    defaults.RecipeDir,
		"episode_dir":   defaults.EpisodeDir,
		"cache_dir":     defaults.CacheDir,
		"max_retries":   defaults.MaxRetries,
		"dry_run":       defaults.DryRun,
		"skip_tests":    defaults.SkipTests,
		"auto_commit":   defaults.AutoCommit,
		"verbose":       defaults.Verbose,
		"git_platform":  defaults.GitPlatform,
		"git_token":     defaults.GitToken,
		"git_owner":     defaults.GitOwner,
		"git_repo":      defaults.GitRepo,
		"git_remote":    defaults.GitRemote,
		"git_pr_branch": defaults.GitPRBranch,
	}), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	configPath := filepath.Join(homeDir, ".patchwork.yaml")

	if _, statErr := os.Stat(configPath); statErr == nil {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load config file %s: %w", configPath, err)
		}
	}

	loadKeyringKeys(k)

	if err := k.Load(env.Provider("PATCHWORK_", ".", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("load env vars: %w", err)
	}

	if flagSet != nil {
		if err := k.Load(posflag.Provider(flagSet, ".", k), nil); err != nil {
			return nil, fmt.Errorf("load cli flags: %w", err)
		}
	}

	var cfg domain.Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.LLMProvider == "" {
		return nil, fmt.Errorf("llm_provider is required")
	}

	return &cfg, nil
}

func loadKeyringKeys(k *koanf.Koanf) {
	if !keyring.IsAvailable() {
		return
	}

	provider := k.String("llm_provider")
	if provider != "" && k.String("llm_api_key") == "" {
		if key, err := keyring.GetLLMAPIKey(provider); err == nil && key != "" {
			_ = k.Set("llm_api_key", key)
		}
	}

	if k.String("git_platform") != "" && k.String("git_token") == "" {
		if token, err := keyring.GetGitToken(k.String("git_platform")); err == nil && token != "" {
			_ = k.Set("git_token", token)
		}
	}
}

type confmapProvider map[string]any

func (p confmapProvider) Read() (map[string]any, error) {
	return p, nil
}

func (p confmapProvider) ReadBytes() ([]byte, error) {
	return nil, fmt.Errorf("confmap provider does not support ReadBytes")
}
