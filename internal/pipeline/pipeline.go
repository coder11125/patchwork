package pipeline

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coder11125/patchwork/internal/analyzer"
	"github.com/coder11125/patchwork/internal/codemod"
	"github.com/coder11125/patchwork/internal/detector"
	"github.com/coder11125/patchwork/internal/planner"
	"github.com/coder11125/patchwork/internal/pr"
	"github.com/coder11125/patchwork/internal/recipe"
	"github.com/coder11125/patchwork/internal/testrunner"
	"github.com/coder11125/patchwork/pkg/domain"
)

type Pipeline struct {
	detectors   *detector.DetectorRegistry
	analyzers   *analyzer.Registry
	planner     *planner.Planner
	codemods    *codemod.Registry
	testRunners *testrunner.Registry
	prCreator   pr.PRCreator
	recipeStore recipe.RecipeStore
	config      *domain.Config
	gitConfig   *domain.GitConfig
	workDir     string
	logger      *slog.Logger
}

func New(cfg *domain.Config, gitCfg *domain.GitConfig, workDir string, detectors *detector.DetectorRegistry, analyzers *analyzer.Registry, planner *planner.Planner, codemods *codemod.Registry, testRunners *testrunner.Registry, prCreator pr.PRCreator, recipeStore recipe.RecipeStore) *Pipeline {
	return &Pipeline{
		detectors:   detectors,
		analyzers:   analyzers,
		planner:     planner,
		codemods:    codemods,
		testRunners: testRunners,
		prCreator:   prCreator,
		recipeStore: recipeStore,
		config:      cfg,
		gitConfig:   gitCfg,
		workDir:     workDir,
		logger:      slog.Default(),
	}
}

func (p *Pipeline) SetWorkDir(dir string) {
	p.workDir = dir
}

func (p *Pipeline) Run(ctx context.Context) (*domain.PlanResult, error) {
	detections, err := p.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect stage failed: %w", err)
	}

	changelogs, err := p.Analyze(ctx, detections)
	if err != nil {
		return nil, fmt.Errorf("analyze stage failed: %w", err)
	}

	plan, err := p.Plan(ctx, detections, changelogs)
	if err != nil {
		return nil, fmt.Errorf("plan stage failed: %w", err)
	}

	if p.config.DryRun {
		p.logger.Info("dry run mode: stopping after plan",
			"upgrades", len(plan.Upgrades),
			"total_risk", plan.TotalRisk,
		)
		for _, u := range plan.Upgrades {
			p.logger.Info("would upgrade",
				"package", u.Upgrade.Name,
				"from", u.Upgrade.Current,
				"to", u.Upgrade.Target,
				"risk", u.Upgrade.RiskLevel,
			)
		}
		return plan, nil
	}

	if err := p.Apply(ctx, plan); err != nil {
		return nil, fmt.Errorf("apply stage failed: %w", err)
	}

	prs, err := p.CreatePRs(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("pr stage failed: %w", err)
	}

	p.logger.Info("pipeline completed successfully", "prs_created", len(prs))
	return plan, nil
}

func (p *Pipeline) Detect(ctx context.Context) ([]*domain.DetectResult, error) {
	p.logger.Info("running detect stage")
	results, err := p.detectors.DetectAll(ctx, p.workDir)
	if err != nil {
		return results, fmt.Errorf("detect: %w", err)
	}

	total := 0
	for _, r := range results {
		total += len(r.Packages)
	}
	p.logger.Info("detect stage completed", "ecosystems", len(results), "packages", total)
	return results, nil
}

func (p *Pipeline) Analyze(ctx context.Context, detections []*domain.DetectResult) ([]*domain.ChangelogEntry, error) {
	p.logger.Info("running analyze stage")
	changelogs, err := p.analyzers.AnalyzeAll(ctx, detections)
	if err != nil {
		return changelogs, fmt.Errorf("analyze: %w", err)
	}
	p.logger.Info("analyze stage completed", "changelogs", len(changelogs))
	return changelogs, nil
}

func (p *Pipeline) Plan(ctx context.Context, detections []*domain.DetectResult, changelogs []*domain.ChangelogEntry) (*domain.PlanResult, error) {
	p.logger.Info("running plan stage")
	plan, err := p.planner.Plan(ctx, detections, changelogs)
	if err != nil {
		return plan, fmt.Errorf("plan: %w", err)
	}
	p.logger.Info("plan stage completed", "upgrades", len(plan.Upgrades), "risk", plan.TotalRisk)
	return plan, nil
}

func (p *Pipeline) Apply(ctx context.Context, plan *domain.PlanResult) error {
	p.logger.Info("running apply stage")

	for i, upgrade := range plan.Upgrades {
		if upgrade.Recipe == nil {
			p.logger.Warn("no recipe for upgrade, skipping apply",
				"package", upgrade.Upgrade.Name,
				"order", i+1,
			)
			continue
		}

		ep := StartEpisode(upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target, upgrade.Upgrade.Ecosystem)
		ep.RecipeUsed = upgrade.Recipe.ID

		AddEpisodeStep(ep, "apply", "running", fmt.Sprintf("applying recipe %s to %s", upgrade.Recipe.Name, upgrade.Upgrade.Name))

		if err := p.codemods.Apply(ctx, p.workDir, upgrade.Recipe); err != nil {
			AddEpisodeStep(ep, "apply", "failed", err.Error())
			FinalizeEpisode(ep, false, err.Error())
			_ = RecordEpisode(ctx, p.recipeStore, ep)
			return fmt.Errorf("apply upgrade %s: %w", upgrade.Upgrade.Name, err)
		}

		AddEpisodeStep(ep, "apply", "completed", "codemod applied successfully")

		if !p.config.SkipTests {
			result, err := p.testRunners.RunTests(ctx, upgrade.Upgrade.Ecosystem, p.workDir)
			if err != nil {
				AddEpisodeStep(ep, "test", "error", err.Error())
				FinalizeEpisode(ep, false, fmt.Sprintf("test error: %v", err))
				_ = RecordEpisode(ctx, p.recipeStore, ep)
				return fmt.Errorf("test upgrade %s: %w", upgrade.Upgrade.Name, err)
			}
			if result.Passed {
				ep.TestResult = domain.TestPassed
				AddEpisodeStep(ep, "test", "passed", "tests passed")
			} else {
				ep.TestResult = domain.TestFailed
				AddEpisodeStep(ep, "test", "failed", "tests failed after upgrade")
				FinalizeEpisode(ep, false, "tests failed")
				_ = RecordEpisode(ctx, p.recipeStore, ep)
				return fmt.Errorf("tests failed for upgrade %s", upgrade.Upgrade.Name)
			}
		} else {
			ep.TestResult = domain.TestSkipped
			AddEpisodeStep(ep, "test", "skipped", "tests skipped by configuration")
		}

		FinalizeEpisode(ep, true, "")
		if err := RecordEpisode(ctx, p.recipeStore, ep); err != nil {
			p.logger.Warn("failed to record episode", "error", err)
		}
	}

	p.logger.Info("apply stage completed")
	return nil
}

func (p *Pipeline) CreatePRs(ctx context.Context, plan *domain.PlanResult) ([]*pr.PRResponse, error) {
	p.logger.Info("running pr stage")

	if p.prCreator == nil {
		p.logger.Warn("PR creator not configured, skipping PR creation")
		return nil, nil
	}

	var prs []*pr.PRResponse

	for _, upgrade := range plan.Upgrades {
		branch := fmt.Sprintf("patchwork/%s-%s-to-%s", upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target)
		title := fmt.Sprintf("Upgrade %s from %s to %s", upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target)
		body := fmt.Sprintf("Automated upgrade of %s from %s to %s.\n\nRisk level: %s\nBreaking changes: %v",
			upgrade.Upgrade.Name, upgrade.Upgrade.Current, upgrade.Upgrade.Target, upgrade.Upgrade.RiskLevel, upgrade.Upgrade.IsBreaking)

		resp, err := p.prCreator.CreatePR(ctx, pr.PRRequest{
			Owner: p.gitConfig.Owner,
			Repo:  p.gitConfig.Repo,
			Head:  branch,
			Base:  p.gitConfig.PRTargetBranch,
			Title: title,
			Body:  body,
		})
		if err != nil {
			return prs, fmt.Errorf("create pr for %s: %w", upgrade.Upgrade.Name, err)
		}
		prs = append(prs, resp)
	}

	p.logger.Info("pr stage completed", "prs_created", len(prs))
	return prs, nil
}
