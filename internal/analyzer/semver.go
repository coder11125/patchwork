package analyzer

import (
	"fmt"
	"log/slog"

	"github.com/Masterminds/semver/v3"
	"github.com/coder11125/patchwork/pkg/domain"
)

type SemverAnalyzer struct{}

func NewSemverAnalyzer() *SemverAnalyzer {
	return &SemverAnalyzer{}
}

func (a *SemverAnalyzer) AnalyzeVersion(currentVersion, latestVersion, packageName string) ([]domain.BreakingChange, domain.RiskLevel, bool) {
	current, err := semver.NewVersion(normalizeVersion(currentVersion))
	if err != nil {
		slog.Warn("failed to parse current version",
			"version", currentVersion,
			"package", packageName,
			"error", err,
		)
		return nil, domain.RiskMedium, false
	}

	latest, err := semver.NewVersion(normalizeVersion(latestVersion))
	if err != nil {
		slog.Warn("failed to parse latest version",
			"version", latestVersion,
			"package", packageName,
			"error", err,
		)
		return nil, domain.RiskMedium, false
	}

	var changes []domain.BreakingChange
	var riskLevel domain.RiskLevel
	isBreaking := false

	currentMajor := current.Major()
	latestMajor := latest.Major()
	currentMinor := current.Minor()
	latestMinor := latest.Minor()
	currentPatch := current.Patch()
	latestPatch := latest.Patch()

	if latestMajor > currentMajor {
		isBreaking = true
		riskLevel = domain.RiskCritical

		changes = append(changes, domain.BreakingChange{
			ID:          fmt.Sprintf("%s-%s-major-bump", packageName, latestVersion),
			PackageName: packageName,
			Version:     latestVersion,
			Description: fmt.Sprintf("Major version bump from v%d to v%d indicates breaking API changes", currentMajor, latestMajor),
			Severity:    "critical",
		})
	} else if latestMinor > currentMinor {
		riskLevel = domain.RiskMedium

		changes = append(changes, domain.BreakingChange{
			ID:          fmt.Sprintf("%s-%s-minor-bump", packageName, latestVersion),
			PackageName: packageName,
			Version:     latestVersion,
			Description: fmt.Sprintf("Minor version bump from v%d.%d to v%d.%d may include breaking changes", currentMajor, currentMinor, latestMajor, latestMinor),
			Severity:    "medium",
		})
	} else if latestPatch > currentPatch {
		riskLevel = domain.RiskLow
	} else if latest.LessThan(current) {
		riskLevel = domain.RiskHigh
		changes = append(changes, domain.BreakingChange{
			ID:          fmt.Sprintf("%s-%s-version-regression", packageName, latestVersion),
			PackageName: packageName,
			Version:     latestVersion,
			Description: fmt.Sprintf("Version regression detected: current v%s is newer than latest v%s", currentVersion, latestVersion),
			Severity:    "high",
		})
	} else {
		riskLevel = domain.RiskLow
	}

	return changes, riskLevel, isBreaking
}

func normalizeVersion(v string) string {
	if v == "" {
		return "0.0.0"
	}
	if v[0] == 'v' || v[0] == 'V' {
		return v[1:]
	}
	return v
}
