package recipe

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

func TestNewDiskStore(t *testing.T) {
	recipeDir := filepath.Join(t.TempDir(), "recipes")
	episodeDir := filepath.Join(t.TempDir(), "episodes")

	store, err := NewDiskStore(recipeDir, episodeDir, slog.Default())
	if err != nil {
		t.Fatalf("NewDiskStore unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("NewDiskStore returned nil")
	}

	if _, err := os.Stat(recipeDir); os.IsNotExist(err) {
		t.Error("expected recipe dir to be created")
	}
	if _, err := os.Stat(episodeDir); os.IsNotExist(err) {
		t.Error("expected episode dir to be created")
	}
}

func TestSaveAndLoad(t *testing.T) {
	store := newTestStore(t)

	recipe := &domain.Recipe{
		ID:          "test-recipe",
		Name:        "test",
		Ecosystem:   domain.EcosystemGo,
		PackageName: "testpkg",
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
		Steps:       []domain.RecipeStep{{Order: 1, Type: "replace", Pattern: "foo", Replacement: "bar"}},
	}

	if err := store.Save(context.Background(), recipe); err != nil {
		t.Fatalf("Save unexpected error: %v", err)
	}

	loaded, err := store.Load(context.Background(), "test-recipe")
	if err != nil {
		t.Fatalf("Load unexpected error: %v", err)
	}
	if loaded.Name != "test" {
		t.Errorf("expected name 'test', got %q", loaded.Name)
	}
	if loaded.Ecosystem != domain.EcosystemGo {
		t.Errorf("expected ecosystem 'go', got %q", loaded.Ecosystem)
	}
}

func TestLoadNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Load(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent recipe")
	}
}

func TestList(t *testing.T) {
	store := newTestStore(t)

	r1 := &domain.Recipe{ID: "r1", Name: "recipe1", Ecosystem: domain.EcosystemGo, PackageName: "pkg1"}
	r2 := &domain.Recipe{ID: "r2", Name: "recipe2", Ecosystem: domain.EcosystemNPM, PackageName: "pkg2"}

	store.Save(context.Background(), r1)
	store.Save(context.Background(), r2)

	recipes, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List unexpected error: %v", err)
	}
	if len(recipes) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(recipes))
	}
}

func TestListEmpty(t *testing.T) {
	store := newTestStore(t)

	recipes, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List unexpected error: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("expected 0 recipes, got %d", len(recipes))
	}
}

func TestFindMatching(t *testing.T) {
	store := newTestStore(t)

	store.Save(context.Background(), &domain.Recipe{
		ID:          "match",
		Ecosystem:   domain.EcosystemGo,
		PackageName: "testpkg",
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
	})

	tests := []struct {
		name      string
		ecosystem domain.Ecosystem
		pkg       string
		from      string
		to        string
		wantMatch bool
	}{
		{"exact match", domain.EcosystemGo, "testpkg", "1.0.0", "2.0.0", true},
		{"wrong ecosystem", domain.EcosystemNPM, "testpkg", "1.0.0", "2.0.0", false},
		{"wrong package", domain.EcosystemGo, "otherpkg", "1.0.0", "2.0.0", false},
		{"version in range", domain.EcosystemGo, "testpkg", "1.5.0", "2.0.0", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := store.FindMatching(context.Background(), tc.ecosystem, tc.pkg, tc.from, tc.to)
			if err != nil {
				t.Fatalf("FindMatching unexpected error: %v", err)
			}
			if tc.wantMatch && len(matched) == 0 {
				t.Error("expected a match, got none")
			}
			if !tc.wantMatch && len(matched) > 0 {
				t.Errorf("expected no match, got %d", len(matched))
			}
		})
	}
}

func TestRecordAndListEpisodes(t *testing.T) {
	store := newTestStore(t)

	ep := &domain.Episode{
		ID:          "test-ep",
		Ecosystem:   domain.EcosystemGo,
		PackageName: "testpkg",
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
		Success:     true,
		TestResult:  domain.TestPassed,
	}

	if err := store.RecordEpisode(context.Background(), ep); err != nil {
		t.Fatalf("RecordEpisode unexpected error: %v", err)
	}

	episodes, err := store.ListEpisodes(context.Background())
	if err != nil {
		t.Fatalf("ListEpisodes unexpected error: %v", err)
	}
	if len(episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(episodes))
	}
	if episodes[0].PackageName != "testpkg" {
		t.Errorf("expected package 'testpkg', got %q", episodes[0].PackageName)
	}
	if !episodes[0].Success {
		t.Error("expected episode to be successful")
	}
}

func TestRecordEpisodeAutoID(t *testing.T) {
	store := newTestStore(t)

	ep := &domain.Episode{
		Ecosystem:   domain.EcosystemGo,
		PackageName: "testpkg",
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
	}

	if err := store.RecordEpisode(context.Background(), ep); err != nil {
		t.Fatalf("RecordEpisode unexpected error: %v", err)
	}
	if ep.ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestUpdateRecipeStats(t *testing.T) {
	store := newTestStore(t)

	recipe := &domain.Recipe{
		ID:          "stats-test",
		Name:        "stats",
		Ecosystem:   domain.EcosystemGo,
		PackageName: "testpkg",
		FromVersion: "1.0.0",
		ToVersion:   "2.0.0",
	}

	if err := store.Save(context.Background(), recipe); err != nil {
		t.Fatalf("Save unexpected error: %v", err)
	}

	if err := store.UpdateRecipeStats(context.Background(), "stats-test", true); err != nil {
		t.Fatalf("UpdateRecipeStats unexpected error: %v", err)
	}

	updated, err := store.Load(context.Background(), "stats-test")
	if err != nil {
		t.Fatalf("Load unexpected error: %v", err)
	}
	if updated.TimesUsed != 1 {
		t.Errorf("expected TimesUsed=1, got %d", updated.TimesUsed)
	}
	if updated.SuccessRate != 1.0 {
		t.Errorf("expected SuccessRate=1.0, got %f", updated.SuccessRate)
	}
}

func TestGenerateEpisodeID(t *testing.T) {
	id := generateEpisodeID()
	if id == "" {
		t.Error("expected non-empty episode ID")
	}
	if len(id) < 10 {
		t.Errorf("expected episode ID to be at least 10 chars, got %q", id)
	}
}

func newTestStore(t *testing.T) *DiskStore {
	t.Helper()
	recipeDir := filepath.Join(t.TempDir(), "recipes")
	episodeDir := filepath.Join(t.TempDir(), "episodes")
	store, err := NewDiskStore(recipeDir, episodeDir, slog.Default())
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}
	return store
}
