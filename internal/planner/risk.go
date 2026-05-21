package planner

import (
	"fmt"

	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/coder11125/patchwork/pkg/semver"
)

func AssessRisk(current, latest string, breakingChanges []domain.BreakingChange) domain.RiskLevel {
	if current == "" || latest == "" {
		return domain.RiskMedium
	}

	curr, err := semver.Parse(current)
	if err != nil {
		return domain.RiskMedium
	}

	lat, err := semver.Parse(latest)
	if err != nil {
		return domain.RiskMedium
	}

	hasBreaking := len(breakingChanges) > 0

	currMajor := curr.Major()
	latMajor := lat.Major()

	if latMajor > currMajor {
		if hasBreaking {
			return domain.RiskCritical
		}
		return domain.RiskHigh
	}

	currMinor := curr.Minor()
	latMinor := lat.Minor()

	if latMinor > currMinor {
		if hasBreaking {
			return domain.RiskHigh
		}
		return domain.RiskMedium
	}

	return domain.RiskLow
}

func VersionDistance(current, latest string) (int, error) {
	if current == "" || latest == "" {
		return 0, fmt.Errorf("version distance: empty version string")
	}

	curr, err := semver.Parse(current)
	if err != nil {
		return 0, fmt.Errorf("version distance parse current %q: %w", current, err)
	}

	lat, err := semver.Parse(latest)
	if err != nil {
		return 0, fmt.Errorf("version distance parse latest %q: %w", latest, err)
	}

	distance := 0

	if lat.Major() != curr.Major() {
		distance += 100
	}

	if lat.Minor() != curr.Minor() {
		distance += 10
	}

	if lat.Patch() != curr.Patch() {
		distance += 1
	}

	return distance, nil
}

func riskRank(r domain.RiskLevel) int {
	switch r {
	case domain.RiskLow:
		return 0
	case domain.RiskMedium:
		return 1
	case domain.RiskHigh:
		return 2
	case domain.RiskCritical:
		return 3
	default:
		return 1
	}
}
