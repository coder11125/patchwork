package detector

import (
	"context"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

type mockDetector struct {
	ecosystem domain.Ecosystem
	detectFn  func(ctx context.Context, dir string) (*domain.DetectResult, error)
	canHandle func(filePath string) bool
}

func (m *mockDetector) Ecosystem() domain.Ecosystem {
	return m.ecosystem
}

func (m *mockDetector) Detect(ctx context.Context, dir string) (*domain.DetectResult, error) {
	return m.detectFn(ctx, dir)
}

func (m *mockDetector) CanHandle(filePath string) bool {
	return m.canHandle(filePath)
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegisterAndForEcosystem(t *testing.T) {
	r := NewRegistry()
	d := &mockDetector{
		ecosystem: domain.EcosystemGo,
		detectFn: func(ctx context.Context, dir string) (*domain.DetectResult, error) {
			return &domain.DetectResult{Ecosystem: domain.EcosystemGo}, nil
		},
		canHandle: func(filePath string) bool {
			return filePath == "go.mod"
		},
	}
	r.Register(d)

	got, err := r.ForEcosystem(domain.EcosystemGo)
	if err != nil {
		t.Fatalf("ForEcosystem(Go) unexpected error: %v", err)
	}
	if got != d {
		t.Error("ForEcosystem returned wrong detector")
	}
}

func TestForEcosystemNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.ForEcosystem(domain.EcosystemNPM)
	if err == nil {
		t.Error("ForEcosystem should return error for unregistered ecosystem")
	}
}

func TestDetectAll(t *testing.T) {
	r := NewRegistry()

	r.Register(&mockDetector{
		ecosystem: domain.EcosystemGo,
		detectFn: func(ctx context.Context, dir string) (*domain.DetectResult, error) {
			return &domain.DetectResult{
				Ecosystem: domain.EcosystemGo,
				Packages:  []domain.PackageInfo{{Name: "pkg1"}},
			}, nil
		},
		canHandle: func(filePath string) bool { return true },
	})

	r.Register(&mockDetector{
		ecosystem: domain.EcosystemNPM,
		detectFn: func(ctx context.Context, dir string) (*domain.DetectResult, error) {
			return &domain.DetectResult{
				Ecosystem: domain.EcosystemNPM,
				Packages:  []domain.PackageInfo{{Name: "pkg2"}},
			}, nil
		},
		canHandle: func(filePath string) bool { return true },
	})

	results, err := r.DetectAll(context.Background(), "/test/dir")
	if err != nil {
		t.Fatalf("DetectAll unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestDetectAllWithError(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockDetector{
		ecosystem: domain.EcosystemGo,
		detectFn: func(ctx context.Context, dir string) (*domain.DetectResult, error) {
			return nil, assertAnError("detect failed")
		},
		canHandle: func(filePath string) bool { return true },
	})

	_, err := r.DetectAll(context.Background(), "/test/dir")
	if err == nil {
		t.Error("DetectAll should return error when a detector fails")
	}
}

func TestDetectAllNilResult(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockDetector{
		ecosystem: domain.EcosystemGo,
		detectFn: func(ctx context.Context, dir string) (*domain.DetectResult, error) {
			return nil, nil
		},
		canHandle: func(filePath string) bool { return true },
	})

	results, err := r.DetectAll(context.Background(), "/test/dir")
	if err != nil {
		t.Fatalf("DetectAll unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results when detector returns nil, got %d", len(results))
	}
}

func TestDetectAllEmptyRegistry(t *testing.T) {
	r := NewRegistry()
	results, err := r.DetectAll(context.Background(), "/test/dir")
	if err != nil {
		t.Fatalf("DetectAll unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results from empty registry, got %d", len(results))
	}
}

type errorString string

func (e errorString) Error() string { return string(e) }

func assertAnError(s string) error {
	return errorString(s)
}
