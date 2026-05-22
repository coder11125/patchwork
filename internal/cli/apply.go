package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/spf13/cobra"
)

var (
	applyDir  string
	applyPlan string
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply an upgrade plan to the codebase",
	Long:  "Execute the generated upgrade plan, modifying source files and manifests.",
	RunE:  runApply,
}

func init() {
	applyCmd.Flags().StringVar(&applyDir, "dir", ".", "directory to apply changes to")
	applyCmd.Flags().StringVar(&applyPlan, "plan", "", "path to a JSON plan file (optional, generates fresh plan if omitted)")
}

func runApply(cmd *cobra.Command, args []string) error {
	ctx := signalContext()

	var plan *domain.PlanResult

	if applyPlan != "" {
		data, err := os.ReadFile(applyPlan)
		if err != nil {
			return fmt.Errorf("read plan file: %w", err)
		}
		plan = &domain.PlanResult{}
		if err := json.Unmarshal(data, plan); err != nil {
			return fmt.Errorf("parse plan file: %w", err)
		}
	} else {
		detections, err := detectorRegistry.DetectAll(ctx, applyDir)
		if err != nil {
			return fmt.Errorf("detect packages: %w", err)
		}

		changelogs, err := analyzerRegistry.AnalyzeAll(ctx, detections)
		if err != nil {
			logger.Warn("some analyses failed", "error", err)
		}

		plan, err = plannerInstance.Plan(ctx, detections, changelogs)
		if err != nil {
			return fmt.Errorf("generate plan: %w", err)
		}
	}

	if appConfig.DryRun {
		logger.Info("dry run: would apply the following upgrades")
		for _, u := range plan.Upgrades {
			logger.Info("would upgrade",
				"package", u.Upgrade.Name,
				"from", u.Upgrade.Current,
				"to", u.Upgrade.Target,
				"risk", u.Upgrade.RiskLevel,
			)
		}
		return nil
	}

	for i, upgrade := range plan.Upgrades {
		if upgrade.Recipe == nil {
			logger.Warn("no recipe for upgrade, skipping",
				"package", upgrade.Upgrade.Name,
				"order", i+1,
			)
			continue
		}

		logger.Info("applying upgrade",
			"package", upgrade.Upgrade.Name,
			"from", upgrade.Upgrade.Current,
			"to", upgrade.Upgrade.Target,
			"recipe", upgrade.Recipe.Name,
		)

		absDir, err := filepath.Abs(applyDir)
		if err != nil {
			return fmt.Errorf("resolve directory: %w", err)
		}

		if err := codemodRegistry.Apply(ctx, absDir, upgrade.Recipe); err != nil {
			return fmt.Errorf("apply codemod for %s: %w", upgrade.Upgrade.Name, err)
		}

		if !appConfig.SkipTests {
			logger.Info("running tests for", "package", upgrade.Upgrade.Name)
			result, err := testRunnerRegistry.RunTests(ctx, upgrade.Upgrade.Ecosystem, absDir)
			if err != nil {
				return fmt.Errorf("run tests for %s: %w", upgrade.Upgrade.Name, err)
			}
			if !result.Passed {
				return fmt.Errorf("tests failed for %s: %s", upgrade.Upgrade.Name, result.Output)
			}
			logger.Info("tests passed", "package", upgrade.Upgrade.Name)
		}
	}

	logger.Info("apply completed", "upgrades_applied", len(plan.Upgrades))
	return nil
}
