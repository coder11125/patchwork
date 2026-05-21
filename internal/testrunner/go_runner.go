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

type GoTestRunner struct{}

func NewGoTestRunner() *GoTestRunner {
	return &GoTestRunner{}
}

func (r *GoTestRunner) Ecosystem() domain.Ecosystem {
	return domain.EcosystemGo
}

func (r *GoTestRunner) CanHandle(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

func (r *GoTestRunner) Run(ctx context.Context, dir string) (*TestResult, error) {
	logger := slog.With("runner", "go", "dir", dir)
	logger.Info("running go tests")

	start := time.Now()

	cmd := exec.CommandContext(ctx, "go", "test", "./...")
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
		result.FailedTests = parseGoFailedTests(outputStr)
		logger.Warn("go tests failed", "duration", duration, "failed_count", len(result.FailedTests))
		return result, nil
	}

	result.Passed = true
	logger.Info("go tests passed", "duration", duration)
	return result, nil
}

func parseGoFailedTests(output string) []string {
	var failed []string
	re := regexp.MustCompile(`^--- FAIL: (\S+)`)
	for _, line := range strings.Split(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			failed = append(failed, matches[1])
		}
	}
	return failed
}
