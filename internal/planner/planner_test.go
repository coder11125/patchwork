package planner

import (
	"context"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

type mockRecipeStore struct {
	findMatchingFn func(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error)
}

func (m *mockRecipeStore) FindMatching(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
	return m.findMatchingFn(ctx, ecosystem, packageName, fromVersion, toVersion)
}

func (m *mockRecipeStore) Save(ctx context.Context, recipe *domain.Recipe) error              { return nil }
func (m *mockRecipeStore) Load(ctx context.Context, id string) (*domain.Recipe, error)         { return nil, nil }
func (m *mockRecipeStore) List(ctx context.Context) ([]*domain.Recipe, error)                   { return nil, nil }
func (m *mockRecipeStore) RecordEpisode(ctx context.Context, episode *domain.Episode) error    { return nil }
func (m *mockRecipeStore) ListEpisodes(ctx context.Context) ([]*domain.Episode, error)          { return nil, nil }
func (m *mockRecipeStore) UpdateRecipeStats(ctx context.Context, recipeID string, success bool) error { return nil }

func TestPlannerPlan(t *testing.T) {
	store := &mockRecipeStore{
		findMatchingFn: func(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
			return []*domain.Recipe{
				{ID: "recipe-1", Name: "upgrade-pkg1"},
			}, nil
		},
	}
	p := New(store)

	detections := []*domain.DetectResult{
		{
			Ecosystem: domain.EcosystemGo,
			Packages: []domain.PackageInfo{
				{Name: "pkg1", Current: "1.0.0", Latest: "2.0.0"},
			},
			Upgrades: []domain.Upgrade{
				{Name: "pkg1", Current: "1.0.0", Target: "2.0.0", RiskLevel: domain.RiskHigh, Ecosystem: domain.EcosystemGo},
			},
		},
	}

	plan, err := p.Plan(context.Background(), detections, nil)
	if err != nil {
		t.Fatalf("Plan unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("Plan returned nil")
	}
	if len(plan.Upgrades) != 1 {
		t.Errorf("expected 1 upgrade, got %d", len(plan.Upgrades))
	}
	if plan.Upgrades[0].Upgrade.Name != "pkg1" {
		t.Errorf("expected upgrade name 'pkg1', got %q", plan.Upgrades[0].Upgrade.Name)
	}
	if plan.Upgrades[0].Recipe == nil {
		t.Error("expected recipe to be matched")
	}
	if len(plan.RecipesMatched) != 1 {
		t.Errorf("expected 1 recipe matched, got %d", len(plan.RecipesMatched))
	}
}

func TestPlannerPlanNoMatches(t *testing.T) {
	store := &mockRecipeStore{
		findMatchingFn: func(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
			return nil, nil
		},
	}
	p := New(store)

	detections := []*domain.DetectResult{
		{
			Ecosystem: domain.EcosystemGo,
			Packages: []domain.PackageInfo{
				{Name: "pkg1", Current: "1.0.0", Latest: "2.0.0"},
			},
			Upgrades: []domain.Upgrade{
				{Name: "pkg1", Current: "1.0.0", Target: "2.0.0", RiskLevel: domain.RiskLow, Ecosystem: domain.EcosystemGo},
			},
		},
	}

	plan, err := p.Plan(context.Background(), detections, nil)
	if err != nil {
		t.Fatalf("Plan unexpected error: %v", err)
	}
	if len(plan.Upgrades) != 1 {
		t.Errorf("expected 1 upgrade, got %d", len(plan.Upgrades))
	}
	if plan.Upgrades[0].Recipe != nil {
		t.Error("expected no recipe match")
	}
}

func TestPlannerPlanFindMatchingError(t *testing.T) {
	store := &mockRecipeStore{
		findMatchingFn: func(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
			return nil, assertError("find error")
		},
	}
	p := New(store)

	detections := []*domain.DetectResult{
		{
			Ecosystem: domain.EcosystemGo,
			Packages: []domain.PackageInfo{
				{Name: "pkg1", Current: "1.0.0", Latest: "2.0.0"},
			},
			Upgrades: []domain.Upgrade{
				{Name: "pkg1", Current: "1.0.0", Target: "2.0.0", RiskLevel: domain.RiskLow, Ecosystem: domain.EcosystemGo},
			},
		},
	}

	plan, err := p.Plan(context.Background(), detections, nil)
	if err != nil {
		t.Fatalf("Plan unexpected error: %v", err)
	}
	if len(plan.Blockers) != 1 {
		t.Errorf("expected 1 blocker, got %d", len(plan.Blockers))
	}
}

func TestPlannerPlanTotalRisk(t *testing.T) {
	store := &mockRecipeStore{
		findMatchingFn: func(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
			return nil, nil
		},
	}
	p := New(store)

	tests := []struct {
		name      string
		risks     []domain.RiskLevel
		wantRisk  domain.RiskLevel
	}{
		{"all low", []domain.RiskLevel{domain.RiskLow, domain.RiskLow}, domain.RiskLow},
		{"low and medium", []domain.RiskLevel{domain.RiskLow, domain.RiskMedium}, domain.RiskMedium},
		{"medium and high", []domain.RiskLevel{domain.RiskMedium, domain.RiskHigh}, domain.RiskHigh},
		{"high and critical", []domain.RiskLevel{domain.RiskHigh, domain.RiskCritical}, domain.RiskCritical},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var upgrades []domain.Upgrade
			for _, risk := range tc.risks {
				upgrades = append(upgrades, domain.Upgrade{
					Name:       "pkg",
					Current:    "1.0.0",
					Target:     "2.0.0",
					RiskLevel:  risk,
					Ecosystem: domain.EcosystemGo,
				})
			}

			detections := []*domain.DetectResult{
				{
					Ecosystem: domain.EcosystemGo,
					Upgrades:  upgrades,
				},
			}

			plan, err := p.Plan(context.Background(), detections, nil)
			if err != nil {
				t.Fatalf("Plan unexpected error: %v", err)
			}
			if plan.TotalRisk != tc.wantRisk {
				t.Errorf("expected total risk %q, got %q", tc.wantRisk, plan.TotalRisk)
			}
		})
	}
}

type errorStr string

func (e errorStr) Error() string { return string(e) }

func assertError(s string) error {
	return errorStr(s)
}
