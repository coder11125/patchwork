package detector

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/coder11125/patchwork/pkg/domain"
)

type PackageDetector interface {
	Ecosystem() domain.Ecosystem
	Detect(ctx context.Context, dir string) (*domain.DetectResult, error)
	CanHandle(filePath string) bool
}

type DetectorRegistry struct {
	mu        sync.RWMutex
	detectors map[domain.Ecosystem]PackageDetector
}

func NewRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: make(map[domain.Ecosystem]PackageDetector),
	}
}

func (r *DetectorRegistry) Register(d PackageDetector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detectors[d.Ecosystem()] = d
}

func (r *DetectorRegistry) ForEcosystem(e domain.Ecosystem) (PackageDetector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.detectors[e]
	if !ok {
		return nil, fmt.Errorf("no detector registered for ecosystem: %s", e)
	}
	return d, nil
}

func (r *DetectorRegistry) DetectAll(ctx context.Context, dir string) ([]*domain.DetectResult, error) {
	r.mu.RLock()
	detectors := make([]PackageDetector, 0, len(r.detectors))
	for _, d := range r.detectors {
		detectors = append(detectors, d)
	}
	r.mu.RUnlock()

	var mu sync.Mutex
	var results []*domain.DetectResult

	g, ctx := errgroup.WithContext(ctx)

	for _, d := range detectors {
		det := d
		g.Go(func() error {
			res, err := det.Detect(ctx, dir)
			if err != nil {
				return err
			}
			if res != nil {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return results, err
	}

	return results, nil
}
