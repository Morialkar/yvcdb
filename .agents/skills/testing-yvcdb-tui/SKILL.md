---
name: testing-yvcdb-tui
description: Live end-to-end testing of the yvcdb TUI (OpenCode provider, resume-interrupted-phase). Use when verifying yvcdb phase runs, cancel/resume, or provider behavior against the real OpenCode CLI.
---

# Testing the yvcdb TUI live

yvcdb is a terminal UI (bubbletea). Test it live in a GUI terminal so the TUI is visible and recordable, and verify on-disk / git state from a **separate** shell (the exec tool), not inside the TUI.

## Environment / prerequisites
- Build the binary: `go build -o /tmp/yvcdb .` (Go lives at `/opt/go1.26.4/bin`).
- OpenCode CLI is installed at `~/.opencode/bin/opencode` (v1.17.13 when this was written). Put it on PATH: `export PATH=$HOME/.opencode/bin:$PATH`.
- Free models need no credentials. `opencode/north-mini-code-free` worked. If a specific free model stops working, list options with `opencode models` and pick another `*-free`, or ask the user to authenticate a paid model.
- GUI terminal: `konsole` is available on DISPLAY=:0 (KDE/plasma). Launch with `DISPLAY=:0 setsid konsole &`, then maximize with `DISPLAY=:0 wmctrl -r :ACTIVE: -b add,maximized_vert,maximized_horz`.
- The VM already has a working global git identity, so commits succeed. Do NOT run `git config` (it is blocked); if identity is ever missing use `GIT_AUTHOR_*` / `GIT_COMMITTER_*` env vars instead.

## Launching a run
`/tmp/yvcdb --provider opencode --mode refactor --model <free-model> <projectDir>`
- Use a throwaway git repo with at least one tracked source file and a baseline commit.
- `--mode refactor` (or `feature`/`debug`) sets the workflow; without `--mode`/`--phase`, launch auto-detects and (importantly) does NOT suppress a resume offer.
- Keys: `Enter` confirm model, `y` approve a phase gate, `s` skip, `q`/`Ctrl+C` quit. Ctrl+C during an active phase is how you interrupt for the resume test.

## Resume-interrupted-phase feature — what to verify
The marker file is `.yvcdb_resume.json` in the project root (git-excluded via `.git/info/exclude` pattern `.yvcdb_*`, alongside the `.yvcdb_<phase>_iter<n>_*.md` prompt file).
1. During an active run, both the marker and prompt file exist; marker JSON records `workflowMode`, `phaseIndex`, `phaseID`, `iteration`, `branchName`, `provider`, `model`, `sessionTimestamp`, `pid`, and file paths.
2. After Ctrl+C, both are **retained** (deletion only happens on clean finish: completion/failure/watchdog). `git status` must NOT list them; no phase commit is made.
3. Relaunch (no `--mode`/`--phase`) opens the resume/discard screen showing mode/phase/iteration/branch matching the marker.
4. `r` resumes on the **exact recorded branch** (check `git branch --show-current` equals the marker's `branchName` — a regression once forked a new-timestamp branch instead). The regenerated prompt file contains a `# Resume instructions` preamble referencing the mode's state artifact (e.g. `REFACTOR_STATE.md`).
5. Approving the resumed phase commits the report/changes on the resumed branch; the commit must contain no `.yvcdb_*` file.
6. `d` (discard) removes marker + prompt file, opens the normal model-select screen, and leaves git branches/log untouched.

Tips:
- Verify disk/git state with the exec tool while the TUI is running (e.g. `cat .yvcdb_resume.json`, `git status --short`, `ls .yvcdb_*.md`, `grep -n "Resume instructions" .yvcdb_*.md`).
- Read-only phases (e.g. refactor Phase 0 Diagnostic) produce no working-tree edits, so live resume proves wiring (branch/preamble/iteration) but not real edit-reconciliation. State this limitation in the report.
- Parallel-decline, missing-branch fallback, live-PID suppression, and malformed-marker cleanup are covered by Go unit tests (`internal/tui`, `cmd`); they are hard to force live.

## Devin Secrets Needed
- None for free-model testing. Optional: an OpenCode provider API key (e.g. `OPENCODE_API_KEY`) if the user wants a specific paid model exercised for `--model` passthrough.
