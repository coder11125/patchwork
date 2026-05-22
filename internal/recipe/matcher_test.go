package recipe

import (
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

func TestScoreRecipe(t *testing.T) {
	tests := []struct {
		name          string
		recipe        *domain.Recipe
		targetVersion string
		wantMinScore  int
	}{
		{
			name: "exact version match",
			recipe: &domain.Recipe{
				PackageName: "testpkg",
				Ecosystem:   domain.EcosystemGo,
				ToVersion:   "2.0.0",
				FromVersion: "1.0.0",
			},
			targetVersion: "2.0.0",
			wantMinScore:  100,
		},
		{
			name: "version range match",
			recipe: &domain.Recipe{
				PackageName: "testpkg",
				Ecosystem:   domain.EcosystemGo,
				ToVersion:   "3.0.0",
				FromVersion: "1.0.0",
			},
			targetVersion: "2.0.0",
			wantMinScore:  25,
		},
		{
			name: "package name match only",
			recipe: &domain.Recipe{
				PackageName: "testpkg",
			},
			targetVersion: "2.0.0",
			wantMinScore:  10,
		},
		{
			name:          "empty recipe",
			recipe:        &domain.Recipe{},
			targetVersion: "2.0.0",
			wantMinScore:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := scoreRecipe(tc.recipe, tc.targetVersion)
			if score < tc.wantMinScore {
				t.Errorf("scoreRecipe = %d, want at least %d", score, tc.wantMinScore)
			}
		})
	}
}

func TestMatchRecipe(t *testing.T) {
	recipes := []*domain.Recipe{
		{
			ID:          "recipe-1",
			PackageName: "testpkg",
			Ecosystem:   domain.EcosystemGo,
			ToVersion:   "2.0.0",
			FromVersion: "1.0.0",
			SuccessRate: 0.9,
			TimesUsed:   10,
		},
		{
			ID:          "recipe-2",
			PackageName: "testpkg",
			Ecosystem:   domain.EcosystemGo,
			ToVersion:   "2.0.0",
			FromVersion: "1.0.0",
			SuccessRate: 1.0,
			TimesUsed:   5,
		},
	}

	tests := []struct {
		name          string
		pkg           string
		targetVersion string
		recipes       []*domain.Recipe
		wantID        string
	}{
		{
			name:          "best exact match",
			pkg:           "testpkg",
			targetVersion: "2.0.0",
			recipes:       recipes,
			wantID:        "recipe-2",
		},
		{
			name:          "no matching package",
			pkg:           "otherpkg",
			targetVersion: "2.0.0",
			recipes:       recipes,
			wantID:        "",
		},
		{
			name:          "empty recipes",
			pkg:           "testpkg",
			targetVersion: "2.0.0",
			recipes:       []*domain.Recipe{},
			wantID:        "",
		},
		{
			name:          "nil recipes",
			pkg:           "testpkg",
			targetVersion: "2.0.0",
			recipes:       nil,
			wantID:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := MatchRecipe(tc.pkg, tc.targetVersion, tc.recipes)
			if tc.wantID == "" {
				if matched != nil {
					t.Errorf("expected nil, got recipe %s", matched.ID)
				}
				return
			}
			if matched == nil {
				t.Fatal("expected a matched recipe, got nil")
			}
			if matched.ID != tc.wantID {
				t.Errorf("expected recipe %s, got %s", tc.wantID, matched.ID)
			}
		})
	}
}

func TestMatchRecipeBestBySuccessRate(t *testing.T) {
	recipes := []*domain.Recipe{
		{
			ID:          "recipe-a",
			PackageName: "testpkg",
			ToVersion:   "2.0.0",
			SuccessRate: 0.5,
			TimesUsed:   2,
		},
		{
			ID:          "recipe-b",
			PackageName: "testpkg",
			ToVersion:   "2.0.0",
			SuccessRate: 1.0,
			TimesUsed:   1,
		},
	}

	matched := MatchRecipe("testpkg", "2.0.0", recipes)
	if matched == nil {
		t.Fatal("expected a matched recipe, got nil")
	}
	if matched.ID != "recipe-b" {
		t.Errorf("expected recipe-b (higher success rate), got %s", matched.ID)
	}
}
