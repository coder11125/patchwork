package analyzer

import (
	"context"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

type mockAnalyzer struct {
	ecosystem domain.Ecosystem
	analyzeFn func(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error)
}

func (m *mockAnalyzer) Ecosystem() domain.Ecosystem {
	return m.ecosystem
}

func (m *mockAnalyzer) Analyze(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error) {
	return m.analyzeFn(ctx, pkg, currentVersion, latestVersion)
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if len(r.analyzers) != 0 {
		t.Errorf("expected empty registry, got %d analyzers", len(r.analyzers))
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry()
	m := &mockAnalyzer{
		ecosystem: domain.EcosystemGo,
		analyzeFn: func(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error) {
			return &domain.ChangelogEntry{Version: latestVersion}, nil
		},
	}
	r.Register(m)

	a, err := r.ForEcosystem(domain.EcosystemGo)
	if err != nil {
		t.Fatalf("ForEcosystem(Go) unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("ForEcosystem(Go) returned nil")
	}
}

func TestForEcosystemNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.ForEcosystem(domain.EcosystemNPM)
	if err == nil {
		t.Error("ForEcosystem should return error for unregistered ecosystem")
	}
}

func TestAnalyzeAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAnalyzer{
		ecosystem: domain.EcosystemGo,
		analyzeFn: func(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error) {
			return &domain.ChangelogEntry{
				Version:    latestVersion,
				RawContent: "test changelog",
			}, nil
		},
	})

	results := []*domain.DetectResult{
		{
			Ecosystem: domain.EcosystemGo,
			Packages: []domain.PackageInfo{
				{Name: "testpkg", Current: "1.0.0", Latest: "2.0.0", IsOutdated: true},
				{Name: "other", Current: "1.0.0", Latest: "1.0.0", IsOutdated: false},
			},
		},
	}

	entries, err := r.AnalyzeAll(context.Background(), results)
	if err != nil {
		t.Fatalf("AnalyzeAll unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 changelog entry (only outdated), got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", entries[0].Version)
	}
}

func TestAnalyzeAllNoOutdated(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAnalyzer{
		ecosystem: domain.EcosystemGo,
		analyzeFn: func(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error) {
			return &domain.ChangelogEntry{Version: latestVersion}, nil
		},
	})

	results := []*domain.DetectResult{
		{
			Ecosystem: domain.EcosystemGo,
			Packages: []domain.PackageInfo{
				{Name: "testpkg", Current: "1.0.0", Latest: "1.0.0", IsOutdated: false},
			},
		},
	}

	entries, err := r.AnalyzeAll(context.Background(), results)
	if err != nil {
		t.Fatalf("AnalyzeAll unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries when nothing is outdated, got %d", len(entries))
	}
}

func TestAnalyzeAllNoAnalyzerForEcosystem(t *testing.T) {
	r := NewRegistry()
	results := []*domain.DetectResult{
		{
			Ecosystem: domain.EcosystemGo,
			Packages: []domain.PackageInfo{
				{Name: "testpkg", Current: "1.0.0", Latest: "2.0.0", IsOutdated: true},
			},
		},
	}

	entries, err := r.AnalyzeAll(context.Background(), results)
	if err != nil {
		t.Fatalf("AnalyzeAll unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries when no analyzer registered, got %d", len(entries))
	}
}

func TestSemverAnalyzerAnalyzeVersion(t *testing.T) {
	sa := NewSemverAnalyzer()

	tests := []struct {
		name           string
		current        string
		latest         string
		pkgName        string
		wantChanges    int
		wantRisk       domain.RiskLevel
		wantIsBreaking bool
	}{
		{
			name:           "major bump",
			current:        "1.0.0",
			latest:         "2.0.0",
			pkgName:        "testpkg",
			wantChanges:    1,
			wantRisk:       domain.RiskCritical,
			wantIsBreaking: true,
		},
		{
			name:           "minor bump",
			current:        "1.0.0",
			latest:         "1.1.0",
			pkgName:        "testpkg",
			wantChanges:    1,
			wantRisk:       domain.RiskMedium,
			wantIsBreaking: false,
		},
		{
			name:           "patch bump",
			current:        "1.0.0",
			latest:         "1.0.1",
			pkgName:        "testpkg",
			wantChanges:    0,
			wantRisk:       domain.RiskLow,
			wantIsBreaking: false,
		},
		{
			name:           "same version",
			current:        "1.0.0",
			latest:         "1.0.0",
			pkgName:        "testpkg",
			wantChanges:    0,
			wantRisk:       domain.RiskLow,
			wantIsBreaking: false,
		},
		{
			name:           "version regression",
			current:        "2.0.0",
			latest:         "1.0.0",
			pkgName:        "testpkg",
			wantChanges:    1,
			wantRisk:       domain.RiskHigh,
			wantIsBreaking: false,
		},
		{
			name:           "v prefix",
			current:        "v1.0.0",
			latest:         "v2.0.0",
			pkgName:        "testpkg",
			wantChanges:    1,
			wantRisk:       domain.RiskCritical,
			wantIsBreaking: true,
		},
		{
			name:           "invalid current version",
			current:        "invalid",
			latest:         "1.0.0",
			pkgName:        "testpkg",
			wantChanges:    0,
			wantRisk:       domain.RiskMedium,
			wantIsBreaking: false,
		},
		{
			name:           "invalid latest version",
			current:        "1.0.0",
			latest:         "invalid",
			pkgName:        "testpkg",
			wantChanges:    0,
			wantRisk:       domain.RiskMedium,
			wantIsBreaking: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changes, risk, isBreaking := sa.AnalyzeVersion(tc.current, tc.latest, tc.pkgName)
			if len(changes) != tc.wantChanges {
				t.Errorf("expected %d breaking changes, got %d", tc.wantChanges, len(changes))
			}
			if risk != tc.wantRisk {
				t.Errorf("expected risk %q, got %q", tc.wantRisk, risk)
			}
			if isBreaking != tc.wantIsBreaking {
				t.Errorf("expected isBreaking=%v, got %v", tc.wantIsBreaking, isBreaking)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
		{"V1.2.3", "1.2.3"},
		{"", "0.0.0"},
		{"v", ""},
	}
	for _, tc := range tests {
		got := normalizeVersion(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeVersion(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
