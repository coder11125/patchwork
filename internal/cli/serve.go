package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/coder11125/patchwork/internal/analyzer"
	"github.com/coder11125/patchwork/internal/codemod"
	"github.com/coder11125/patchwork/internal/detector"
	"github.com/coder11125/patchwork/internal/pipeline"
	"github.com/coder11125/patchwork/internal/planner"
	"github.com/coder11125/patchwork/internal/pr"
	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/internal/server"
	"github.com/coder11125/patchwork/internal/testrunner"
	"github.com/spf13/cobra"
)

var (
	serveAddr string
	serveDir  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  "Run Patchwork as an HTTP API server for programmatic access.",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", ":8080", "address to listen on")
	serveCmd.Flags().StringVar(&serveDir, "dir", ".", "working directory for scans")
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	detReg := detector.NewRegistry()
	detReg.Register(&detector.GoModDetector{})
	detReg.Register(&detector.NPMDetector{})
	detReg.Register(&detector.PipDetector{})
	detReg.Register(&detector.CargoDetector{})

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
	cmReg.Register(codemod.NewCargoModifier())

	trReg := testrunner.NewRegistry()
	trReg.Register(testrunner.NewGoTestRunner())
	trReg.Register(testrunner.NewNPMTestRunner())
	trReg.Register(testrunner.NewCargoTestRunner())

	var prCreator pr.PRCreator
	gitCfg := appConfig.GitConfig()
	if gitCfg.Platform != "" && gitCfg.Token != "" {
		prCreator, err = pr.NewCreator(gitCfg)
		if err != nil {
			logger.Warn("PR creator unavailable", "error", err)
		}
	}

	p := pipeline.New(appConfig, &gitCfg, serveDir, detReg, anReg, plan, cmReg, trReg, prCreator, recipeStore)

	srv := server.New(p, serveAddr)
	return srv.Start(ctx)
}
