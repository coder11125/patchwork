package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/coder11125/patchwork/internal/detector"
	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/spf13/cobra"
)

var (
	detectDir       string
	detectEcosystem string
	detectOutput    string
	detectFormat    string
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect packages and ecosystems in a directory",
	Long:  "Scan a directory for package manifests and detect the ecosystems and packages present.",
	RunE:  runDetect,
}

func init() {
	detectCmd.Flags().StringVar(&detectDir, "dir", ".", "directory to scan")
	detectCmd.Flags().StringVar(&detectEcosystem, "ecosystem", "", "filter by ecosystem (go, npm, pip, cargo, bundler, maven)")
	detectCmd.Flags().StringVar(&detectOutput, "output", "", "write JSON output to file")
	detectCmd.Flags().StringVar(&detectFormat, "format", "table", "output format: table or json")
}

func runDetect(cmd *cobra.Command, args []string) error {
	registry := detector.NewRegistry()
	registry.Register(&detector.GoModDetector{})
	registry.Register(&detector.NPMDetector{})
	registry.Register(&detector.PipDetector{})
	registry.Register(&detector.CargoDetector{})
	registry.Register(&detector.BundlerDetector{})
	registry.Register(&detector.MavenDetector{})

	ctx := signalContext()

	var results []*domain.DetectResult
	var err error

	if detectEcosystem != "" {
		d, err := registry.ForEcosystem(domain.Ecosystem(detectEcosystem))
		if err != nil {
			return fmt.Errorf("unknown ecosystem %q: %w", detectEcosystem, err)
		}
		result, err := d.Detect(ctx, detectDir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}
		if result != nil {
			results = append(results, result)
		}
	} else {
		results, err = registry.DetectAll(ctx, detectDir)
		if err != nil {
			return fmt.Errorf("detection failed: %w", err)
		}
	}

	if detectOutput != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling results: %w", err)
		}
		if err := os.WriteFile(detectOutput, data, 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		logger.Info("results written to file", "path", detectOutput)
	}

	switch detectFormat {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling results: %w", err)
		}
		fmt.Println(string(data))
	case "table":
		printTable(results)
	default:
		return fmt.Errorf("unknown format %q, expected table or json", detectFormat)
	}

	if len(results) == 0 {
		logger.Info("no packages detected")
	}

	return nil
}

func printTable(results []*domain.DetectResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ECOSYSTEM\tPACKAGE\tCURRENT\tLATEST\tMANIFEST")
	for _, r := range results {
		for _, pkg := range r.Packages {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Ecosystem, pkg.Name, pkg.Current, pkg.Latest, r.ManifestPath)
		}
	}
	w.Flush()
}
