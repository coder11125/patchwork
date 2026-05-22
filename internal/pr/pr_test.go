package pr

import (
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

func TestNewCreator(t *testing.T) {
	tests := []struct {
		name    string
		cfg     domain.GitConfig
		wantErr bool
	}{
		{
			name: "github with token",
			cfg: domain.GitConfig{
				Platform: "github",
				Token:    "ghp_test",
				Owner:    "owner",
				Repo:     "repo",
			},
			wantErr: false,
		},
		{
			name: "github without token",
			cfg: domain.GitConfig{
				Platform: "github",
				Token:    "",
			},
			wantErr: true,
		},
		{
			name: "gitlab with token",
			cfg: domain.GitConfig{
				Platform: "gitlab",
				Token:    "glpat_test",
				Owner:    "owner",
				Repo:     "repo",
			},
			wantErr: false,
		},
		{
			name: "gitlab without token",
			cfg: domain.GitConfig{
				Platform: "gitlab",
				Token:    "",
			},
			wantErr: true,
		},
		{
			name: "unsupported platform",
			cfg: domain.GitConfig{
				Platform: "bitbucket",
				Token:    "token",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			creator, err := NewCreator(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got creator: %v", creator)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewCreator unexpected error: %v", err)
			}
			if creator == nil {
				t.Fatal("NewCreator returned nil")
			}
		})
	}
}

func TestPRExistsError(t *testing.T) {
	err := &PRExistsError{Message: "pull request already exists"}
	if err.Error() != "pull request already exists" {
		t.Errorf("expected 'pull request already exists', got %q", err.Error())
	}
}

func TestContainsMessage(t *testing.T) {
	tests := []struct {
		body   []byte
		substr string
		want   bool
	}{
		{[]byte("pull request already exists"), "already exists", true},
		{[]byte("open merge request"), "open merge request", true},
		{[]byte("validation failed"), "already exists", false},
		{[]byte(""), "test", false},
	}
	for _, tc := range tests {
		got := containsMessage(tc.body, tc.substr)
		if got != tc.want {
			t.Errorf("containsMessage(%q, %q) = %v, want %v", string(tc.body), tc.substr, got, tc.want)
		}
	}
}
