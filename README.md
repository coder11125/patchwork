# Patchwork

**BYOK (Bring Your Own Key) autonomous dependency upgrade framework.**

Patchwork detects outdated packages, analyzes changelogs for breaking changes, plans migration strategies, applies codemods, runs tests in isolation, and opens pull requests. It learns from each attempt by writing reusable recipes to disk.

## Features

- **Multi-ecosystem detection** — Go modules, npm, pip (requirements.txt), Cargo (Cargo.toml)
- **Breaking change analysis** — GitHub releases, changelog parsing, semver risk assessment, LLM-powered analysis
- **Recipe-driven learning** — Successful upgrades are saved as reusable recipes; future upgrades match against historical knowledge
- **Safe codemod application** — Regex-based transformations, manifest updates (go.mod, package.json, requirements.txt, Cargo.toml)
- **Isolated test execution** — Tests run in temp directories before changes touch your working tree
- **PR automation** — Creates GitHub/GitLab pull requests for each upgrade
- **Full pipeline** — `patchwork run` chains detect → analyze → plan → apply → pr
- **OS keychain integration** — API keys and tokens stored in native OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- **BYOK** — Bring your own LLM API key. Supports Anthropic Claude, Mistral, Groq, and local Ollama
- **Local-first** — Works fully offline with Ollama as the default provider
- **CLI-first** — Single binary, no server required

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         patchwork CLI                               │
│  configure │ detect │ analyze │ plan │ apply │ pr │ run │ serve      │
└────────────────────────┬────────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────────┐
│                        Pipeline                                     │
│  Detect → Analyze → Plan → Apply → PR                               │
└────┬──────────┬─────────┬────────┬────────┬─────────────────────────┘
     │          │         │        │        │
┌────▼───┐ ┌───▼────┐ ┌──▼────┐ ┌─▼─────┐ ┌▼────────┐
│Detector│ │Analyzer│ │Planner│ │Codemod│ │TestRunner│
│Registry│ │Registry│ │       │ │Registry│ │Registry  │
└────┬───┘ └───┬────┘ └──┬────┘ └─┬─────┘ └┬────────┘
     │          │         │        │        │
  ┌──▼────┐  ┌──▼──┐   ┌──▼──┐  ┌─▼──────┐ ┌─▼────────┐
  │go.mod  │  │GitHub│  │Recipe│  │regex   │ │go test   │
  │pkg.json│  │release│  │Store │  │go.mod  │ │npm test  │
  │req.txt │  │semver │  │      │  │pkg.json│ │cargo test│
  │Cargo   │  │LLM   │  │      │  │req.txt │ │          │
  │.toml   │  │      │  │      │  │Cargo   │ │          │
  └────────┘  └──────┘  └──────┘  │.toml   │ └──────────┘
                                  └────────┘           │
                                                  ┌─────▼─────┐
                                                  │   PRCreator │
                                                  │  GitHub API │
                                                  │  GitLab API │
                                                  └─────────────┘
```

### Package Layout

| Package | Responsibility |
|---|---|
| `cmd/patchwork` | Single entry point, wires CLI |
| `internal/cli` | Cobra commands: configure, detect, analyze, plan, apply, pr, run, serve |
| `internal/config` | Koanf-based config loading (defaults → YAML → keychain → env → flags) |
| `internal/detector` | Package detectors for Go, npm, pip, Cargo ecosystems |
| `internal/analyzer` | Changelog fetching, semver risk, LLM-powered breaking change analysis |
| `internal/planner` | Upgrade plan generation with recipe matching and risk ordering |
| `internal/codemod` | Code transformation engine (regex, go.mod, package.json, requirements.txt, Cargo.toml) |
| `internal/testrunner` | Isolated test execution for Go, npm, and Cargo |
| `internal/pr` | PR creation via GitHub/GitLab REST APIs |
| `internal/recipe` | Disk-based recipe store and episode recording |
| `internal/pipeline` | Full workflow orchestration |
| `internal/llm` | LLM provider abstraction (Anthropic, Mistral, Groq, Ollama) |
| `internal/keyring` | OS keychain integration for API keys and tokens |
| `pkg/domain` | Core domain types (Package, Upgrade, Recipe, Episode, etc.) |
| `pkg/git` | Git operations wrapper |
| `pkg/semver` | Semantic version utilities |

## Install

### curl (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/coder11125/patchwork/main/install.sh | bash
```

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/coder11125/patchwork/main/install.sh | INSTALL_DIR=$HOME/.local/bin bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/coder11125/patchwork/main/install.sh | bash -s -- v0.1.0
```

### From source

```bash
git clone https://github.com/coder11125/patchwork.git
cd patchwork
make build
./bin/patchwork --help
```

### With Go install

```bash
go install github.com/coder11125/patchwork/cmd/patchwork@latest
```

### Make targets

```bash
make build        # Build binary to bin/patchwork
make test         # Run tests with coverage
make lint         # Run gofmt check and go vet
make install      # Build and install to /usr/local/bin
make clean        # Remove build artifacts
make help         # Show all targets
```

## Configuration

Patchwork reads configuration in this precedence order: defaults → `~/.patchwork.yaml` → OS keychain → `PATCHWORK_*` env vars → CLI flags.

### Quick setup via CLI

```bash
# Interactively store API keys in your OS keychain
patchwork configure
```

This stores your LLM API key and Git token in the native OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager). These are never written to disk or exposed in plaintext.

### Environment variables

```bash
# LLM provider (required)
export PATCHWORK_LLM_PROVIDER=ollama        # anthropic, mistral, groq, ollama
export PATCHWORK_LLM_MODEL=llama3.2         # model name
export PATCHWORK_LLM_API_KEY=sk-...         # required for cloud providers
export PATCHWORK_LLM_BASE_URL=http://localhost:11434  # for Ollama

# Git/PR (optional, for pr command)
export PATCHWORK_GIT_PLATFORM=github        # github, gitlab
export PATCHWORK_GIT_TOKEN=ghp_...          # personal access token
export PATCHWORK_GIT_OWNER=coder11125
export PATCHWORK_GIT_REPO=myproject
export PATCHWORK_GIT_PR_BRANCH=main
export PATCHWORK_GIT_REMOTE=origin

# Behavior
export PATCHWORK_DRY_RUN=false
export PATCHWORK_SKIP_TESTS=false
export PATCHWORK_VERBOSE=false
export PATCHWORK_MAX_RETRIES=3

# Directories
export PATCHWORK_RECIPE_DIR=~/.patchwork/recipes
export PATCHWORK_EPISODE_DIR=~/.patchwork/episodes
export PATCHWORK_CACHE_DIR=~/.patchwork/cache
```

### Config file (`~/.patchwork.yaml`)

```yaml
llm_provider: anthropic
llm_model: claude-sonnet-4-20250514
llm_api_key: sk-ant-...

git_platform: github
git_token: ghp_...
git_owner: coder11125
git_repo: myproject
git_pr_branch: main

dry_run: false
skip_tests: false
verbose: true
max_retries: 3
```

API keys and tokens can be omitted from the config file and env vars if stored via `patchwork configure` — Patchwork falls back to the OS keychain automatically.

## Usage

### Configure credentials

```bash
# Store LLM API key and Git token in OS keychain
patchwork configure
```

### Detect outdated packages

```bash
# Scan current directory
patchwork detect

# Scan specific directory, output as JSON
patchwork detect --dir /path/to/repo --format json

# Filter by ecosystem
patchwork detect --ecosystem cargo --output results.json
```

### Analyze packages for breaking changes

```bash
# Analyze all detected packages
patchwork analyze

# Analyze specific package
patchwork analyze --package serde --ecosystem cargo
```

### Generate upgrade plan

```bash
# Generate plan from current directory
patchwork plan

# Output as JSON
patchwork plan --format json
```

### Apply upgrades

```bash
# Apply upgrades (generates fresh plan)
patchwork apply

# Apply from a saved plan file
patchwork apply --plan plan.json

# Dry run (show what would be done)
patchwork apply --dry-run
```

### Create pull requests

```bash
# Create PRs for all upgrades
patchwork pr

# Requires git_platform and git_token in config or keychain
```

### Run full pipeline

```bash
# Detect → analyze → plan → apply → PR
patchwork run

# Dry run (stops after plan)
patchwork run --dry-run

# Skip tests
patchwork run --skip-tests
```

### API Server

Patchwork can run as an HTTP API server for programmatic access. All CLI commands are available as REST endpoints.

```bash
# Start API server on default port 8080
patchwork serve

# Custom port and working directory
patchwork serve --addr :3000 --dir /path/to/repo
```

#### Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `POST` | `/detect` | Scan directory for outdated packages |
| `POST` | `/plan` | Generate upgrade plan |
| `POST` | `/run` | Execute full pipeline |

#### Examples

```bash
# Health check
curl http://localhost:8080/health

# Detect outdated packages
curl -X POST http://localhost:8080/detect \
  -H "Content-Type: application/json" \
  -d '{"dir": "/path/to/repo"}'

# Generate upgrade plan
curl -X POST http://localhost:8080/plan \
  -H "Content-Type: application/json" \
  -d '{"dir": "/path/to/repo"}'

# Run full pipeline
curl -X POST http://localhost:8080/run \
  -H "Content-Type: application/json" \
  -d '{"dir": "/path/to/repo"}'
```

## LLM Providers

| Provider | Env Var | Default Model | Notes |
|---|---|---|---|
| **Ollama** | `PATCHWORK_LLM_PROVIDER=ollama` | `llama3.2` | Local, no API key needed |
| **Anthropic** | `PATCHWORK_LLM_PROVIDER=anthropic` | `claude-sonnet-4-20250514` | Requires `PATCHWORK_LLM_API_KEY` |
| **Mistral** | `PATCHWORK_LLM_PROVIDER=mistral` | `mistral-large-latest` | Requires `PATCHWORK_LLM_API_KEY` |
| **Groq** | `PATCHWORK_LLM_PROVIDER=groq` | `llama-3.3-70b-versatile` | Requires `PATCHWORK_LLM_API_KEY` |

## Recipe System

Every upgrade attempt is recorded as an **Episode**. Successful episodes generate or update **Recipes** stored as JSON on disk. Future upgrades query the recipe store before falling back to LLM analysis.

Recipes live in `~/.patchwork/recipes/` as `{recipeID}.json`. Episodes live in `~/.patchwork/episodes/` as `{episodeID}.json`.

### Recipe format

```json
{
  "id": "rec-go-gin-v2",
  "name": "Gin v1 to v2 migration",
  "description": "Upgrade gin-gonic/gin from v1.x to v2.x",
  "ecosystem": "go",
  "package_name": "github.com/gin-gonic/gin",
  "from_version": "<2.0.0",
  "to_version": ">=2.0.0",
  "steps": [
    {
      "order": 1,
      "type": "codemod",
      "description": "Update import path",
      "pattern": "github.com/gin-gonic/gin",
      "replacement": "github.com/gin-gonic/gin/v2",
      "file_globs": ["**/*.go"]
    },
    {
      "order": 2,
      "type": "config_change",
      "description": "Update go.mod",
      "file_globs": ["go.mod"]
    }
  ],
  "success_rate": 0.95,
  "times_used": 12
}
```

## Design Principles

- **No Python runtime** — Pure Go binary
- **No OpenAI SDK** — Direct HTTP calls to each provider's REST API
- **No external SDKs** — GitHub/GitLab via raw REST, git via CLI wrapper, crates.io via raw REST
- **Explicit error handling** — No panics, no `must*` functions
- **Interfaces for testability** — Every external dependency is behind an interface
- **Local-first** — Ollama works fully offline; recipes bridge cloud and local modes
- **CLI-first** — Single binary, no daemon, no web UI

## License

MIT
