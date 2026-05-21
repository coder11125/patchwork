package analyzer

import (
	"context"
	"fmt"
	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/coder11125/patchwork/pkg/domain"
)

type Analyzer interface {
	Ecosystem() domain.Ecosystem
	Analyze(ctx context.Context, pkg domain.PackageInfo, currentVersion, latestVersion string) (*domain.ChangelogEntry, error)
}

type Registry struct {
	analyzers map[domain.Ecosystem]Analyzer
}

func NewRegistry() *Registry {
	return &Registry{
		analyzers: make(map[domain.Ecosystem]Analyzer),
	}
}

func (r *Registry) Register(a Analyzer) {
	r.analyzers[a.Ecosystem()] = a
	slog.Info("registered analyzer", "ecosystem", a.Ecosystem())
}

func (r *Registry) ForEcosystem(e domain.Ecosystem) (Analyzer, error) {
	a, ok := r.analyzers[e]
	if !ok {
		return nil, fmt.Errorf("no analyzer registered for ecosystem: %s", e)
	}
	return a, nil
}

func (r *Registry) AnalyzeAll(ctx context.Context, results []*domain.DetectResult) ([]*domain.ChangelogEntry, error) {
	type workItem struct {
		pkg            domain.PackageInfo
		currentVersion string
		latestVersion  string
		ecosystem      domain.Ecosystem
	}

	var items []workItem
	for _, res := range results {
		for _, pkg := range res.Packages {
			if pkg.IsOutdated {
				items = append(items, workItem{
					pkg:            pkg,
					currentVersion: pkg.Current,
					latestVersion:  pkg.Latest,
					ecosystem:      res.Ecosystem,
				})
			}
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	entries := make([]*domain.ChangelogEntry, len(items))

	g, ctx := errgroup.WithContext(ctx)
	for i, item := range items {
		i, item := i, item
		g.Go(func() error {
			a, err := r.ForEcosystem(item.ecosystem)
			if err != nil {
				slog.Warn("no analyzer for ecosystem, skipping",
					"ecosystem", item.ecosystem,
					"package", item.pkg.Name,
				)
				return nil
			}

			entry, err := a.Analyze(ctx, item.pkg, item.currentVersion, item.latestVersion)
			if err != nil {
				slog.Error("analysis failed",
					"package", item.pkg.Name,
					"ecosystem", item.ecosystem,
					"error", err,
				)
				return nil
			}

			entries[i] = entry
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var filtered []*domain.ChangelogEntry
	for _, e := range entries {
		if e != nil {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}
