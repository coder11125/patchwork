package keyring

import (
	"testing"
)

func TestIsAvailable(t *testing.T) {
	available := IsAvailable()
	if !available {
		t.Skip("OS keychain not available on this system, skipping")
	}
}

func TestGetLLMAPIKeyNotFound(t *testing.T) {
	if !IsAvailable() {
		t.Skip("OS keychain not available")
	}
	key, err := GetLLMAPIKey("nonexistent-provider")
	if err != nil {
		t.Fatalf("GetLLMAPIKey unexpected error: %v", err)
	}
	if key != "" {
		t.Logf("expected empty key, got %q", key)
	}
}

func TestErrNotAvailable(t *testing.T) {
	if ErrNotAvailable == nil {
		t.Error("ErrNotAvailable should not be nil")
	}
	if ErrNotAvailable.Error() == "" {
		t.Error("ErrNotAvailable should have a non-empty error message")
	}
}
