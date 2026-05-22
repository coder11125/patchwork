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
	analyzePackage   string
	analyzeEcosystem string
	analyzeDir       string
	analyzeFormat    string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze packages for upgrade opportunities",
	Long:  "Analyze detected packages for available upgrades and breaking changes.",
	RunE:  runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringVar(&analyzePackage, "package", "", "package to analyze")
	analyzeCmd.Flags().StringVar(&analyzeEcosystem, "ecosystem", "", "ecosystem filter")
	analyzeCmd.Flags().StringVar(&analyzeDir, "dir", ".", "directory to scan")
	analyzeCmd.Flags().StringVar(&analyzeFormat, "format", "table", "output format (table|json)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	ctx := signalContext()

	var ecosystemFilter domain.Ecosystem
	if analyzeEcosystem != "" {
		ecosystemFilter = domain.Ecosystem(analyzeEcosystem)
	}

	results, err := detectorRegistry.DetectAll(ctx, analyzeDir)
	if err != nil {
		return fmt.Errorf("detect packages: %w", err)
	}

	if ecosystemFilter != "" {
		var filtered []*domain.DetectResult
		for _, r := range results {
			if r.Ecosystem == ecosystemFilter {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	if analyzePackage != "" {
		var filtered []*domain.DetectResult
		for _, r := range results {
			var pkgs []domain.PackageInfo
			for _, p := range r.Packages {
				if p.Name == analyzePackage {
					pkgs = append(pkgs, p)
				}
			}
			if len(pkgs) > 0 {
				r.Packages = pkgs
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	changelogs, err := analyzerRegistry.AnalyzeAll(ctx, results)
	if err != nil {
		logger.Warn("some analyses failed", "error", err)
	}

	if analyzeFormat == "json" {
		return outputAnalyzeJSON(results, changelogs)
	}
	return outputAnalyzeTable(results, changelogs)
}

func outputAnalyzeTable(results []*domain.DetectResult, changelogs []*domain.ChangelogEntry) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PACKAGE\tECOSYSTEM\tCURRENT\tLATEST\tBREAKING CHANGES")

	changelogMap := make(map[string]*domain.ChangelogEntry)
	for _, c := range changelogs {
		changelogMap[c.Version] = c
	}

	for _, r := range results {
		for _, p := range r.Packages {
			if !p.IsOutdated {
				continue
			}
			bc := 0
			if cl, ok := changelogMap[p.Latest]; ok {
				bc = len(cl.BreakingChanges)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", p.Name, r.Ecosystem, p.Current, p.Latest, bc)
		}
	}
	return w.Flush()
}

func outputAnalyzeJSON(results []*domain.DetectResult, changelogs []*domain.ChangelogEntry) error {
	output := struct {
		Results    []*domain.DetectResult   `json:"results"`
		Changelogs []*domain.ChangelogEntry `json:"changelogs"`
	}{
		Results:    results,
		Changelogs: changelogs,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
