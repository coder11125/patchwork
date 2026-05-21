package testrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type TestResult struct {
	Passed      bool          `json:"passed"`
	Output      string        `json:"output"`
	Duration    time.Duration `json:"duration"`
	FailedTests []string      `json:"failed_tests,omitempty"`
}

type TestRunner interface {
	Ecosystem() domain.Ecosystem
	Run(ctx context.Context, dir string) (*TestResult, error)
	CanHandle(dir string) bool
}

type Registry struct {
	runners map[domain.Ecosystem]TestRunner
}

func NewRegistry() *Registry {
	return &Registry{
		runners: make(map[domain.Ecosystem]TestRunner),
	}
}

func (r *Registry) Register(runner TestRunner) {
	r.runners[runner.Ecosystem()] = runner
}

func (r *Registry) ForEcosystem(e domain.Ecosystem) (TestRunner, error) {
	runner, ok := r.runners[e]
	if !ok {
		return nil, fmt.Errorf("no test runner registered for ecosystem: %s", e)
	}
	return runner, nil
}

func (r *Registry) RunTests(ctx context.Context, ecosystem domain.Ecosystem, dir string) (*TestResult, error) {
	runner, err := r.ForEcosystem(ecosystem)
	if err != nil {
		return nil, err
	}
	return runner.Run(ctx, dir)
}
