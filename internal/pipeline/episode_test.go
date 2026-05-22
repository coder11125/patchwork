package pipeline

import (
	"context"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

type mockStore struct {
	recorded []*domain.Episode
}

func (m *mockStore) FindMatching(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
	return nil, nil
}
func (m *mockStore) Save(ctx context.Context, recipe *domain.Recipe) error       { return nil }
func (m *mockStore) Load(ctx context.Context, id string) (*domain.Recipe, error) { return nil, nil }
func (m *mockStore) List(ctx context.Context) ([]*domain.Recipe, error)          { return nil, nil }
func (m *mockStore) RecordEpisode(ctx context.Context, episode *domain.Episode) error {
	m.recorded = append(m.recorded, episode)
	return nil
}
func (m *mockStore) ListEpisodes(ctx context.Context) ([]*domain.Episode, error) { return nil, nil }
func (m *mockStore) UpdateRecipeStats(ctx context.Context, recipeID string, success bool) error {
	return nil
}

func TestStartEpisode(t *testing.T) {
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)
	if ep == nil {
		t.Fatal("StartEpisode returned nil")
	}
	if ep.PackageName != "testpkg" {
		t.Errorf("expected PackageName 'testpkg', got %q", ep.PackageName)
	}
	if ep.FromVersion != "1.0.0" {
		t.Errorf("expected FromVersion '1.0.0', got %q", ep.FromVersion)
	}
	if ep.ToVersion != "2.0.0" {
		t.Errorf("expected ToVersion '2.0.0', got %q", ep.ToVersion)
	}
	if ep.Ecosystem != domain.EcosystemGo {
		t.Errorf("expected Ecosystem 'go', got %q", ep.Ecosystem)
	}
	if ep.TestResult != domain.TestSkipped {
		t.Errorf("expected initial TestResult 'skipped', got %q", ep.TestResult)
	}
	if ep.Timestamp == "" {
		t.Error("expected non-empty Timestamp")
	}
	if ep.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestAddEpisodeStep(t *testing.T) {
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)

	AddEpisodeStep(ep, "apply", "running", "applying codemod")
	if len(ep.StepsExecuted) != 1 {
		t.Fatalf("expected 1 step, got %d", len(ep.StepsExecuted))
	}
	if ep.StepsExecuted[0].Type != "apply" {
		t.Errorf("expected step type 'apply', got %q", ep.StepsExecuted[0].Type)
	}
	if ep.StepsExecuted[0].Status != "running" {
		t.Errorf("expected step status 'running', got %q", ep.StepsExecuted[0].Status)
	}
	if ep.StepsExecuted[0].Order != 1 {
		t.Errorf("expected step order 1, got %d", ep.StepsExecuted[0].Order)
	}

	AddEpisodeStep(ep, "test", "passed", "all tests passed")
	if len(ep.StepsExecuted) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(ep.StepsExecuted))
	}
	if ep.StepsExecuted[1].Order != 2 {
		t.Errorf("expected step order 2, got %d", ep.StepsExecuted[1].Order)
	}
}

func TestFinalizeEpisodeSuccess(t *testing.T) {
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)
	FinalizeEpisode(ep, true, "")
	if !ep.Success {
		t.Error("expected Success to be true")
	}
	if ep.FailureReason != "" {
		t.Errorf("expected empty FailureReason, got %q", ep.FailureReason)
	}
	if ep.Duration == "" {
		t.Error("expected non-empty Duration")
	}
}

func TestFinalizeEpisodeFailure(t *testing.T) {
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)
	FinalizeEpisode(ep, false, "tests failed")
	if ep.Success {
		t.Error("expected Success to be false")
	}
	if ep.FailureReason != "tests failed" {
		t.Errorf("expected FailureReason 'tests failed', got %q", ep.FailureReason)
	}
	if ep.Duration == "" {
		t.Error("expected non-empty Duration")
	}
}

func TestRecordEpisode(t *testing.T) {
	store := &mockStore{}
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)
	ep.RecipeUsed = "recipe-1"
	FinalizeEpisode(ep, true, "")

	if err := RecordEpisode(context.Background(), store, ep); err != nil {
		t.Fatalf("RecordEpisode unexpected error: %v", err)
	}

	if len(store.recorded) != 1 {
		t.Fatalf("expected 1 recorded episode, got %d", len(store.recorded))
	}
}

func TestRecordEpisodeLearnsRecipe(t *testing.T) {
	store := &mockStore{}
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)
	ep.RecipeUsed = "recipe-1"
	FinalizeEpisode(ep, true, "")

	if err := RecordEpisode(context.Background(), store, ep); err != nil {
		t.Fatalf("RecordEpisode unexpected error: %v", err)
	}

	if ep.LearnedRecipe == "" {
		t.Error("expected LearnedRecipe to be set on successful episode with recipe")
	}
}

func TestRecordEpisodeNoRecipeDoesNotLearn(t *testing.T) {
	store := &mockStore{}
	ep := StartEpisode("testpkg", "1.0.0", "2.0.0", domain.EcosystemGo)
	FinalizeEpisode(ep, true, "")

	if err := RecordEpisode(context.Background(), store, ep); err != nil {
		t.Fatalf("RecordEpisode unexpected error: %v", err)
	}

	if ep.LearnedRecipe != "" {
		t.Errorf("expected no LearnedRecipe when no recipe used, got %q", ep.LearnedRecipe)
	}
}

func TestMustParseTime(t *testing.T) {
	parsed := mustParseTime("2024-01-15T10:30:00Z")
	if parsed.IsZero() {
		t.Error("expected a valid time")
	}
}

func TestMustParseTimeInvalid(t *testing.T) {
	parsed := mustParseTime("invalid")
	if parsed.IsZero() {
		t.Error("expected fallback to current time, not zero")
	}
}
