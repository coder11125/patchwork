package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/coder11125/patchwork/pkg/semver"
)

type DiskStore struct {
	recipeDir  string
	episodeDir string
	logger     *slog.Logger
}

func NewDiskStore(recipeDir, episodeDir string, logger *slog.Logger) (*DiskStore, error) {
	if err := os.MkdirAll(recipeDir, 0755); err != nil {
		return nil, fmt.Errorf("create recipe dir: %w", err)
	}
	if err := os.MkdirAll(episodeDir, 0755); err != nil {
		return nil, fmt.Errorf("create episode dir: %w", err)
	}
	return &DiskStore{
		recipeDir:  recipeDir,
		episodeDir: episodeDir,
		logger:     logger,
	}, nil
}

func (s *DiskStore) Save(ctx context.Context, recipe *domain.Recipe) error {
	if recipe.UpdatedAt == "" {
		recipe.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if recipe.CreatedAt == "" {
		recipe.CreatedAt = recipe.UpdatedAt
	}
	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal recipe %s: %w", recipe.ID, err)
	}
	path := filepath.Join(s.recipeDir, recipe.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write recipe %s: %w", recipe.ID, err)
	}
	s.logger.InfoContext(ctx, "saved recipe", "id", recipe.ID, "path", path)
	return nil
}

func (s *DiskStore) Load(ctx context.Context, id string) (*domain.Recipe, error) {
	path := filepath.Join(s.recipeDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("recipe %s not found", id)
		}
		return nil, fmt.Errorf("read recipe %s: %w", id, err)
	}
	var recipe domain.Recipe
	if err := json.Unmarshal(data, &recipe); err != nil {
		return nil, fmt.Errorf("unmarshal recipe %s: %w", id, err)
	}
	return &recipe, nil
}

func (s *DiskStore) List(ctx context.Context) ([]*domain.Recipe, error) {
	entries, err := os.ReadDir(s.recipeDir)
	if err != nil {
		return nil, fmt.Errorf("read recipe dir: %w", err)
	}
	var recipes []*domain.Recipe
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.recipeDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to read recipe file", "path", path, "error", err)
			continue
		}
		var recipe domain.Recipe
		if err := json.Unmarshal(data, &recipe); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal recipe", "path", path, "error", err)
			continue
		}
		recipes = append(recipes, &recipe)
	}
	return recipes, nil
}

func (s *DiskStore) FindMatching(ctx context.Context, ecosystem domain.Ecosystem, packageName, fromVersion, toVersion string) ([]*domain.Recipe, error) {
	all, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var matched []*domain.Recipe
	for _, r := range all {
		if r.Ecosystem != ecosystem {
			continue
		}
		if r.PackageName != packageName {
			continue
		}
		if r.FromVersion != "" && fromVersion != "" {
			ok, err := semver.Satisfies(fromVersion, ">="+r.FromVersion)
			if err != nil || !ok {
				continue
			}
		}
		if r.ToVersion != "" && toVersion != "" {
			ok, err := semver.Satisfies(toVersion, "<="+r.ToVersion)
			if err != nil || !ok {
				continue
			}
		}
		matched = append(matched, r)
	}
	return matched, nil
}

func (s *DiskStore) RecordEpisode(ctx context.Context, episode *domain.Episode) error {
	if episode.ID == "" {
		episode.ID = generateEpisodeID()
	}
	if episode.Timestamp == "" {
		episode.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(episode, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal episode %s: %w", episode.ID, err)
	}
	path := filepath.Join(s.episodeDir, episode.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write episode %s: %w", episode.ID, err)
	}
	s.logger.InfoContext(ctx, "recorded episode", "id", episode.ID, "path", path)
	return nil
}

func (s *DiskStore) ListEpisodes(ctx context.Context) ([]*domain.Episode, error) {
	entries, err := os.ReadDir(s.episodeDir)
	if err != nil {
		return nil, fmt.Errorf("read episode dir: %w", err)
	}
	var episodes []*domain.Episode
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.episodeDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to read episode file", "path", path, "error", err)
			continue
		}
		var episode domain.Episode
		if err := json.Unmarshal(data, &episode); err != nil {
			s.logger.WarnContext(ctx, "failed to unmarshal episode", "path", path, "error", err)
			continue
		}
		episodes = append(episodes, &episode)
	}
	return episodes, nil
}

func (s *DiskStore) UpdateRecipeStats(ctx context.Context, recipeID string, success bool) error {
	recipe, err := s.Load(ctx, recipeID)
	if err != nil {
		return err
	}
	recipe.TimesUsed++
	total := float64(recipe.TimesUsed)
	if success {
		recipe.SuccessRate = ((recipe.SuccessRate * (total - 1)) + 1.0) / total
	} else {
		recipe.SuccessRate = (recipe.SuccessRate * (total - 1)) / total
	}
	recipe.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Save(ctx, recipe); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "updated recipe stats", "id", recipeID, "success", success, "rate", recipe.SuccessRate, "used", recipe.TimesUsed)
	return nil
}

func generateEpisodeID() string {
	now := time.Now().UTC()
	return fmt.Sprintf("ep_%s_%09d", now.Format("20060102T150405"), now.Nanosecond())
}
