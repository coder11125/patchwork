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

type CargoTestRunner struct{}

func NewCargoTestRunner() *CargoTestRunner {
	return &CargoTestRunner{}
}

func (r *CargoTestRunner) Ecosystem() domain.Ecosystem {
	return domain.EcosystemCargo
}

func (r *CargoTestRunner) CanHandle(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "Cargo.toml"))
	return err == nil
}

func (r *CargoTestRunner) Run(ctx context.Context, dir string) (*TestResult, error) {
	logger := slog.With("runner", "cargo", "dir", dir)
	logger.Info("running cargo tests")

	start := time.Now()

	cmd := exec.CommandContext(ctx, "cargo", "test")
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
		result.FailedTests = parseCargoFailedTests(outputStr)
		logger.Warn("cargo tests failed", "duration", duration, "failed_count", len(result.FailedTests))
		return result, nil
	}

	result.Passed = true
	logger.Info("cargo tests passed", "duration", duration)
	return result, nil
}

func parseCargoFailedTests(output string) []string {
	var failed []string
	re := regexp.MustCompile(`^test\s+(\S+)\s+\.\.\.\s+FAILED`)
	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			failed = append(failed, matches[1])
		}
	}
	return failed
}
