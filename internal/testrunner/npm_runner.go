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

type NPMTestRunner struct{}

func NewNPMTestRunner() *NPMTestRunner {
	return &NPMTestRunner{}
}

func (r *NPMTestRunner) Ecosystem() domain.Ecosystem {
	return domain.EcosystemNPM
}

func (r *NPMTestRunner) CanHandle(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}

func (r *NPMTestRunner) Run(ctx context.Context, dir string) (*TestResult, error) {
	logger := slog.With("runner", "npm", "dir", dir)
	logger.Info("running npm tests")

	start := time.Now()

	cmd := exec.CommandContext(ctx, "npm", "test")
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
		result.FailedTests = parseNPMFailedTests(outputStr)
		logger.Warn("npm tests failed", "duration", duration, "failed_count", len(result.FailedTests))
		return result, nil
	}

	result.Passed = true
	logger.Info("npm tests passed", "duration", duration)
	return result, nil
}

func parseNPMFailedTests(output string) []string {
	var failed []string
	seen := make(map[string]bool)

	jestRe := regexp.MustCompile(`^\s*(?:✕|×)\s+(.+)$`)
	mochaRe := regexp.MustCompile(`^\s+\d+\)\s+(.+)$`)
	genericRe := regexp.MustCompile(`(?:FAIL|failed)\s+(.+?)(?:\s|$)`)

	for _, line := range strings.Split(output, "\n") {
		if matches := jestRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.TrimSpace(matches[1])
			if name != "" && !seen[name] {
				seen[name] = true
				failed = append(failed, name)
			}
		} else if matches := mochaRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.TrimSpace(matches[1])
			if name != "" && !seen[name] {
				seen[name] = true
				failed = append(failed, name)
			}
		} else if matches := genericRe.FindStringSubmatch(line); len(matches) > 1 {
			name := strings.TrimSpace(matches[1])
			if name != "" && !seen[name] && len(name) > 1 {
				seen[name] = true
				failed = append(failed, name)
			}
		}
	}

	return failed
}
