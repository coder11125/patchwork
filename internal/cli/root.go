package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder11125/patchwork/internal/analyzer"
	"github.com/coder11125/patchwork/internal/codemod"
	"github.com/coder11125/patchwork/internal/config"
	"github.com/coder11125/patchwork/internal/detector"
	"github.com/coder11125/patchwork/internal/llm"
	"github.com/coder11125/patchwork/internal/planner"
	"github.com/coder11125/patchwork/internal/pr"
	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/internal/testrunner"
	"github.com/coder11125/patchwork/pkg/domain"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func signalContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	return ctx
}

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

	if err := initRegistries(); err != nil {
		return err
	}
	return nil
}

func initRegistries() error {
	detectorRegistry = detector.NewRegistry()
	detectorRegistry.Register(&detector.GoModDetector{})
	detectorRegistry.Register(&detector.NPMDetector{})
	detectorRegistry.Register(&detector.PipDetector{})
	detectorRegistry.Register(&detector.CargoDetector{})
	detectorRegistry.Register(&detector.BundlerDetector{})
	detectorRegistry.Register(&detector.MavenDetector{})

	analyzerRegistry = analyzer.NewRegistry()
	populateAnalyzerRegistry(analyzerRegistry)

	var err error
	recipeStoreInstance, err = recipe.NewDiskStore(appConfig.RecipeDir, appConfig.EpisodeDir, logger)
	if err != nil {
		return fmt.Errorf("create recipe store: %w", err)
	}
	plannerInstance = planner.New(recipeStoreInstance)

	codemodRegistry = codemod.NewRegistry()
	codemodRegistry.Register(codemod.NewRegexModifier())
	codemodRegistry.Register(codemod.NewGoModModifier())
	codemodRegistry.Register(codemod.NewPackageJSONModifier())
	codemodRegistry.Register(codemod.NewRequirementsModifier())
	codemodRegistry.Register(codemod.NewCargoModifier())
	codemodRegistry.Register(codemod.NewGemfileModifier())
	codemodRegistry.Register(codemod.NewMavenModifier())

	testRunnerRegistry = testrunner.NewRegistry()
	testRunnerRegistry.Register(testrunner.NewGoTestRunner())
	testRunnerRegistry.Register(testrunner.NewNPMTestRunner())
	testRunnerRegistry.Register(testrunner.NewCargoTestRunner())
	testRunnerRegistry.Register(testrunner.NewBundlerTestRunner())
	testRunnerRegistry.Register(testrunner.NewMavenTestRunner())

	if appConfig.GitConfig().Platform != "" && appConfig.GitConfig().Token != "" {
		prCreatorInstance, err = pr.NewCreator(appConfig.GitConfig())
		if err != nil {
			logger.Warn("failed to create PR creator", "error", err)
		}
	}

	return nil
}

func populateAnalyzerRegistry(reg *analyzer.Registry) {
	reg.Register(analyzer.NewGitHubAnalyzer(appConfig.GitToken))

	if appConfig.LLMProvider == "" || appConfig.LLMAPIKey == "" {
		return
	}

	provider, err := llm.NewProvider(domain.LLMConfig{
		Provider: domain.LLMProviderType(appConfig.LLMProvider),
		Model:    appConfig.LLMModel,
		APIKey:   appConfig.LLMAPIKey,
		BaseURL:  appConfig.LLMBaseURL,
	})
	if err != nil {
		logger.Warn("failed to create LLM provider, falling back to semver-only analysis", "error", err)
		return
	}

	for _, eco := range []domain.Ecosystem{
		domain.EcosystemGo,
		domain.EcosystemNPM,
		domain.EcosystemPip,
		domain.EcosystemCargo,
		domain.EcosystemRuby,
		domain.EcosystemMaven,
	} {
		reg.Register(analyzer.NewLLMAnalyzerForEcosystem(provider, eco))
	}
}
