package cli

import (
	"log/slog"
	"os"

	"github.com/coder11125/patchwork/internal/analyzer"
	"github.com/coder11125/patchwork/internal/codemod"
	"github.com/coder11125/patchwork/internal/config"
	"github.com/coder11125/patchwork/internal/detector"
	"github.com/coder11125/patchwork/internal/planner"
	"github.com/coder11125/patchwork/internal/pr"
	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/internal/testrunner"
	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use:   "patchwork",
	Short: "Automated dependency upgrade tool",
	Long:  "Patchwork detects outdated dependencies, analyzes breaking changes, and applies upgrades safely.",
}

var (
	cfgFile   string
	verbose   bool
	dryRun    bool
	appConfig *domain.Config
	logger    *slog.Logger

	detectorRegistry    *detector.DetectorRegistry
	analyzerRegistry    *analyzer.Registry
	plannerInstance     *planner.Planner
	codemodRegistry     *codemod.Registry
	testRunnerRegistry  *testrunner.Registry
	prCreatorInstance   pr.PRCreator
	recipeStoreInstance recipe.RecipeStore
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate actions without making changes")

	rootCmd.PersistentPreRunE = loadConfig

	rootCmd.AddCommand(detectCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(serveCmd)
}

func loadConfig(cmd *cobra.Command, args []string) error {
	flagSet := pflag.NewFlagSet("patchwork", pflag.ContinueOnError)
	flagSet.AddFlagSet(cmd.PersistentFlags())
	flagSet.AddFlagSet(cmd.Flags())

	var err error
	appConfig, err = config.LoadConfig(flagSet)
	if err != nil {
		return err
	}

	level := slog.LevelInfo
	if verbose || appConfig.Verbose {
		level = slog.LevelDebug
	}

	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	if dryRun || appConfig.DryRun {
		logger.Info("dry-run mode enabled")
	}

	initRegistries()
	return nil
}

func initRegistries() {
	detectorRegistry = detector.NewRegistry()
	detectorRegistry.Register(&detector.GoModDetector{})
	detectorRegistry.Register(&detector.NPMDetector{})
	detectorRegistry.Register(&detector.PipDetector{})
	detectorRegistry.Register(&detector.CargoDetector{})

	analyzerRegistry = analyzer.NewRegistry()

	var err error
	recipeStoreInstance, err = recipe.NewDiskStore(appConfig.RecipeDir, appConfig.EpisodeDir, logger)
	if err != nil {
		logger.Error("failed to create recipe store", "error", err)
	}
	plannerInstance = planner.New(recipeStoreInstance)

	codemodRegistry = codemod.NewRegistry()
	codemodRegistry.Register(codemod.NewRegexModifier())
	codemodRegistry.Register(codemod.NewGoModModifier())
	codemodRegistry.Register(codemod.NewPackageJSONModifier())
	codemodRegistry.Register(codemod.NewRequirementsModifier())
	codemodRegistry.Register(codemod.NewCargoModifier())

	testRunnerRegistry = testrunner.NewRegistry()
	testRunnerRegistry.Register(testrunner.NewGoTestRunner())
	testRunnerRegistry.Register(testrunner.NewNPMTestRunner())
	testRunnerRegistry.Register(testrunner.NewCargoTestRunner())

	if appConfig.LLMProvider != "" {
		llmCfg := domain.LLMConfig{
			Provider: domain.LLMProviderType(appConfig.LLMProvider),
			Model:    appConfig.LLMModel,
			APIKey:   appConfig.LLMAPIKey,
			BaseURL:  appConfig.LLMBaseURL,
		}
		_ = llmCfg
	}

	if appConfig.GitConfig().Platform != "" && appConfig.GitConfig().Token != "" {
		prCreatorInstance, err = pr.NewCreator(appConfig.GitConfig())
		if err != nil {
			logger.Warn("failed to create PR creator", "error", err)
		}
	}
}
