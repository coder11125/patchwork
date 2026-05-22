package testrunner

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder11125/patchwork/pkg/domain"
)

type MavenTestRunner struct{}

func NewMavenTestRunner() *MavenTestRunner {
	return &MavenTestRunner{}
}

func (r *MavenTestRunner) Ecosystem() domain.Ecosystem {
	return domain.EcosystemMaven
}

func (r *MavenTestRunner) CanHandle(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "pom.xml"))
	return err == nil
}

func (r *MavenTestRunner) Run(ctx context.Context, dir string) (*TestResult, error) {
	logger := slog.With("runner", "maven", "dir", dir)
	logger.Info("running maven tests")

	start := time.Now()

	cmd := exec.CommandContext(ctx, "mvn", "test")
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
		result.FailedTests = parseMavenFailedTests(outputStr)
		logger.Warn("maven tests failed", "duration", duration, "failed_count", len(result.FailedTests))
		return result, nil
	}

	result.Passed = true
	logger.Info("maven tests passed", "duration", duration)
	return result, nil
}

func parseMavenFailedTests(output string) []string {
	var failed []string
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "FAILURE") || strings.Contains(line, "FAILED") {
			if i > 0 {
				failed = append(failed, strings.TrimSpace(lines[i-1]))
			}
		}
	}
	return failed
}
