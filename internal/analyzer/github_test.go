package analyzer

import (
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

func TestIsBreakingLine(t *testing.T) {
	tests := []struct {
		line  string
		want  bool
	}{
		{"BREAKING: changed API", true},
		{"breaking: changed API", true},
		{"MAJOR CHANGE: removed function", true},
		{"BC: incompatible", true},
		{"Deprecated: old method", true},
		{"removed support for X", true},
		{"dropped support for Y", true},
		{"no longer supports Z", true},
		{"incompatible version", true},
		{"migration required", true},
		{"migration needed", true},
		{"API change", true},
		{"signature change", true},
		{"breaking change", true},
		{"added new feature", false},
		{"fixed bug", false},
		{"updated documentation", false},
		{"", false},
		{"performance improvement", false},
	}
	for _, tc := range tests {
		got := isBreakingLine(tc.line)
		if got != tc.want {
			t.Errorf("isBreakingLine(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestExtractBreakingChanges(t *testing.T) {
	a := NewGitHubAnalyzer("")

	tests := []struct {
		name    string
		body    string
		pkgName string
		version string
		want    int
	}{
		{
			name:    "empty body",
			body:    "",
			pkgName: "testpkg",
			version: "2.0.0",
			want:    0,
		},
		{
			name: "single breaking change",
			body: `## What's Changed
* BREAKING: removed old API
* Fixed minor bug`,
			pkgName: "testpkg",
			version: "2.0.0",
			want:    1,
		},
		{
			name: "multiple breaking changes",
			body: `# Release 2.0.0
- BREAKING CHANGE: API v1 deprecated
- MAJOR CHANGE: removed old auth
- Fixed a bug`,
			pkgName: "testpkg",
			version: "2.0.0",
			want:    2,
		},
		{
			name: "no breaking changes",
			body: `# Release 1.1.0
* New feature
* Bug fix`,
			pkgName: "testpkg",
			version: "1.1.0",
			want:    0,
		},
		{
			name:    "body with only whitespace lines",
			body:    "\n\n  \n",
			pkgName: "testpkg",
			version: "1.0.0",
			want:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changes := a.extractBreakingChanges(tc.body, tc.pkgName, tc.version, "https://example.com")
			if len(changes) != tc.want {
				t.Errorf("expected %d breaking changes, got %d: %+v", tc.want, len(changes), changes)
			}
		})
	}
}

func TestExtractBreakingChangesMultiLine(t *testing.T) {
	a := NewGitHubAnalyzer("")
	body := `## Changes
* BREAKING: This is a breaking change
  that spans two lines
* Fixed a bug`

	changes := a.extractBreakingChanges(body, "testpkg", "2.0.0", "")
	if len(changes) != 1 {
		t.Fatalf("expected 1 breaking change, got %d", len(changes))
	}
	expected := "BREAKING: This is a breaking change that spans two lines"
	if changes[0].Description != expected {
		t.Errorf("expected %q, got %q", expected, changes[0].Description)
	}
}

func TestInferRepoFromPackage(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"github.com/user/repo", "user/repo"},
		{"github.com/user/repo/subpkg", "user/repo"},
		{"github.com/user/repo/v2", "user/repo"},
		{"example.com/pkg", ""},
		{"stdlib", ""},
		{"", ""},
	}
	for _, tc := range tests {
		got := inferRepoFromPackage(tc.name)
		if got != tc.want {
			t.Errorf("inferRepoFromPackage(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestFallbackChangelog(t *testing.T) {
	a := NewGitHubAnalyzer("")
	entry := a.fallbackChangelog(packageInfo("testpkg"), "1.0.0", "2.0.0")
	if entry == nil {
		t.Fatal("fallbackChangelog returned nil")
	}
	if entry.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", entry.Version)
	}
	if entry.BreakingChanges != nil {
		t.Errorf("expected nil breaking changes, got %+v", entry.BreakingChanges)
	}
}

func packageInfo(name string) domain.PackageInfo {
	return domain.PackageInfo{Name: name}
}
