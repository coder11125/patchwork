package llm

import "fmt"

const (
	SystemRole    = "system"
	UserRole      = "user"
	AssistantRole = "assistant"
)

func SystemPrompt(content string) Message {
	return Message{Role: SystemRole, Content: content}
}

func UserPrompt(content string) Message {
	return Message{Role: UserRole, Content: content}
}

func AssistantPrompt(content string) Message {
	return Message{Role: AssistantRole, Content: content}
}

const changelogAnalysisSystemPrompt = `You are an expert software release engineer. Analyze git changelogs and commit histories to produce structured release notes.

Your task is to:
1. Categorize changes into: Features, Bug Fixes, Breaking Changes, Performance Improvements, and Internal Changes
2. Identify breaking changes and API modifications that require migration
3. Extract the semantic version bump recommendation (major, minor, patch)
4. Summarize the impact on downstream consumers

Be precise and technical. Focus on what changed and why, not how it was implemented.
Flag any changes that could break existing integrations or require code updates from consumers.`

const codemodGenerationSystemPrompt = `You are an expert codemod engineer. Generate AST-based code transformation scripts (codemods) that safely migrate codebases from one API version to another.

Your task is to:
1. Analyze the API diff between old and new versions
2. Generate precise code transformations that handle:
   - Function/method renames
   - Parameter changes (additions, removals, reordering)
   - Import path changes
   - Type signature changes
   - Deprecated API replacements
3. Ensure transformations are idempotent and safe
4. Include comments explaining each transformation rule

Output codemods in a structured format that can be applied programmatically.
Prioritize correctness over completeness - it is better to skip an ambiguous case than to produce an incorrect transformation.`

const migrationPlanningSystemPrompt = `You are an expert migration architect. Create detailed migration plans for upgrading software dependencies, frameworks, or APIs.

Your task is to:
1. Assess the scope and complexity of the migration
2. Break the migration into ordered, incremental steps
3. Identify risks and rollback strategies for each step
4. Estimate effort and prioritize critical path items
5. Specify testing requirements to validate each migration step

The plan should be executable: each step should be small enough to review and test independently.
Include pre-migration checks, the migration steps themselves, and post-migration validation.`

func BuildChangelogMessages(changelog string) []Message {
	return []Message{
		SystemPrompt(changelogAnalysisSystemPrompt),
		UserPrompt(changelog),
	}
}

func BuildCodemodMessages(apiDiff string, language string) []Message {
	return []Message{
		SystemPrompt(codemodGenerationSystemPrompt),
		UserPrompt(fmt.Sprintf("Generate a codemod for %s.\n\nAPI Diff:\n%s", language, apiDiff)),
	}
}

func BuildMigrationPlanMessages(currentVersion string, targetVersion string, context string) []Message {
	userContent := fmt.Sprintf("Create a migration plan from version %s to %s.\n\nContext:\n%s", currentVersion, targetVersion, context)
	return []Message{
		SystemPrompt(migrationPlanningSystemPrompt),
		UserPrompt(userContent),
	}
}
