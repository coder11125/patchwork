package recipe

import (
	"context"

	"github.com/coder11125/patchwork/pkg/domain"
)

type RecipeStore interface {
	FindMatching(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error)
	Save(ctx context.Context, recipe *domain.Recipe) error
	Load(ctx context.Context, id string) (*domain.Recipe, error)
	List(ctx context.Context) ([]*domain.Recipe, error)
	RecordEpisode(ctx context.Context, episode *domain.Episode) error
	ListEpisodes(ctx context.Context) ([]*domain.Episode, error)
	UpdateRecipeStats(ctx context.Context, recipeID string, success bool) error
}
