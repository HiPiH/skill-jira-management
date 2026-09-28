---
name: jira-management
description: Drive jira-mgmt CLI for Jira operations (Cloud & Server/DC, auto-detected). Translates natural language intent to CLI commands. Uses DSL for reads (token-efficient), CLI for writes. Handles multi-step workflows (create epic with stories, bulk transitions, sprint reviews).
triggers:
  - jira
  - джира
  - ticket
  - тикет
  - issue
  - задача
  - issues
  - задачи
  - epic
  - эпик
  - epics
  - эпики
  - story
  - стори
  - stories
  - сторя
  - board
  - борда
  - boards
  - борды
  - доска
  - sprint
  - спринт
  - sprints
  - спринты
  - create issue
  - создай задачу
  - create ticket
  - заведи тикет
  - move issue
  - двинь задачу
  - transition
  - переведи статус
  - show board
  - покажи борду
  - покажи доску
  - issue status
  - статус задачи
  - comment
  - комментарий
  - коммент
  - add comment
  - добавь коммент
  - search
  - поиск
  - find
  - найди
  - jql
  - dod
  - дод
  - definition of done
---

# Jira Management Skill

**Purpose:** Agent-agnostic skill (Claude Code / Codex CLI) for managing Jira (Cloud & Server/DC) via `jira-mgmt` CLI. Instance type is auto-detected during `auth whoami` and on first live client use.

**Tool:** `jira-mgmt` (installed via `./setup.sh` or `.\setup.ps1`)

---

## Quick Start

### 1. Setup

**Cloud (email + API token → Basic auth):**
```bash
jira-mgmt auth set-access --instance https://mycompany.atlassian.net --email user@company.com --token API_TOKEN
```

**Server/DC (Personal Access Token → Bearer auth):**
```bash
jira-mgmt auth set-access --instance https://jira.company.com --token PAT_TOKEN
```

Auth type is auto-determined: email provided → Basic (Cloud), no email → Bearer (Server/DC PAT).
Credential source names are stable across platforms: `auto`, `keychain`, `env_or_file`.

```bash
# Validate the stored credentials and detect instance type
jira-mgmt auth whoami

# Optional debug helpers
jira-mgmt auth resolve
jira-mgmt auth config-path
```

```bash
# Set active project/board/locale
jira-mgmt config set project YOUR-KEY
jira-mgmt config set board 123
jira-mgmt config set locale en
```

### 2. Basic Operations

**Read (use DSL for token efficiency):**
```bash
# Single issue
jira-mgmt q 'get(PROJ-123)'

# Targeted comments and attachment metadata
jira-mgmt q 'get(PROJ-123){comments attachments}'

# Bounded authenticated attachment download
jira-mgmt attachment fetch PROJ-123 20001 --out .temp/PROJ-123/review.docx

# List current sprint
jira-mgmt q 'list(sprint=current){default}'

# Search with JQL
jira-mgmt q 'search(jql="assignee=currentUser() AND statusCategory!=Done"){default}'
```

**Write (use CLI commands):**
```bash
# Create issue
jira-mgmt create --type story --summary "Login UI" --project PROJ

# Assign
jira-mgmt assign PROJ-123 --to me

# Transition
jira-mgmt transition PROJ-123 --to "In Progress"

# Cancel with reason
jira-mgmt cancel PROJ-123 --reason "прекращение работы с ICONIA"

# Comment
jira-mgmt comment PROJ-123 --body "Started work"

# Set DoD
jira-mgmt dod PROJ-123 --set "Tests pass\nCode reviewed"
```

---

## Commands Overview

### Authentication & Config
- `jira-mgmt auth set-access` — store credentials in the default backend for the platform:
  - Cloud: `jira-mgmt auth set-access --instance URL --email EMAIL --token API_TOKEN`
  - Server/DC: `jira-mgmt auth set-access --instance URL --token PAT`
- `jira-mgmt auth whoami` — canonical live auth probe; use `--check=false` for local inspection only
- `jira-mgmt auth resolve` — show where credentials resolve from without printing the token
- `jira-mgmt auth clean` — remove stored credentials for the selected instance
- `jira-mgmt auth config-path` — print the global `auth.json` path
- `jira-mgmt auth` — compatibility alias for `auth set-access`
- `jira-mgmt config set <key> <value>` — set project/board/locale
- `jira-mgmt config show` — display current config (includes instance type, auth type)

### Queries (DSL)
- `jira-mgmt q 'get(KEY){preset}'` — single issue
- `jira-mgmt q 'list(filters){preset}'` — multiple issues
- `jira-mgmt q 'summary()'` — board statistics
- `jira-mgmt q 'search(jql="..."){preset}'` — JQL search

**Presets:** `minimal`, `default`, `overview`, `full` (includes subtasks)

**List Filters:**
- `sprint=current|ID`
- `assignee=me|email`
- `status=!done|in-progress|todo`
- `type=epic|story|task|bug`

**Batch queries:** Use `;` separator

### Search
- `jira-mgmt grep <pattern>` — search issues/comments
  - Flags: `--scope issues|comments|all`, `-i`, `-C <num>`

### Create
- `jira-mgmt create --type <type> --summary "..." --project KEY`
  - Types: `epic`, `story`, `task`, `subtask`, `bug`
  - Optional: `--description`, `--parent`, `--assignee`, `--priority`, `--labels`

### Update
- `jira-mgmt update ISSUE-KEY --summary "..." --description "..."` — update issue fields
- `jira-mgmt assign ISSUE-KEY [ISSUE-KEY...] --to me|username|email` — set the assignee; `--clear` unassigns
- `jira-mgmt transition ISSUE-KEY --to "Status Name"` — move to status
  - `--list-fields` prints the transition's screen fields, marking the required ones and listing allowed values
  - `--field "Name=Value"` (repeatable) fills a screen field; accepts the display name or the field id
- `jira-mgmt cancel ISSUE-KEY --reason "..."` — cancel an issue with workflow-aware required fields
- `jira-mgmt comment ISSUE-KEY --body "text"` — add comment
- `jira-mgmt dod ISSUE-KEY --set "criteria"` — set Definition of Done
- `jira-mgmt link ISSUE-KEY [ISSUE-KEY...] --type NAME --to ISSUE-KEY` — link issues
  - `--list-types` prints the link types the instance offers, with the wording each side reads with
  - `--show` prints the links already on the named issues, with their ids
  - `--reverse` swaps the sides, which is what a batch under one parent needs
- `jira-mgmt unlink LINK-ID [LINK-ID...]` — remove links by id (ids come from `--show`)
- `jira-mgmt fields ISSUE-KEY` — every field the issue carries, custom ones included, with their ids (`--all` includes empty, `--field ID` prints one in full)
- `jira-mgmt update ISSUE-KEY --field "NAME=VALUE"` / `--clear-field NAME` — set or clear any field by name or id
- `jira-mgmt api METHOD /rest/...` — call a REST path directly, for plugin endpoints the typed commands do not cover

A custom field can only be written when it is on the issue's edit screen. Jira
refuses the rest with "cannot be set. It is not on the appropriate screen", and
that is a screen configuration matter, not something the CLI can route around.

### Global Flags
- `--project KEY` — override default project
- `--board ID` — override default board
- `--format json|text` — output format

---

## Agent Usage Patterns

### Natural Language → CLI

**User:** "Show me all epics in the current sprint"
```bash
jira-mgmt q 'list(sprint=current,type=epic){overview}'
```

**User:** "Create a story for login UI under epic PROJ-100"
```bash
jira-mgmt create --type story --summary "Login UI implementation" --parent PROJ-100 --project PROJ
```

**User:** "Find all issues mentioning 'performance'"
```bash
jira-mgmt grep -i "performance"
```

**User:** "Move PROJ-123 to done"
```bash
jira-mgmt transition PROJ-123 --to "Done"
```

**User:** "Put these tasks under the epic PROJ-100"
```bash
# Jira has no API for converting a standard issue into a sub-task -- that is the
# UI's Move operation. An issue link is the API-reachable relationship.
jira-mgmt link --list-types
jira-mgmt link PROJ-1 PROJ-2 PROJ-3 --type Parent --to PROJ-100 --reverse
jira-mgmt link PROJ-100 --show
```

**User:** "Assign these to me and put them in progress"
```bash
jira-mgmt assign PROJ-123 PROJ-124 --to me
# A workflow may require screen fields; ask what it wants before guessing:
jira-mgmt transition PROJ-123 --to "In Progress" --list-fields
jira-mgmt transition PROJ-123 --to "In Progress" --field "Вид деятельности=Разработка"
```

**User:** "Cancel PROJ-123 because work with the vendor stopped"
```bash
jira-mgmt cancel PROJ-123 --reason "прекращение работы с ICONIA"
```

### Read Operations: Use DSL

Prefer DSL over JQL for token efficiency.

**Good (DSL):**
```bash
jira-mgmt q 'list(sprint=current,assignee=me){default}'
```

**Use JQL only when:**
- DSL filters insufficient
- Complex multi-condition queries needed
- Historical/advanced queries required

### Write Operations: Use CLI

Always use explicit CLI commands for modifications:
```bash
jira-mgmt create --type story --summary "..." --project PROJ
jira-mgmt transition PROJ-123 --to "In Progress"
jira-mgmt cancel PROJ-123 --reason "..."
jira-mgmt comment PROJ-123 --body "..."
```

### Locale-Aware Content

Respect `locale` config for all content creation:
```bash
# Check locale
LOCALE=$(jira-mgmt config show | grep Locale | awk '{print $2}')

# Create in configured locale
if [[ "$LOCALE" == "ru-RU" ]]; then
  jira-mgmt create --type task --summary "Добавить тесты" --project PROJ
else
  jira-mgmt create --type task --summary "Add tests" --project PROJ
fi
```

---

## Cloud vs Server/DC Differences

The CLI auto-detects instance type and adapts. Key differences to be aware of:

| Aspect | Cloud | Server/DC |
|--------|-------|-----------|
| Auth | Basic (email + API token) | Bearer (Personal Access Token) |
| API version | v3 (`/rest/api/3/`) | v2 (`/rest/api/2/`) |
| Search pagination | Cursor-based (`nextPageToken`) | Offset-based (`startAt`) |
| Descriptions/comments | ADF (Atlassian Document Format) | ADF (Jira 8.x+) or wiki markup |
| Project listing | Paginated `/project/search` | `/project` returns full array |
| Status names | English by default | May be localized (e.g. Russian) |
| Assignee addressing | `accountId` | `name` (username) |

**Assignee addressing does not overlap.** Cloud rejects `name` and Server/DC
rejects `accountId`, so `assign` resolves the user and picks the field from the
detected instance type. A `--to` query that matches several users is refused
rather than guessed.

**Required transition fields.** A workflow screen can demand fields the issue
does not carry; Jira then refuses the transition and names an internal field id.
`--list-fields` shows what the transition wants, `--field` supplies it.

**Status names on Server/DC** may be in the instance's language. Use exact names as returned by Jira:
```bash
# Check actual status names for an issue
jira-mgmt q 'get(PROJ-123){full}'
# Transition using the exact name
jira-mgmt transition PROJ-123 --to "В работе"
```

---

## References

- `references/cli-commands.md` — Complete CLI command reference with all options
- `references/dsl-examples.md` — Comprehensive DSL query patterns and examples
- `references/workflows.md` — Multi-step workflow patterns (sprint review, daily standup, bulk ops)
- `references/jql-patterns.md` — JQL query patterns for advanced searches
- `references/troubleshooting.md` — Common issues and solutions (auth, JQL, transitions)
- `references/dev-notes.md` — Architecture notes for agents modifying the CLI codebase

---

**Skill Version:** 1.2
**Last Updated:** 2026-02-12
**Tool:** jira-mgmt CLI
