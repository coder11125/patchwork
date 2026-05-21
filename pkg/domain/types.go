package domain

import "time"

type Ecosystem string

const (
	EcosystemGo    Ecosystem = "go"
	EcosystemNPM   Ecosystem = "npm"
	EcosystemPip   Ecosystem = "pip"
	EcosystemCargo Ecosystem = "cargo"
)

func (e Ecosystem) String() string {
	return string(e)
}

type LLMProviderType string

const (
	ProviderAnthropic LLMProviderType = "anthropic"
	ProviderMistral   LLMProviderType = "mistral"
	ProviderGroq      LLMProviderType = "groq"
	ProviderOllama    LLMProviderType = "ollama"
)

func (p LLMProviderType) String() string {
	return string(p)
}

type LLMConfig struct {
	Provider    LLMProviderType `json:"provider"`
	Model       string          `json:"model"`
	APIKey      string          `json:"api_key,omitempty"`
	BaseURL     string          `json:"base_url,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Timeout     string          `json:"timeout"`
	TimeoutSec  int             `json:"timeout_sec"`
}

type GitConfig struct {
	Remote         string `json:"remote"`
	PRTargetBranch string `json:"pr_target"`
	Platform       string `json:"platform"`
	Token          string `json:"token,omitempty"`
	Owner          string `json:"owner"`
	Repo           string `json:"repo"`
}

type Config struct {
	LLMProvider string `koanf:"llm_provider" json:"llm_provider"`
	LLMModel    string `koanf:"llm_model" json:"llm_model"`
	LLMBaseURL  string `koanf:"llm_base_url" json:"llm_base_url"`
	LLMAPIKey   string `koanf:"llm_api_key" json:"llm_api_key,omitempty"`

	RecipeDir  string `koanf:"recipe_dir" json:"recipe_dir"`
	EpisodeDir string `koanf:"episode_dir" json:"episode_dir"`
	CacheDir   string `koanf:"cache_dir" json:"cache_dir"`

	MaxRetries int  `koanf:"max_retries" json:"max_retries"`
	DryRun     bool `koanf:"dry_run" json:"dry_run"`
	SkipTests  bool `koanf:"skip_tests" json:"skip_tests"`
	AutoCommit bool `koanf:"auto_commit" json:"auto_commit"`
	Verbose    bool `koanf:"verbose" json:"verbose"`

	GitPlatform string `koanf:"git_platform" json:"git_platform"`
	GitToken    string `koanf:"git_token" json:"git_token,omitempty"`
	GitOwner    string `koanf:"git_owner" json:"git_owner"`
	GitRepo     string `koanf:"git_repo" json:"git_repo"`
	GitRemote   string `koanf:"git_remote" json:"git_remote"`
	GitPRBranch string `koanf:"git_pr_branch" json:"git_pr_branch"`
}

func (c *Config) GitConfig() GitConfig {
	return GitConfig{
		Remote:         c.GitRemote,
		PRTargetBranch: c.GitPRBranch,
		Platform:       c.GitPlatform,
		Token:          c.GitToken,
		Owner:          c.GitOwner,
		Repo:           c.GitRepo,
	}
}

type Package struct {
	Name           string            `json:"name"`
	Ecosystem      Ecosystem         `json:"ecosystem"`
	CurrentVersion string            `json:"current_version"`
	LatestVersion  string            `json:"latest_version"`
	ManifestPath   string            `json:"manifest_path"`
	IsTransitive   bool              `json:"is_transitive"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type PackageInfo struct {
	Name       string `json:"name"`
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	IsOutdated bool   `json:"is_outdated"`
	IsDirect   bool   `json:"is_direct"`
}

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

func (r RiskLevel) String() string {
	return string(r)
}

type BreakingChange struct {
	ID            string   `json:"id"`
	PackageName   string   `json:"package_name"`
	Version       string   `json:"version"`
	Description   string   `json:"description"`
	Severity      string   `json:"severity"`
	AffectedAPIs  []string `json:"affected_apis,omitempty"`
	MigrationHint string   `json:"migration_hint,omitempty"`
	SourceURL     string   `json:"source_url"`
}

type RecipeStep struct {
	Order       int      `json:"order"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Pattern     string   `json:"pattern,omitempty"`
	Replacement string   `json:"replacement,omitempty"`
	FileGlobs   []string `json:"file_globs,omitempty"`
	ManualHint  string   `json:"manual_hint,omitempty"`
}

type Recipe struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Ecosystem   Ecosystem    `json:"ecosystem"`
	PackageName string       `json:"package_name"`
	FromVersion string       `json:"from_version"`
	ToVersion   string       `json:"to_version"`
	Steps       []RecipeStep `json:"steps"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	SuccessRate float64      `json:"success_rate"`
	TimesUsed   int          `json:"times_used"`
	Tags        []string     `json:"tags,omitempty"`
}

type Upgrade struct {
	Name       string    `json:"name"`
	Current    string    `json:"current"`
	Target     string    `json:"target"`
	RiskLevel  RiskLevel `json:"risk_level"`
	IsBreaking bool      `json:"is_breaking"`
	Ecosystem  Ecosystem `json:"ecosystem"`
}

type UpgradePlan struct {
	Upgrade         Upgrade          `json:"upgrade"`
	BreakingChanges []BreakingChange `json:"breaking_changes"`
	Recipe          *Recipe          `json:"recipe,omitempty"`
	Codemods        []string         `json:"codemods"`
	TestCommand     string           `json:"test_command"`
	Order           int              `json:"order"`
}

type DetectResult struct {
	Ecosystem    Ecosystem     `json:"ecosystem"`
	Dir          string        `json:"dir"`
	ManifestPath string        `json:"manifest_path"`
	Packages     []PackageInfo `json:"packages"`
	Upgrades     []Upgrade     `json:"upgrades"`
	ScannedAt    time.Time     `json:"scanned_at"`
	DetectedAt   time.Time     `json:"detected_at"`
	Error        string        `json:"error,omitempty"`
}

type PlanResult struct {
	Upgrades          []UpgradePlan `json:"upgrades"`
	TotalRisk         RiskLevel     `json:"total_risk"`
	EstimatedDuration string        `json:"estimated_duration"`
	RecipesMatched    []string      `json:"recipes_matched"`
	Blockers          []string      `json:"blockers,omitempty"`
}

type EpisodeStep struct {
	Order  int    `json:"order"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type TestOutcome string

const (
	TestPassed  TestOutcome = "passed"
	TestFailed  TestOutcome = "failed"
	TestSkipped TestOutcome = "skipped"
)

func (t TestOutcome) String() string {
	return string(t)
}

type Episode struct {
	ID            string        `json:"id"`
	Timestamp     string        `json:"timestamp"`
	Ecosystem     Ecosystem     `json:"ecosystem"`
	PackageName   string        `json:"package_name"`
	FromVersion   string        `json:"from_version"`
	ToVersion     string        `json:"to_version"`
	RecipeUsed    string        `json:"recipe_used,omitempty"`
	StepsExecuted []EpisodeStep `json:"steps_executed"`
	TestResult    TestOutcome   `json:"test_result"`
	PRCreated     bool          `json:"pr_created"`
	PRURL         string        `json:"pr_url,omitempty"`
	Success       bool          `json:"success"`
	FailureReason string        `json:"failure_reason,omitempty"`
	LearnedRecipe string        `json:"learned_recipe,omitempty"`
	Duration      string        `json:"duration"`
}

type ChangelogEntry struct {
	Version         string           `json:"version"`
	ReleaseDate     string           `json:"release_date"`
	BreakingChanges []BreakingChange `json:"breaking_changes"`
	NewFeatures     []string         `json:"new_features,omitempty"`
	BugFixes        []string         `json:"bug_fixes,omitempty"`
	RawContent      string           `json:"raw_content"`
}
