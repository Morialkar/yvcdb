# YVCDB — Your Vibe Code Deserves Better

YVCDB is an interactive CLI that orchestrates structured codebase refactoring through Claude Code or Codex CLI. It runs specialized review phases, isolates parallel changes in Git worktrees, and keeps a human approval step before changes are integrated.

The interface defaults to English and also supports French.

## Requirements

- Go 1.26 or newer
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) or Codex CLI
- Git, unless YVCDB is run with `--no-git`
- An authenticated session for the selected provider

Verify the dependencies:

```sh
go version
claude --version
codex --version
git --version
```

## Installation

From the repository:

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
| `q` | Quit |

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

If a rebase conflicts, it is aborted and the affected worktree is preserved for manual resolution. Run with a clean working tree for predictable results.

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

The phase prompts are embedded from `cmd/prompts/`. Core orchestration lives in `internal/tui`, provider execution in `internal/runner`, and Git operations in `internal/git`.

## Français

Configurez l'interface et les réponses de l'agent en français avec :

```sh
yvcdb config
```

Choisissez `fr` comme langue. Pour une seule exécution, utilisez `yvcdb --lang fr`.
