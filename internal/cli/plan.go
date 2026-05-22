package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/spf13/cobra"
)

var (
	planDir    string
	planFormat string
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate an upgrade plan",
	Long:  "Create a detailed plan for applying package upgrades across the codebase.",
	RunE:  runPlan,
}

func init() {
	planCmd.Flags().StringVar(&planDir, "dir", ".", "directory to scan")
	planCmd.Flags().StringVar(&planFormat, "format", "table", "output format (table|json)")
}

func runPlan(cmd *cobra.Command, args []string) error {
	ctx := signalContext()

	detections, err := detectorRegistry.DetectAll(ctx, planDir)
	if err != nil {
		return fmt.Errorf("detect packages: %w", err)
	}

	changelogs, err := analyzerRegistry.AnalyzeAll(ctx, detections)
	if err != nil {
		logger.Warn("some analyses failed", "error", err)
	}

	plan, err := plannerInstance.Plan(ctx, detections, changelogs)
	if err != nil {
		return fmt.Errorf("generate plan: %w", err)
	}

	if planFormat == "json" {
		return outputPlanJSON(plan)
	}
	return outputPlanTable(plan)
}

func outputPlanTable(plan *domain.PlanResult) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

func outputPlanJSON(plan *domain.PlanResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(plan)
}
