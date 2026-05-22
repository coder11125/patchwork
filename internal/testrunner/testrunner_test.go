package testrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coder11125/patchwork/pkg/domain"
)

type mockRunner struct {
	ecosystem domain.Ecosystem
	runFn     func(dir string) *TestResult
	canHandle func(dir string) bool
}

func (m *mockRunner) Ecosystem() domain.Ecosystem { return m.ecosystem }

func (m *mockRunner) Run(ctx context.Context, dir string) (*TestResult, error) {
	return m.runFn(dir), nil
}

func (m *mockRunner) CanHandle(dir string) bool { return m.canHandle(dir) }

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegisterAndForEcosystem(t *testing.T) {
	r := NewRegistry()
	runner := &mockRunner{
		ecosystem: domain.EcosystemGo,
		runFn:     func(dir string) *TestResult { return &TestResult{Passed: true} },
		canHandle: func(dir string) bool { return true },
	}
	r.Register(runner)

	got, err := r.ForEcosystem(domain.EcosystemGo)
	if err != nil {
		t.Fatalf("ForEcosystem unexpected error: %v", err)
	}
	if got != runner {
		t.Error("ForEcosystem returned wrong runner")
	}
}

func TestForEcosystemNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.ForEcosystem(domain.EcosystemNPM)
	if err == nil {
		t.Error("ForEcosystem should return error for unregistered ecosystem")
	}
}

func TestRunTests(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockRunner{
		ecosystem: domain.EcosystemGo,
		runFn:     func(dir string) *TestResult { return &TestResult{Passed: true} },
		canHandle: func(dir string) bool { return true },
	})

	result, err := r.RunTests(context.Background(), domain.EcosystemGo, "/test")
	if err != nil {
		t.Fatalf("RunTests unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected tests to pass")
	}
}

func TestRunTestsNoRunner(t *testing.T) {
	r := NewRegistry()
	_, err := r.RunTests(context.Background(), domain.EcosystemGo, "/test")
	if err == nil {
		t.Error("RunTests should return error for unregistered ecosystem")
	}
}

func TestParseGoFailedTests(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{"no failures", "ok  package\nok  another", 0},
		{"single failure", "--- FAIL: TestSomething\n    test.go:42: assertion failed", 1},
		{"multiple failures", "--- FAIL: TestA\n--- FAIL: TestB\n--- FAIL: TestC", 3},
		{"empty output", "", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			failed := parseGoFailedTests(tc.output)
			if len(failed) != tc.want {
				t.Errorf("expected %d failed tests, got %d: %v", tc.want, len(failed), failed)
			}
		})
	}
}

func TestParseNPMFailedTests(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{"no failures", "PASS tests passed", 0},
		{"jest failure", "  ✕ should work correctly", 1},
		{"mocha failure", "  1) should do something", 1},
		{"generic failure", "FAIL test/foo.test.js", 1},
		{"multiple failures", "  ✕ test one\n  ✕ test two\nFAIL test/bar.test.js", 3},
		{"empty output", "", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			failed := parseNPMFailedTests(tc.output)
			if len(failed) != tc.want {
				t.Errorf("expected %d failed tests, got %d: %v", tc.want, len(failed), failed)
			}
		})
	}
}

func TestParseCargoFailedTests(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{"no failures", "test result: ok. 10 passed; 0 failed", 0},
		{"single failure", "test test_foo ... FAILED", 1},
		{"multiple failures", "test test_a ... FAILED\ntest test_b ... FAILED", 2},
		{"empty output", "", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			failed := parseCargoFailedTests(tc.output)
			if len(failed) != tc.want {
				t.Errorf("expected %d failed tests, got %d: %v", tc.want, len(failed), failed)
			}
		})
	}
}

func TestCreateTempDir(t *testing.T) {
	dir, err := CreateTempDir("patchwork-test-")
	if err != nil {
		t.Fatalf("CreateTempDir unexpected error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty dir")
	}
	defer Cleanup(dir)

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat temp dir failed: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected path to be a directory")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcFile := filepath.Join(src, "test.txt")
	if err := os.WriteFile(srcFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir unexpected error: %v", err)
	}

	dstFile := filepath.Join(dst, "test.txt")
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Error("expected copied file to exist")
	}
}

func TestCleanup(t *testing.T) {
	dir, err := os.MkdirTemp("", "patchwork-test-cleanup")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	if err := Cleanup(dir); err != nil {
		t.Fatalf("Cleanup unexpected error: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected directory to be removed, stat err: %v", err)
	}
}

func TestCleanupEmptyPath(t *testing.T) {
	if err := Cleanup(""); err != nil {
		t.Errorf("Cleanup empty path should not error: %v", err)
	}
}
