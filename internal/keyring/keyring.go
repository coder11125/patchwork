package keyring

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const llmService = "patchwork-llm"
const gitService = "patchwork-git"

func GetLLMAPIKey(provider string) (string, error) {
	key, err := keyring.Get(llmService, provider)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	return key, err
}

func SetLLMAPIKey(provider, key string) error {
	return keyring.Set(llmService, provider, key)
}

func DeleteLLMAPIKey(provider string) error {
	return keyring.Delete(llmService, provider)
}

func GetGitToken(platform string) (string, error) {
	key, err := keyring.Get(gitService, platform)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	return key, err
}

func SetGitToken(platform, token string) error {
	return keyring.Set(gitService, platform, token)
}

func DeleteGitToken(platform string) error {
	return keyring.Delete(gitService, platform)
}

func IsAvailable() bool {
	_, err := keyring.Get("patchwork-check", "availability")
	return err == nil || errors.Is(err, keyring.ErrNotFound)
}

var ErrNotAvailable = fmt.Errorf("OS keychain not available")
