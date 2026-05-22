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

	commands := [][]string{
		{"bundle", "exec", "rspec"},
		{"bundle", "exec", "rake", "test"},
		{"bundle", "exec", "rake"},
	}

	var lastOutput []byte
	var lastErr error

	for _, args := range commands {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir

		output, err := cmd.CombinedOutput()
		lastOutput = output
		lastErr = err

		if err == nil {
			duration := time.Since(start)
			result := &TestResult{
				Output:   string(output),
				Duration: duration,
				Passed:   true,
			}
			logger.Info("ruby tests passed", "duration", duration)
			return result, nil
		}
	}

	duration := time.Since(start)
	outputStr := string(lastOutput)

	result := &TestResult{
		Output:       outputStr,
		Duration:     duration,
		Passed:       false,
		FailedTests:  parseRSpecFailedTests(outputStr),
	}
	logger.Warn("ruby tests failed", "duration", duration, "error", lastErr, "failed_count", len(result.FailedTests))
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
