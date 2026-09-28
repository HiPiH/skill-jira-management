# CLI Commands Reference

Complete reference for `jira-mgmt` CLI commands with all options and examples.

---

## Authentication & Configuration

### jira-mgmt auth / auth set-access

Authentication setup. Supports both interactive and non-interactive (agent-friendly) modes.

**Non-interactive (recommended for agents):**
```bash
# Cloud (Basic auth — email + API token)
jira-mgmt auth --instance https://mycompany.atlassian.net --email user@company.com --token API_TOKEN

# Server/DC (Bearer auth — Personal Access Token, no email)
jira-mgmt auth --instance https://jira.company.com --token PAT_TOKEN
```

Equivalent explicit command:
```bash
jira-mgmt auth set-access --instance https://mycompany.atlassian.net --email user@company.com --token API_TOKEN
```

**Interactive (prompts for input):**
```bash
jira-mgmt auth
```

**Flags:**
- `--instance URL` — Jira instance URL
- `--email EMAIL` — Account email (omit for Server/DC PAT auth)
- `--token TOKEN` — API token (Cloud) or PAT (Server/DC)

**Auth type detection:**
- Email provided → Basic auth (Cloud: `base64(email:token)`)
- No email → Bearer auth (Server/DC: `Bearer <token>`)

**Validation:**
Use `jira-mgmt auth whoami` as the canonical live auth probe. Add `--check=false` to inspect local resolution without hitting Jira.

**Storage:**
- Credential source names: `auto`, `keychain`, `env_or_file`
- `auto` defaults to system keychain on macOS and Windows
- Fallback auth file: `os.UserConfigDir()/jira-mgmt/auth.json`
- Config: `os.UserConfigDir()/jira-mgmt/config.yaml`

**What gets saved:**
- `instance_url` — Jira base URL
- `instance_type` — `cloud` or `server` (detected by `auth whoami` or first live client use)
- `auth_type` — `basic` or `bearer` (auto-determined)

**Notes:**
- Cloud: create API token at https://id.atlassian.com/manage-profile/security/api-tokens
- Server/DC: create PAT in Jira → Profile → Personal Access Tokens
- Corporate Jira behind SSO/OAuth proxy may require VPN for API access
- One-time setup per environment

### jira-mgmt auth whoami

Show resolved Jira auth state and optionally validate it against the live API.

```bash
jira-mgmt auth whoami
jira-mgmt auth whoami --check=false
```

### jira-mgmt auth resolve

Show which backend would provide credentials without printing the token.

```bash
jira-mgmt auth resolve
jira-mgmt auth resolve --source env_or_file
```

### jira-mgmt auth clean

Remove stored credentials for the selected instance.

```bash
jira-mgmt auth clean
jira-mgmt auth clean --instance https://jira.company.com
```

### jira-mgmt auth config-path

Print the global `auth.json` path used by the `env_or_file` backend.

```bash
jira-mgmt auth config-path
```

---

### jira-mgmt config set

Set configuration values.

**Syntax:**
```bash
jira-mgmt config set <key> <value>
```

**Keys:**
- `project` — default project key (e.g., `PROJ`, `ACME`)
- `board` — default board ID (numeric, e.g., `123`)
- `locale` — locale for content creation (e.g., `en-US`, `ru-RU`, `hy-AM`)

**Examples:**
```bash
# Set default project
jira-mgmt config set project ACME

# Set default board
jira-mgmt config set board 456

# Set locale to Russian
jira-mgmt config set locale ru-RU

# Set locale to English
jira-mgmt config set locale en-US
```

**Notes:**
- Config persists across sessions
- Can override per-command with flags
- Locale affects summary/description language for created issues

---

### jira-mgmt config show

Display current configuration.

**Syntax:**
```bash
jira-mgmt config show
```

**Example Output:**
```
Jira Configuration
------------------
Instance: https://acme.atlassian.net
Email: user@example.com
Project: ACME
Board: 456
Locale: en-US
```

---

## Query Commands (DSL)

### jira-mgmt q

Execute DSL queries. Token-efficient for reads.

**Syntax:**
```bash
jira-mgmt q '<query>'
```

**Query Types:**

#### 1. get(ISSUE-KEY){preset}

Get single issue.

**Presets:**
- `minimal` — key, status
- `default` — key, summary, status, assignee
- `overview` — + type, priority, parent
- `full` — all fields including subtasks (key, summary, status per subtask)

**Examples:**
```bash
# Minimal (default)
jira-mgmt q 'get(PROJ-123)'

# With full details
jira-mgmt q 'get(PROJ-123){full}'
```

**Output (minimal):**
```
PROJ-123: User Authentication
Status: In Progress
```

---

#### 2. list(filters){preset}

List multiple issues.

**Filters (comma-separated):**
- `sprint=current|ID` — filter by sprint
- `assignee=me|email` — filter by assignee
- `status=!done|in-progress|todo` — filter by status (use `!` for negation)
- `type=epic|story|task|bug|subtask` — filter by issue type

**Examples:**
```bash
# Current sprint
jira-mgmt q 'list(sprint=current){default}'

# My open issues
jira-mgmt q 'list(assignee=me,status=!done){default}'

# Epics only
jira-mgmt q 'list(type=epic){overview}'
```

---

#### 3. summary()

Board statistics and summary.

**Example:**
```bash
jira-mgmt q 'summary()'
```

---

#### 4. search(jql="..."){preset}

JQL search.

**Examples:**
```bash
# All epics
jira-mgmt q 'search(jql="project=PROJ AND issuetype=Epic"){overview}'

# Overdue issues
jira-mgmt q 'search(jql="due<now() AND statusCategory!=Done"){default}'
```

See `jql-patterns.md` for comprehensive JQL examples.

---

#### Batch Queries

Execute multiple queries with `;` separator.

**Examples:**
```bash
# Summary + current sprint
jira-mgmt q 'summary(); list(sprint=current){default}'
```

---

## Search Commands

### jira-mgmt grep

Search issues and comments with regex.

**Syntax:**
```bash
jira-mgmt grep <pattern> [flags]
```

**Flags:**
- `--scope <issues|comments|all>` — search scope (default: `all`)
- `-i` — case-insensitive
- `-C <num>` — context lines around match

**Examples:**
```bash
# Search everywhere
jira-mgmt grep "authentication"

# Case-insensitive, issues only
jira-mgmt grep -i "AUTH" --scope issues

# With context
jira-mgmt grep -C 2 "performance"
```

---

## Attachment Commands

### jira-mgmt attachment fetch

Download one attachment after verifying that it belongs to the named issue:

```bash
jira-mgmt attachment fetch PROJ-123 20001 --out .temp/PROJ-123/review.docx
```

The output must not already exist. `--max-bytes` defaults to 32 MiB and is
capped at 100 MiB. The client refuses cross-origin content URLs and redirects.

---

## Create Commands

### jira-mgmt create

Create new issue.

**Syntax:**
```bash
jira-mgmt create --type <type> --summary "..." [options]
```

**Required:**
- `--type <epic|story|task|subtask|bug>` — issue type
- `--summary "..."` — issue title
- `--project KEY` — project key (or use default from config)

**Optional:**
- `--description "..."` — issue description
- `--parent ISSUE-KEY` — parent issue (required for `subtask`)
- `--assignee email@example.com` — assignee email
- `--priority <highest|high|medium|low|lowest>` — priority level
- `--labels "label1,label2"` — comma-separated labels

**Examples:**

#### Epic
```bash
jira-mgmt create --type epic --summary "User Authentication System" --description "Implement OAuth2 and local authentication" --project PROJ
```

#### Story
```bash
jira-mgmt create --type story --summary "Login page UI" --parent PROJ-123 --project PROJ
```

#### Task
```bash
jira-mgmt create --type task --summary "Write unit tests" --project PROJ
```

#### Subtask
```bash
jira-mgmt create --type subtask --summary "Add password validation" --parent PROJ-125 --project PROJ
```

#### Bug
```bash
jira-mgmt create --type bug --summary "Login fails with special characters" --priority high --labels "security,authentication" --project PROJ
```

---

## Update Commands

### jira-mgmt update

Update issue fields (summary, description).

**Syntax:**
```bash
jira-mgmt update ISSUE-KEY [flags]
```

**Flags:**
- `--summary "..."` — new issue summary/title
- `--description "..."` — new issue description

At least one flag is required.

**Examples:**
```bash
# Update summary only
jira-mgmt update PROJ-123 --summary "New title"

# Update description only
jira-mgmt update PROJ-123 --description "New description text"

# Update both
jira-mgmt update PROJ-123 --summary "New title" --description "Updated description"
```

**Notes:**
- Description format is handled automatically: plain string for Server/DC, ADF for Cloud
- Useful for bulk cleanup of subtask titles/descriptions

---

### jira-mgmt transition

Move issue to different status.

**Syntax:**
```bash
jira-mgmt transition ISSUE-KEY --to "Status Name"
```

**Examples:**
```bash
# Start work
jira-mgmt transition PROJ-123 --to "In Progress"

# Mark done
jira-mgmt transition PROJ-123 --to "Done"
```

**Notes:**
- Status name must match workflow exactly (case-sensitive)
- Check available transitions: `jira-mgmt q 'get(ISSUE-KEY){full}'`

---

### jira-mgmt cancel

Cancel issue using the workflow `Cancel` transition with best-effort required field handling.

**Syntax:**
```bash
jira-mgmt cancel ISSUE-KEY [--reason "text"] [--cascade-subtasks]
```

**Examples:**
```bash
# Cancel issue with a reason comment
jira-mgmt cancel PROJ-123 --reason "прекращение работы с ICONIA"

# Cancel parent after direct subtasks
jira-mgmt cancel PROJ-123 --reason "прекращение работы с ICONIA" --cascade-subtasks
```

**Notes:**
- Automatically fills common cancel requirements such as `resolution=Отменено`
- Falls back to `Другое`/`Other` for workflow select fields when no exact reason option exists
- Returns Jira workflow errors verbatim when an external blocker still prevents cancel

---

### jira-mgmt comment

Add comment to issue.

**Syntax:**
```bash
jira-mgmt comment ISSUE-KEY --body "text"
```

**Examples:**
```bash
# Simple comment
jira-mgmt comment PROJ-123 --body "Started implementation"

# Multi-line comment
jira-mgmt comment PROJ-123 --body "Implementation notes:
- Added OAuth2 provider
- Configured redirect URIs
- Updated tests"
```

---

### jira-mgmt link

Link issues to each other.

```bash
jira-mgmt link [ISSUE-KEY...] --type NAME --to ISSUE-KEY [flags]
```

**Flags:**
- `--type NAME` — link type name, case-insensitive (required unless `--list-types`)
- `--to ISSUE-KEY` — the issue the named issues point at (required unless `--list-types`/`--show`)
- `--list-types` — list the link types configured on the instance
- `--show` — print the links already on the named issues, with their ids
- `--reverse` — swap the sides: `--to` becomes the subject of the sentence

**Reading direction.** The named issues read with the type's outward wording and
`--to` with its inward wording:

```bash
jira-mgmt link PROJ-1 --type Blocks --to PROJ-2     # "PROJ-1 blocks PROJ-2"
```

**Batch under one parent.** Several issues in one call all link to `--to`, so a
parent needs `--reverse` to stay the subject:

```bash
jira-mgmt link PROJ-2 PROJ-3 --type Parent --to PROJ-1 --reverse
# PROJ-1 parent of PROJ-2
# PROJ-1 parent of PROJ-3
```

An unknown type is refused once, before any link is created, and the refusal
lists the types the instance does have. A batch therefore cannot half-apply on a
typo.

**Sub-tasks.** Jira has no REST endpoint that converts a standard issue into a
sub-task; that is the UI's Move operation. Where a parent/child relationship is
wanted from the CLI, an issue link of a parent-shaped type is what is reachable.

---

### jira-mgmt unlink

Remove issue links by id.

```bash
jira-mgmt unlink <LINK-ID...>
```

Ids come from `jira-mgmt link <ISSUE-KEY> --show`. Several ids in one call are
removed in order.

```bash
jira-mgmt link PROJ-1 --show
# PROJ-1:
#   [5932251] parent of PROJ-2  Child task
jira-mgmt unlink 5932251
```

---

### jira-mgmt dod

Set Definition of Done criteria.

**Syntax:**
```bash
jira-mgmt dod ISSUE-KEY --set "criteria"
```

**Example:**
```bash
jira-mgmt dod PROJ-123 --set "Unit tests pass
E2E tests pass
Code reviewed
Documentation updated"
```

---

## Global Flags

All commands support:
- `--project KEY` — override default project
- `--board ID` — override default board
- `--format <json|text>` — output format (default: `text`)

**Examples:**
```bash
# JSON output for scripting
jira-mgmt q 'get(PROJ-123)' --format json

# Override project
jira-mgmt create --type task --summary "Test" --project TEMP
```

---

## Version

### jira-mgmt version

Display CLI version.

**Syntax:**
```bash
jira-mgmt version
```

---

**Document Version:** 1.0
**Last Updated:** 2026-02-12
