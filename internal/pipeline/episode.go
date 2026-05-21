package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/pkg/domain"
)

func StartEpisode(packageName, fromVersion, toVersion string, ecosystem domain.Ecosystem) *domain.Episode {
	return &domain.Episode{
		ID:            fmt.Sprintf("ep-%d", time.Now().UnixNano()),
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Ecosystem:     ecosystem,
		PackageName:   packageName,
		FromVersion:   fromVersion,
		ToVersion:     toVersion,
		TestResult:    domain.TestSkipped,
		StepsExecuted: make([]domain.EpisodeStep, 0),
	}
}

func FinalizeEpisode(ep *domain.Episode, success bool, failureReason string) {
	ep.Success = success
	ep.FailureReason = failureReason
	ep.Duration = time.Since(mustParseTime(ep.Timestamp)).String()
}

func AddEpisodeStep(ep *domain.Episode, stepType, status, detail string) {
	ep.StepsExecuted = append(ep.StepsExecuted, domain.EpisodeStep{
		Order:  len(ep.StepsExecuted) + 1,
		Type:   stepType,
		Status: status,
		Detail: detail,
	})
}

func RecordEpisode(ctx context.Context, store recipe.RecipeStore, ep *domain.Episode) error {
	if ep.RecipeUsed != "" && ep.Success {
		learned := fmt.Sprintf("learned-%s", ep.ID)
		ep.LearnedRecipe = learned
	}
	return store.RecordEpisode(ctx, ep)
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}
