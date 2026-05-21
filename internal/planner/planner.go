package planner

import (
	"context"

	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/pkg/domain"
)

type Planner struct {
	recipeStore recipe.RecipeStore
}

func New(recipeStore recipe.RecipeStore) *Planner {
	return &Planner{recipeStore: recipeStore}
}

func (p *Planner) Plan(ctx context.Context, detections []*domain.DetectResult, changelogs []*domain.ChangelogEntry) (*domain.PlanResult, error) {
	var upgrades []domain.UpgradePlan
	var blockers []string
	var recipesMatched []string

	for _, dr := range detections {
		for _, upgrade := range dr.Upgrades {
			recipes, err := p.recipeStore.FindMatching(ctx, dr.Ecosystem, upgrade.Name, upgrade.Current, upgrade.Target)
			if err != nil {
				blockers = append(blockers, err.Error())
				continue
			}

			plan := domain.UpgradePlan{
				Upgrade: upgrade,
				Order:   len(upgrades) + 1,
			}
			if len(recipes) > 0 {
				plan.Recipe = recipes[0]
				recipesMatched = append(recipesMatched, recipes[0].ID)
			}
			upgrades = append(upgrades, plan)
		}
	}

	totalRisk := domain.RiskLow
	for _, u := range upgrades {
		if u.Upgrade.RiskLevel == domain.RiskCritical {
			totalRisk = domain.RiskCritical
			break
		}
		if u.Upgrade.RiskLevel == domain.RiskHigh && totalRisk != domain.RiskCritical {
			totalRisk = domain.RiskHigh
		}
		if u.Upgrade.RiskLevel == domain.RiskMedium && totalRisk == domain.RiskLow {
			totalRisk = domain.RiskMedium
		}
	}

	return &domain.PlanResult{
		Upgrades:       upgrades,
		TotalRisk:      totalRisk,
		RecipesMatched: recipesMatched,
		Blockers:       blockers,
	}, nil
}
