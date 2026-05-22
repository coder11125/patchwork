package domain

import "testing"

func TestEcosystemString(t *testing.T) {
	tests := []struct {
		e        Ecosystem
		expected string
	}{
		{EcosystemGo, "go"},
		{EcosystemNPM, "npm"},
		{EcosystemPip, "pip"},
		{EcosystemCargo, "cargo"},
		{Ecosystem("unknown"), "unknown"},
	}
	for _, tc := range tests {
		got := tc.e.String()
		if got != tc.expected {
			t.Errorf("Ecosystem(%q).String() = %q, want %q", tc.e, got, tc.expected)
		}
	}
}

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		r        RiskLevel
		expected string
	}{
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskCritical, "critical"},
		{RiskLevel("unknown"), "unknown"},
	}
	for _, tc := range tests {
		got := tc.r.String()
		if got != tc.expected {
			t.Errorf("RiskLevel(%q).String() = %q, want %q", tc.r, got, tc.expected)
		}
	}
}

func TestTestOutcomeString(t *testing.T) {
	tests := []struct {
		o        TestOutcome
		expected string
	}{
		{TestPassed, "passed"},
		{TestFailed, "failed"},
		{TestSkipped, "skipped"},
		{TestOutcome("unknown"), "unknown"},
	}
	for _, tc := range tests {
		got := tc.o.String()
		if got != tc.expected {
			t.Errorf("TestOutcome(%q).String() = %q, want %q", tc.o, got, tc.expected)
		}
	}
}

func TestLLMProviderTypeString(t *testing.T) {
	tests := []struct {
		p        LLMProviderType
		expected string
	}{
		{ProviderAnthropic, "anthropic"},
		{ProviderMistral, "mistral"},
		{ProviderGroq, "groq"},
		{ProviderOllama, "ollama"},
	}
	for _, tc := range tests {
		got := tc.p.String()
		if got != tc.expected {
			t.Errorf("LLMProviderType(%q).String() = %q, want %q", tc.p, got, tc.expected)
		}
	}
}

func TestGitConfig(t *testing.T) {
	cfg := &Config{
		GitRemote:   "origin",
		GitPRBranch: "main",
		GitPlatform: "github",
		GitToken:    "tok",
		GitOwner:    "testowner",
		GitRepo:     "testrepo",
	}
	gc := cfg.GitConfig()
	if gc.Remote != "origin" {
		t.Errorf("GitConfig().Remote = %q, want %q", gc.Remote, "origin")
	}
	if gc.PRTargetBranch != "main" {
		t.Errorf("GitConfig().PRTargetBranch = %q, want %q", gc.PRTargetBranch, "main")
	}
	if gc.Platform != "github" {
		t.Errorf("GitConfig().Platform = %q, want %q", gc.Platform, "github")
	}
	if gc.Token != "tok" {
		t.Errorf("GitConfig().Token = %q, want %q", gc.Token, "tok")
	}
	if gc.Owner != "testowner" {
		t.Errorf("GitConfig().Owner = %q, want %q", gc.Owner, "testowner")
	}
	if gc.Repo != "testrepo" {
		t.Errorf("GitConfig().Repo = %q, want %q", gc.Repo, "testrepo")
	}
}

func TestDefaultConfigDefaults(t *testing.T) {
	cfg := Config{}
	if cfg.LLMProvider != "" {
		t.Errorf("default LLMProvider should be empty")
	}
	if cfg.MaxRetries != 0 {
		t.Errorf("default MaxRetries should be 0")
	}
}
