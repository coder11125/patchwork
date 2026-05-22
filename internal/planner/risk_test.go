package planner

import (
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

func TestAssessRisk(t *testing.T) {
	tests := []struct {
		name           string
		current        string
		latest         string
		breaking       []domain.BreakingChange
		want           domain.RiskLevel
	}{
		{
			name:    "major bump with breaking changes",
			current: "1.0.0",
			latest:  "2.0.0",
			breaking: []domain.BreakingChange{{Description: "breaking change"}},
			want:    domain.RiskCritical,
		},
		{
			name:    "major bump without breaking changes",
			current: "1.0.0",
			latest:  "2.0.0",
			want:    domain.RiskHigh,
		},
		{
			name:    "minor bump with breaking changes",
			current: "1.0.0",
			latest:  "1.1.0",
			breaking: []domain.BreakingChange{{Description: "breaking change"}},
			want:    domain.RiskHigh,
		},
		{
			name:    "minor bump without breaking changes",
			current: "1.0.0",
			latest:  "1.1.0",
			want:    domain.RiskMedium,
		},
		{
			name:    "patch bump",
			current: "1.0.0",
			latest:  "1.0.1",
			want:    domain.RiskLow,
		},
		{
			name:    "same version",
			current: "1.0.0",
			latest:  "1.0.0",
			want:    domain.RiskLow,
		},
		{
			name:    "empty versions",
			current: "",
			latest:  "",
			want:    domain.RiskMedium,
		},
		{
			name:    "invalid current version",
			current: "invalid",
			latest:  "1.0.0",
			want:    domain.RiskMedium,
		},
		{
			name:    "invalid latest version",
			current: "1.0.0",
			latest:  "invalid",
			want:    domain.RiskMedium,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AssessRisk(tc.current, tc.latest, tc.breaking)
			if got != tc.want {
				t.Errorf("AssessRisk(%q, %q) = %q, want %q", tc.current, tc.latest, got, tc.want)
			}
		})
	}
}

func TestVersionDistance(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
		wantErr bool
	}{
		{"same version", "1.0.0", "1.0.0", 0, false},
		{"patch bump", "1.0.0", "1.0.1", 1, false},
		{"minor bump", "1.0.0", "1.1.0", 10, false},
		{"major bump", "1.0.0", "2.0.0", 100, false},
		{"major and minor", "1.0.0", "2.1.0", 110, false},
		{"all three", "1.0.0", "2.1.1", 111, false},
		{"empty current", "", "1.0.0", 0, true},
		{"empty latest", "1.0.0", "", 0, true},
		{"invalid current", "invalid", "1.0.0", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := VersionDistance(tc.current, tc.latest)
			if tc.wantErr {
				if err == nil {
					t.Errorf("VersionDistance(%q, %q) expected error", tc.current, tc.latest)
				}
				return
			}
			if err != nil {
				t.Fatalf("VersionDistance(%q, %q) unexpected error: %v", tc.current, tc.latest, err)
			}
			if got != tc.want {
				t.Errorf("VersionDistance(%q, %q) = %d, want %d", tc.current, tc.latest, got, tc.want)
			}
		})
	}
}

func TestRiskRank(t *testing.T) {
	tests := []struct {
		risk domain.RiskLevel
		want int
	}{
		{domain.RiskLow, 0},
		{domain.RiskMedium, 1},
		{domain.RiskHigh, 2},
		{domain.RiskCritical, 3},
		{domain.RiskLevel("unknown"), 1},
	}
	for _, tc := range tests {
		got := riskRank(tc.risk)
		if got != tc.want {
			t.Errorf("riskRank(%q) = %d, want %d", tc.risk, got, tc.want)
		}
	}
}
