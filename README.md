# jira-management

CLI tool + agent skill for working with Jira Cloud and Server/DC projects.

## Architecture

```
Skill (SKILL.md) → CLI (jira-mgmt) → Library (internal/) → Jira REST API
```

**Monorepo** with three components:

| Component | Location | Purpose |
|-----------|----------|---------|
| Go library | `internal/jira/` | Jira Cloud + Server/DC REST client |
| Config | `internal/config/` | Cross-platform auth resolution + YAML config |
| CLI tool | `cmd/jira-mgmt/` | Agents-facing CLI with DSL query layer |
| Skill | `agents/skills/jira-management/` | Claude Code / Codex skill |

## Setup

```bash
# macOS / Linux shells
./setup.sh

# Windows PowerShell
.\setup.ps1

# Verify installed binary
jira-mgmt version

# Store credentials and validate them
jira-mgmt auth set-access --instance https://mycompany.atlassian.net --email user@company.com --token API_TOKEN
jira-mgmt auth whoami

# Set active project
jira-mgmt config set project YOUR-KEY
```

### Deinit

```bash
# Remove installed artifacts (keep config)
./scripts/deinit.sh

# Remove installed artifacts + config
./scripts/deinit.sh --purge
```

## CLI Reference

### Auth & Config

```bash
jira-mgmt auth set-access --instance URL --email EMAIL --token TOKEN
jira-mgmt auth whoami                  # Canonical live auth probe
jira-mgmt auth resolve                 # Show where credentials resolve from
jira-mgmt auth clean                   # Remove stored credentials
jira-mgmt auth config-path             # Print global auth.json path
jira-mgmt auth                         # Compatibility alias for set-access
jira-mgmt config set project KEY       # Set active project
jira-mgmt config set board 42          # Set active board
jira-mgmt config set locale en         # Set locale (en/ru)
jira-mgmt config show                  # Show current config
```

### DSL Queries (agents-facing reads)

```bash
jira-mgmt q 'get(PROJ-123) { default }'                           # Single issue
jira-mgmt q 'get(PROJ-123) { comments attachments }'              # Targeted discussion and files
jira-mgmt q 'list(project=PROJ, type=epic) { overview }'          # List epics
jira-mgmt q 'summary()'                                            # Project overview
jira-mgmt q 'search(jql="assignee=currentUser()") { minimal }'    # JQL search
jira-mgmt q 'get(A-1) { status }; get(A-2) { status }'            # Batch queries
```

Field presets: `minimal` (key, status), `default` (+summary, assignee), `overview` (+type, priority, parent), `full` (all fields, including comments and attachment metadata for a targeted issue read).

### Grep (text search)

```bash
jira-mgmt grep "pattern"                    # Search all
jira-mgmt grep "bug" --scope issues -i      # Search issues, case-insensitive
jira-mgmt grep "deploy" --scope comments    # Search comments
```

### Attachments

```bash
jira-mgmt q 'get(PROJ-123){attachments}'
jira-mgmt attachment fetch PROJ-123 20001 --out .temp/PROJ-123/review.docx
```

Attachment fetch first verifies that the attachment belongs to the named issue,
requires a same-origin Jira content URL, refuses redirects, caps the response,
and writes a new mode-`0600` file atomically.

### Write Operations

```bash
jira-mgmt create --type epic --summary "New feature" --project KEY
jira-mgmt create --type story --summary "Auth flow" --parent PROJ-10
jira-mgmt create --type task --summary "Implement login" --parent PROJ-11
jira-mgmt transition PROJ-123 --to "In Progress"
jira-mgmt cancel PROJ-123 --reason "прекращение работы с ICONIA"
jira-mgmt comment PROJ-123 --body "Done, ready for review"
jira-mgmt dod PROJ-123 --set "- Unit tests\n- Code review\n- Docs updated"
jira-mgmt link PROJ-2 PROJ-3 --type Parent --to PROJ-1 --reverse
jira-mgmt unlink 5932251
```

### Global Flags

```bash
--project KEY    # Override active project
--board ID       # Override active board
--format json    # Output format (json/text)
```

## Testing

```bash
go test ./... -v                                   # Run all tests
go test ./internal/jira/... -v                     # Library tests only
go test ./internal/config/... -v                   # Config tests only
go test ./cmd/jira-mgmt/... -v                     # CLI tests only
go test ./... -cover                               # Coverage report
UPDATE_SNAPSHOTS=1 go test ./... -v                # Update golden files
```

## Project Structure

```
.
├── cmd/jira-mgmt/          # CLI entry point + commands
├── internal/
│   ├── jira/               # Jira Cloud API client library
│   ├── config/             # Auth resolution + config (YAML/JSON)
│   ├── query/              # DSL parser & executor
│   ├── fields/             # Field selection & projection
│   └── search/             # Scoped grep
├── setup.sh                # Root wrapper for scripts/setup.sh
├── setup.ps1               # Root wrapper for scripts/setup.ps1
├── agents/skills/jira-management/   # Skill
│   ├── SKILL.md
│   └── references/
│       ├── cli-commands.md
│       ├── dsl-examples.md
│       ├── jql-patterns.md
│       └── workflows.md
├── scripts/
│   ├── setup.sh            # macOS/Linux build + install
│   ├── setup.ps1           # Windows build + install
│   └── deinit.sh           # Uninstall
├── .spec/                  # Project specifications
├── .task-board/            # Task board (project management)
├── .research/              # Research artifacts
└── .planning/              # Planning artifacts
```

## Tools

| Tool | Purpose | How to run |
|------|---------|-----------|
| `jira-mgmt` | CLI for Jira Cloud and Server/DC | `jira-mgmt --help` |
| `go test` | Run tests | `go test ./... -v` |
| `task-board` | Project management board | `task-board summary` |
| `setup.sh` / `setup.ps1` | Build and install runtime artifacts | `./setup.sh` / `.\setup.ps1` |
| `scripts/deinit.sh` | Uninstall | `./scripts/deinit.sh [--purge]` |

## Config Location

- **Config:** `os.UserConfigDir()/jira-mgmt/config.yaml`
  - macOS: `~/Library/Application Support/jira-mgmt/config.yaml`
  - Windows: `%AppData%\jira-mgmt\config.yaml`
- **Credentials:** `auto | keychain | env_or_file`
  - `auto` defaults to system keychain on macOS and Windows
  - fallback file path: `os.UserConfigDir()/jira-mgmt/auth.json`
- **Install state:** `os.UserConfigDir()/jira-mgmt/install.json`
- **Binary:** `~/.local/bin/jira-mgmt` (standalone installed copy)
- **Installed skill artifact:** `~/.agents/skills/jira-management` (degitized runtime copy)
- **Skill entrypoints:** `~/.claude/skills/jira-management` + `~/.codex/skills/jira-management` (symlinks to the installed artifact)

<!-- relux-ecosystem:start -->

## About Relux Works

This project is part of the open-source ecosystem of
[Relux Works](https://relux.works), an AI-native software development studio.
We build fixed-price MVPs, rescue vibe-coded apps, run local AI inference, and
train teams to work with coding agents. Much of the infrastructure behind that
work is open source.

- Full catalog: [relux.works/en/open-source](https://relux.works/en/open-source/)
- Agentic enablement: [agent harnesses & team training](https://relux.works/en/agentic-enablement/)
- Hire us the agent-native way: point your assistant at `https://api.relux.works/mcp`
- Contact: ivan@relux.works

<!-- relux-ecosystem:end -->

## License

Apache 2.0