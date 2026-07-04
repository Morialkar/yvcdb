# YVCDB — Your Vibe Code Deserves Better

[![CI](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml/badge.svg)](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml)
[![Release](https://github.com/Morialkar/yvcdb/actions/workflows/release.yml/badge.svg)](https://github.com/Morialkar/yvcdb/releases)
[![Coverage](https://raw.githubusercontent.com/Morialkar/yvcdb/badges/coverage.svg)](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml)

*[Documentation en français](README.fr.md)*

YVCDB is an interactive CLI that applies the AFTER methodology through Claude Code or Codex CLI. It can refactor an existing codebase or guide a new project from specification through adversarial review, with a human approval gate after every phase.

The interface defaults to English and also supports French.

## The AFTER methodology

YVCDB is the reference implementation of the "Test Everything Rigorously" half of the AFTER methodology (Architect First, Test Everything Rigorously), my personal approach to AI-assisted development, applied to existing AI-generated codebases. The two halves map to the two ends of the workflow:

- **Architect First**: no code generation before specification. Detailed specs, quality standards files (`CLAUDE.md`), explicit constraints, and architectural decisions are made by the human before the AI generates anything.
- **Test Everything Rigorously**: nothing ships on trust. Generated code goes through tests (nominal, edge, and error cases), phased review with human validation gates, and a final human-approved checklist. The engineer's value shows up after generation, hence the name.

## Requirements

- Go 1.26 or newer
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) or Codex CLI
- Git, unless YVCDB is run with `--no-git`
- An authenticated session for the selected provider

Verify Go, Git, and the provider you intend to use:

```sh
go version
git --version

# One of these is required:
claude --version
codex --version
```

## Installation

### Prebuilt release

Download the archive for your operating system and architecture from the [latest GitHub release](https://github.com/Morialkar/yvcdb/releases/latest). Each archive contains both the `yvcdb` command and the backwards-compatible `tvcmm` alias.

- macOS and Linux: extract the `.tar.gz` archive and move `yvcdb` to a directory on your `PATH`, such as `/usr/local/bin`.
- Windows: extract the `.zip` archive and add its directory to your `PATH`.

Verify downloaded files against `checksums.txt` from the same release.

Confirm the installed version with `yvcdb --version`.

### Install with Go

Directly from the module proxy:

```sh
go install github.com/Morialkar/yvcdb@latest
```

Or from a local clone:

```sh
go install .
```

This installs the main `yvcdb` command in `$(go env GOPATH)/bin`.

To also install the backwards-compatible `tvcmm` alias:

```sh
go install ./...
```

Ensure the Go binary directory is on your `PATH`:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Configuration

Run the interactive configuration tool once:

```sh
yvcdb config
```

It configures:

- interface and response language: `en` or `fr`;
- AI CLI provider: `claude` or `codex`;
- the provider's default model, such as `sonnet` for Claude or `gpt-5.4` for Codex.

Defaults are English, Claude, and `sonnet`. On macOS, configuration is stored at:

```text
~/Library/Application Support/yvcdb/config.json
```

YVCDB reads the legacy `tvcmm` configuration if no YVCDB configuration exists.

The persistent file can also be edited directly:

```json
{
  "language": "en",
  "provider": "codex",
  "model": "gpt-5.4"
}
```

Codex runs non-interactively with JSONL output, ephemeral sessions, and a `workspace-write` sandbox. Claude continues to use its `stream-json` output mode.

YVCDB ships parallel English and French core prompts. The configured language selects both the interface strings and the embedded prompt set.

## Usage

Run YVCDB against the current directory:

```sh
yvcdb
```

Or specify a project:

```sh
yvcdb /path/to/project
```

Common overrides:

```sh
yvcdb --model opus --lang fr --max-turns 30 /path/to/project
yvcdb --provider codex --model gpt-5.4 /path/to/project
yvcdb --phase security /path/to/project
yvcdb --mode greenfield /path/to/empty-project
yvcdb --mode feature /path/to/project
yvcdb --mode debug /path/to/project
yvcdb --no-git /path/to/project
```

Available flags:

| Flag | Description |
| --- | --- |
| `--provider claude\|codex` | Override the configured AI CLI provider for this run |
| `--model <model>` | Override the configured provider model for this run |
| `--lang en\|fr` | Override the configured language for this run |
| `--max-turns <n>` | Set maximum turns for Claude; default: `20`. Codex CLI has no equivalent flag |
| `--mode auto\|refactor\|greenfield\|feature\|debug` | Select the workflow; `auto` uses greenfield only when the directory has no project files |
| `--phase <id>` | Start at a phase available in the selected workflow |
| `--no-git` | Disable automatic branches, commits, worktrees, and merges |

The selected model is always shown for confirmation before the pipeline starts.

## Managed AFTER workflows

With `--mode auto` (the default), an empty directory, including a Git-only directory, selects `greenfield`; a directory containing project files selects `refactor`. The mode can always be overridden explicitly.

The refactor workflow runs six sequential phases:

1. **Diagnostic** — inventories the codebase and identifies risks without modifying files.
2. **Safety net** — adds smoke tests and records the current state.
3. **Security** — addresses security findings and flags sensitive code for review.
4. **Structure** — extracts business logic and handles duplication.
5. **Readability** — improves naming, decomposition, and documentation.
6. **Devil's advocate** — performs a final adversarial review.

The greenfield workflow runs seven sequential phases:

1. **Specification** — produces `AFTER_SPEC.md`; no code is generated.
2. **Architecture** — produces `AFTER_ARCHITECTURE.md` and `AFTER_STANDARDS.md`; no code is generated.
3. **Planning** — produces the self-contained task plan in `AFTER_PLAN.md`; no code is generated.
4. **Foundation** — creates the approved scaffold, tooling, and test harness.
5. **Implementation** — implements approved tasks with production code and tests together.
6. **Verification** — proves requirements, coverage, errors, and security checks.
7. **Devil's advocate** — performs the final adversarial review without modifying files.

The feature workflow targets adding a feature to an existing codebase and updates `AFTER_SPEC.md`, `AFTER_ARCHITECTURE.md`, and `AFTER_PLAN.md` in place as it goes. It runs six sequential phases:

1. **Scoping** — reads the existing codebase and documents a spec delta in `AFTER_SPEC.md`; no product code is generated.
2. **Impact analysis** — updates `AFTER_ARCHITECTURE.md` in place to reflect touched modules, schema or migration changes, API changes, and risks.
3. **Planning** — updates `AFTER_PLAN.md` with small ordered tasks that keep the project testable after each step.
4. **Implementation** — delivers each approved task with production code and tests together.
5. **Verification** — validates the feature against the approved documents and runs the full existing test suite; any regression is a blocker.
6. **Devil's advocate** — performs the final adversarial review without modifying files.

The debug workflow fixes a bug in an existing codebase, starts from a required bug description, and proves the fix with a test that fails before and passes after. It runs six sequential phases:

1. **Report** — requires a bug description, reads the repository and `AFTER_*.md`, and writes `AFTER_BUG.md`; no product code is generated.
2. **Reproduction** — adds the smallest failing automated test and records it in `AFTER_BUG.md`.
3. **Diagnosis** — documents the root cause and proposed fix strategy in `AFTER_BUG.md`; no product code is generated.
4. **Fix** — applies the minimal fix targeting the root cause and adds regression tests together.
5. **Verification** — proves the fix, confirms the reproduction test fails without it, and runs the full existing suite; any regression is a blocker.
6. **Devil's advocate** — performs the final adversarial review without modifying files.

`AFTER_STANDARDS.md`, once created, is injected into every later agent session. All workflows require `ASSUMPTION`, `DECISION_REQUIRED`, and `REQUIRES_REVIEW` markers where applicable.

Each completed phase waits for a human decision:

| Key | Action |
| --- | --- |
| `y` or `o` | Approve and commit the result |
| `r` | Retry with the previous iteration context |
| `f` | Send precise free-form feedback to the agent and start another iteration |
| `s` | Skip the result |
| `q` | Quit and cancel active agent subprocesses |

After all phases, YVCDB presents a workflow-specific quality checklist. Failed criteria can be sent through an additional interactive correction loop.

## Git behavior

When Git integration is enabled, YVCDB:

- offers to initialize a repository when none exists;
- creates phase branches named `<mode>/<timestamp>/<phase>`;
- commits approved changes;

If branch creation, commit, rebase, or merge fails, YVCDB stops that path and reports the error instead of silently advancing. Conflicted rebases are aborted and their worktrees are preserved for manual resolution. Run with a clean working tree for predictable results.

## Logs

Raw provider stream events are written to:

```text
<project>/refactor-logs/<timestamp>_<phase>_iter<n>.md
```

The directory is ignored by this repository's `.gitignore`.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

CI runs these checks on every push and pull request, builds natively on Linux, macOS, and Windows, and rejects total coverage below 93%. Entry-point `main` packages are excluded from the measurement. The coverage badge is generated by CI itself and pushed to the `badges` branch — no external service involved.

To publish a release, push a semantic-version tag:

```sh
git tag v1.0.0
git push origin v1.0.0
```

The release workflow reruns CI, builds `amd64` and `arm64` archives for macOS, Linux, and Windows, publishes checksums and release notes, and creates GitHub artifact attestations.

The localized phase prompts are embedded from `cmd/prompts/en/` and `cmd/prompts/fr/`. Core orchestration lives in `internal/tui`, provider execution in `internal/runner`, and Git operations in `internal/git`.

## How this tool was built

YVCDB was developed with two AI assistants — Claude and Codex — following the AFTER methodology described above. The human designs the architecture and workflow up front (phases, Git strategy, approval loops), the assistants implement against that design, and every piece of behavior is then locked in with rigorous tests — including error paths, race conditions, and deadlocks, several of which were caught by the tests themselves. An article detailing the methodology is coming soon.

## License

YVCDB is licensed under the [MIT License](LICENSE).
