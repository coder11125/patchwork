package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coder11125/patchwork/internal/pr"
	"github.com/spf13/cobra"
)

var (
	prDir    string
	prFormat string
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create pull requests for upgrades",
	Long:  "Open pull requests for each upgrade change, optionally batching related changes together.",
	RunE:  runPR,
}

func init() {
	prCmd.Flags().StringVar(&prDir, "dir", ".", "directory to scan")
	prCmd.Flags().StringVar(&prFormat, "format", "table", "output format (table|json)")
}

func runPR(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if prCreatorInstance == nil {
		return fmt.Errorf("PR creator not configured: set git_platform and git_token in config or environment")
	}

	detections, err := detectorRegistry.DetectAll(ctx, prDir)
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

	gitCfg := appConfig.GitConfig()
	var responses []*pr.PRResponse

	for _, upgrade := range plan.Upgrades {
		branch := fmt.Sprintf("patchwork/%s-%s-to-%s", upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target)
		title := fmt.Sprintf("Upgrade %s from %s to %s", upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target)
		body := fmt.Sprintf("Automated upgrade of %s from %s to %s.\n\nRisk level: %s\nBreaking changes: %v",
			upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target, upgrade.Upgrade.RiskLevel, upgrade.Upgrade.IsBreaking)

		resp, err := prCreatorInstance.CreatePR(ctx, pr.PRRequest{
			Owner: gitCfg.Owner,
			Repo:  gitCfg.Repo,
			Head:  branch,
			Base:  gitCfg.PRTargetBranch,
			Title: title,
			Body:  body,
		})
		if err != nil {
			var existsErr *pr.PRExistsError
			if ok := false; !ok {
				logger.Warn("failed to create PR", "package", upgrade.Upgrade.Name, "error", err)
				continue
			}
			_ = existsErr
		}
		responses = append(responses, resp)
	}

	if prFormat == "json" {
		return outputPRJSON(responses)
	}
	return outputPRTable(responses)
}

func outputPRTable(responses []*pr.PRResponse) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PR#\tURL\tSTATE")
	for _, r := range responses {
		if r == nil {
			continue
		}
		fmt.Fprintf(w, "%d\t%s\t%s\n", r.Number, r.URL, r.State)
	}
	return w.Flush()
}

func outputPRJSON(responses []*pr.PRResponse) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(responses)
}
