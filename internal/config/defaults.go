package config

import (
	"os"
	"path/filepath"

	"github.com/coder11125/patchwork/pkg/domain"
)

func DefaultConfig() domain.Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}

	return domain.Config{
		LLMProvider: "ollama",
		LLMModel:    "llama3.2",
		LLMBaseURL:  "http://localhost:11434",
		LLMAPIKey:   "",

		RecipeDir:  filepath.Join(homeDir, ".patchwork", "recipes"),
		EpisodeDir: filepath.Join(homeDir, ".patchwork", "episodes"),
		CacheDir:   filepath.Join(homeDir, ".patchwork", "cache"),

		MaxRetries: 3,
		DryRun:     false,
		SkipTests:  false,
		AutoCommit: true,
		Verbose:    false,

		GitPlatform: "",
		GitToken:    "",
		GitOwner:    "",
		GitRepo:     "",
		GitRemote:   "origin",
		GitPRBranch: "main",
	}
}
