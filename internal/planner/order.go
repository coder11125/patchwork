package planner

import (
	"sort"

	"github.com/coder11125/patchwork/pkg/domain"
)

func OrderUpgrades(upgrades []domain.UpgradePlan) []domain.UpgradePlan {
	if len(upgrades) <= 1 {
		return upgrades
	}

	sorted := make([]domain.UpgradePlan, len(upgrades))
	copy(sorted, upgrades)

	sort.SliceStable(sorted, func(i, j int) bool {
		ri := riskRank(sorted[i].Upgrade.RiskLevel)
		rj := riskRank(sorted[j].Upgrade.RiskLevel)

		if ri != rj {
			return ri < rj
		}

		return sorted[i].Upgrade.Name < sorted[j].Upgrade.Name
	})

	return sorted
}
