package planner

import (
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

func TestOrderUpgrades(t *testing.T) {
	upgrades := []domain.UpgradePlan{
		{Upgrade: domain.Upgrade{Name: "high-pkg", RiskLevel: domain.RiskHigh}, Order: 1},
		{Upgrade: domain.Upgrade{Name: "low-pkg", RiskLevel: domain.RiskLow}, Order: 2},
		{Upgrade: domain.Upgrade{Name: "medium-pkg", RiskLevel: domain.RiskMedium}, Order: 3},
	}

	sorted := OrderUpgrades(upgrades)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 upgrades, got %d", len(sorted))
	}

	if sorted[0].Upgrade.Name != "low-pkg" {
		t.Errorf("expected first upgrade to be 'low-pkg', got %q", sorted[0].Upgrade.Name)
	}
	if sorted[1].Upgrade.Name != "medium-pkg" {
		t.Errorf("expected second upgrade to be 'medium-pkg', got %q", sorted[1].Upgrade.Name)
	}
	if sorted[2].Upgrade.Name != "high-pkg" {
		t.Errorf("expected third upgrade to be 'high-pkg', got %q", sorted[2].Upgrade.Name)
	}
}

func TestOrderUpgradesSameRisk(t *testing.T) {
	upgrades := []domain.UpgradePlan{
		{Upgrade: domain.Upgrade{Name: "zeta", RiskLevel: domain.RiskMedium}, Order: 1},
		{Upgrade: domain.Upgrade{Name: "alpha", RiskLevel: domain.RiskMedium}, Order: 2},
	}

	sorted := OrderUpgrades(upgrades)
	if sorted[0].Upgrade.Name != "alpha" {
		t.Errorf("expected 'alpha' first when same risk, got %q", sorted[0].Upgrade.Name)
	}
	if sorted[1].Upgrade.Name != "zeta" {
		t.Errorf("expected 'zeta' second when same risk, got %q", sorted[1].Upgrade.Name)
	}
}

func TestOrderUpgradesSingle(t *testing.T) {
	upgrades := []domain.UpgradePlan{
		{Upgrade: domain.Upgrade{Name: "only-pkg", RiskLevel: domain.RiskHigh}, Order: 1},
	}

	sorted := OrderUpgrades(upgrades)
	if len(sorted) != 1 {
		t.Fatalf("expected 1 upgrade, got %d", len(sorted))
	}
	if sorted[0].Upgrade.Name != "only-pkg" {
		t.Errorf("expected 'only-pkg', got %q", sorted[0].Upgrade.Name)
	}
}

func TestOrderUpgradesEmpty(t *testing.T) {
	sorted := OrderUpgrades(nil)
	if sorted != nil {
		t.Errorf("expected nil for empty input, got %v", sorted)
	}
}

func TestOrderUpgradesDoesNotMutateOriginal(t *testing.T) {
	original := []domain.UpgradePlan{
		{Upgrade: domain.Upgrade{Name: "b-pkg", RiskLevel: domain.RiskHigh}, Order: 1},
		{Upgrade: domain.Upgrade{Name: "a-pkg", RiskLevel: domain.RiskLow}, Order: 2},
	}

	sorted := OrderUpgrades(original)

	if original[0].Upgrade.Name != "b-pkg" {
		t.Error("OrderUpgrades should not mutate the original slice")
	}
	_ = sorted
}
