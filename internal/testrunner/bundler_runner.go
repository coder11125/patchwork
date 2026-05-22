package testrunner

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type BundlerTestRunner struct{}

func NewBundlerTestRunner() *BundlerTestRunner {
	return &BundlerTestRunner{}
}

func (r *BundlerTestRunner) Ecosystem() domain.Ecosystem {
	return domain.EcosystemRuby
}

func (r *BundlerTestRunner) CanHandle(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "Gemfile"))
	return err == nil
}

func (r *BundlerTestRunner) Run(ctx context.Context, dir string) (*TestResult, error) {
	logger := slog.With("runner", "bundler", "dir", dir)
	logger.Info("running ruby tests")

	start := time.Now()

	cmd := exec.CommandContext(ctx, "bundle", "exec", "rspec")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	outputStr := string(output)

	result := &TestResult{
		Output:   outputStr,
		Duration: duration,
	}

	if err != nil {
		result.Passed = false
		result.FailedTests = parseRSpecFailedTests(outputStr)
		logger.Warn("ruby tests failed", "duration", duration, "failed_count", len(result.FailedTests))
		return result, nil
	}

	result.Passed = true
	logger.Info("ruby tests passed", "duration", duration)
	return result, nil
}

func parseRSpecFailedTests(output string) []string {
	var failed []string
	re := regexp.MustCompile(`^\d+\)\s+(.+)$`)
	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			failed = append(failed, matches[1])
		}
	}
	return failed
}
