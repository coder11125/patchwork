package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/coder11125/patchwork/internal/analyzer"
	"github.com/coder11125/patchwork/internal/codemod"
	"github.com/coder11125/patchwork/internal/detector"
	"github.com/coder11125/patchwork/internal/pipeline"
	"github.com/coder11125/patchwork/internal/planner"
	"github.com/coder11125/patchwork/internal/pr"
	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/internal/testrunner"
	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/spf13/cobra"
)

var (
	runDir    string
	runFormat string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the full upgrade pipeline",
	Long:  "Execute detect, analyze, plan, apply, and pr in sequence for a complete automated upgrade workflow.",
	RunE:  runRun,
}

func init() {
	runCmd.Flags().StringVar(&runDir, "dir", ".", "directory to scan")
	runCmd.Flags().StringVar(&runFormat, "format", "table", "output format (table|json)")
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	absDir, err := filepath.Abs(runDir)
	if err != nil {
		return fmt.Errorf("resolve directory: %w", err)
	}

	detReg := detector.NewRegistry()
	detReg.Register(&detector.GoModDetector{})
	detReg.Register(&detector.NPMDetector{})
	detReg.Register(&detector.PipDetector{})

	anReg := analyzer.NewRegistry()

	recipeStore, err := recipe.NewDiskStore(appConfig.RecipeDir, appConfig.EpisodeDir, logger)
	if err != nil {
		return fmt.Errorf("create recipe store: %w", err)
	}
	plan := planner.New(recipeStore)

	cmReg := codemod.NewRegistry()
	cmReg.Register(codemod.NewRegexModifier())
	cmReg.Register(codemod.NewGoModModifier())
	cmReg.Register(codemod.NewPackageJSONModifier())
	cmReg.Register(codemod.NewRequirementsModifier())

	trReg := testrunner.NewRegistry()
	trReg.Register(testrunner.NewGoTestRunner())
	trReg.Register(testrunner.NewNPMTestRunner())

	var prCreator pr.PRCreator
	gitCfg := appConfig.GitConfig()
	if gitCfg.Platform != "" && gitCfg.Token != "" {
		prCreator, err = pr.NewCreator(gitCfg)
		if err != nil {
			logger.Warn("PR creator unavailable", "error", err)
		}
	}

	p := pipeline.New(appConfig, &gitCfg, absDir, detReg, anReg, plan, cmReg, trReg, prCreator, recipeStore)

	planResult, err := p.Run(ctx)
	if err != nil {
		return err
	}

	if runFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(planResult)
	}
	return outputRunTable(planResult)
}

func outputRunTable(plan *domain.PlanResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Pipeline completed\n")
	fmt.Fprintf(w, "Total risk: %s\n", plan.TotalRisk)
	fmt.Fprintf(w, "Upgrades planned: %d\n", len(plan.Upgrades))
	fmt.Fprintf(w, "Recipes matched: %d\n", len(plan.RecipesMatched))
	if len(plan.Blockers) > 0 {
		fmt.Fprintf(w, "Blockers: %v\n", plan.Blockers)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "ORDER\tPACKAGE\tFROM\tTO\tRISK\tRECIPE")
	for _, u := range plan.Upgrades {
		recipe := "none"
		if u.Recipe != nil {
			recipe = u.Recipe.Name
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", u.Order, u.Upgrade.Name, u.Upgrade.Current, u.Upgrade.Target, u.Upgrade.RiskLevel, recipe)
	}
	return w.Flush()
}
