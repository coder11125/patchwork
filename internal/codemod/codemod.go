package codemod

import (
	"context"
	"sync"

	"github.com/coder11125/patchwork/pkg/domain"
)

type Codemod interface {
	Name() string
	Ecosystem() domain.Ecosystem
	Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error
}

type Registry struct {
	mu       sync.RWMutex
	codemods map[string]Codemod
}

func NewRegistry() *Registry {
	return &Registry{
		codemods: make(map[string]Codemod),
	}
}

func (r *Registry) Register(c Codemod) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codemods[c.Name()] = c
}

func (r *Registry) Get(name string) (Codemod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.codemods[name]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (r *Registry) Apply(ctx context.Context, workDir string, recipe *domain.Recipe) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.codemods {
		if c.Ecosystem() == recipe.Ecosystem {
			if err := c.Apply(ctx, workDir, recipe); err != nil {
				return err
			}
		}
	}
	return nil
}
