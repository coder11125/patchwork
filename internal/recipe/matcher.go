package recipe

import (
	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/coder11125/patchwork/pkg/semver"
)

type recipeScore struct {
	recipe *domain.Recipe
	score  int
}

func MatchRecipe(pkg string, targetVersion string, recipes []*domain.Recipe) *domain.Recipe {
	if len(recipes) == 0 {
		return nil
	}
	var scored []recipeScore
	for _, r := range recipes {
		if r.PackageName != pkg {
			continue
		}
		score := scoreRecipe(r, targetVersion)
		if score > 0 {
			scored = append(scored, recipeScore{recipe: r, score: score})
		}
	}
	if len(scored) == 0 {
		return nil
	}
	best := scored[0]
	for _, s := range scored[1:] {
		if s.score > best.score {
			best = s
		} else if s.score == best.score && s.recipe.SuccessRate > best.recipe.SuccessRate {
			best = s
		}
	}
	return best.recipe
}

func scoreRecipe(r *domain.Recipe, targetVersion string) int {
	score := 0
	if r.ToVersion != "" && targetVersion != "" {
		if r.ToVersion == targetVersion {
			score += 100
		} else {
			ok, err := semver.Satisfies(targetVersion, "<="+r.ToVersion)
			if err == nil && ok {
				score += 50
			}
		}
	}
	if r.FromVersion != "" && targetVersion != "" {
		ok, err := semver.Satisfies(targetVersion, ">="+r.FromVersion)
		if err == nil && ok {
			score += 25
		}
	}
	if r.PackageName != "" {
		score += 10
	}
	if r.Ecosystem != "" {
		score += 5
	}
	return score
}
