# YVCDB — Your Vibe Code Deserves Better

[![CI](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml/badge.svg)](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml)
[![Release](https://github.com/Morialkar/yvcdb/actions/workflows/release.yml/badge.svg)](https://github.com/Morialkar/yvcdb/releases)
[![Coverage](https://raw.githubusercontent.com/Morialkar/yvcdb/badges/coverage.svg)](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml)

*[Documentation en français](README.fr.md)*

YVCDB is an interactive CLI that orchestrates structured codebase refactoring through Claude Code or Codex CLI. It runs specialized review phases, isolates parallel changes in Git worktrees, and keeps a human approval step before changes are integrated.

The interface defaults to English and also supports French.

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
yvcdb --no-git /path/to/project
```

Available flags:

| Flag | Description |
| --- | --- |
| `--provider claude\|codex` | Override the configured AI CLI provider for this run |
| `--model <model>` | Override the configured provider model for this run |
| `--lang en\|fr` | Override the configured language for this run |
| `--max-turns <n>` | Set maximum turns for Claude; default: `20`. Codex CLI has no equivalent flag |
| `--phase <id>` | Start at `diagnostic`, `safety`, `security`, `structure`, `readability`, or `devil` |
| `--no-git` | Disable automatic branches, commits, worktrees, and merges |

The selected model is always shown for confirmation before the pipeline starts.

## Refactoring pipeline

YVCDB runs four stages:

1. **Diagnostic** — inventories the codebase and identifies risks without modifying files.
2. **Safety net** — adds smoke tests and records the current state.
3. **Parallel review** — runs security, structure, and readability phases in separate Git worktrees.
4. **Devil's advocate** — performs a final adversarial review.

Each completed phase waits for a human decision:

| Key | Action |
| --- | --- |
| `y` or `o` | Approve and commit the result |
| `r` | Retry with the previous iteration context |
| `f` | Send precise free-form feedback to the agent and start another iteration |
| `s` | Skip the result |
| `q` | Quit and cancel active agent subprocesses |

During the parallel stage, use `Tab` or `1`–`3` to switch between runs.

After all phases, YVCDB presents an eight-item quality checklist. Failed criteria can be sent through an additional interactive correction loop.

## Git behavior

When Git integration is enabled, YVCDB:

- offers to initialize a repository when none exists;
- creates phase branches named `refactor/<timestamp>/<phase>`;
- runs parallel phases in temporary worktrees under the system temporary directory;
- commits approved changes;
- rebases parallel branches sequentially onto the updated base branch;
- integrates them with fast-forward merges.

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

YVCDB was developed with two AI assistants — Claude and Codex — following what I call the **AFTER** methodology: **Architect First, Test Everything Rigorously**. The human designs the architecture and workflow up front (phases, Git strategy, approval loops), the assistants implement against that design, and every piece of behavior is then locked in with rigorous tests — including error paths, race conditions, and deadlocks, several of which were caught by the tests themselves. An article detailing the methodology is coming soon.

## License

YVCDB is licensed under the [MIT License](LICENSE).
