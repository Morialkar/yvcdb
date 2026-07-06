package git

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryQueries(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "YVCDB Test")
	runGit(t, dir, "config", "user.email", "test@example.invalid")

	if !IsRepo(dir) {
		t.Fatal("expected a git repository")
	}
	hasChanges, err := HasChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if hasChanges {
		t.Fatal("new repository should be clean")
	}

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hasChanges, err = HasChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasChanges {
		t.Fatal("untracked file should be reported as a change")
	}

	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "initial")
	exists, err := BranchExists(dir, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("missing branch reported as existing")
	}
	if err := CreateBranch(dir, "feature"); err != nil {
		t.Fatal(err)
	}
	exists, err = BranchExists(dir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("created branch was not found")
	}
}

func TestInitCreatesSnapshot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_AUTHOR_NAME", "YVCDB Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "YVCDB Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if !IsRepo(dir) {
		t.Fatal("Init did not create a repository")
	}
	if output := runGitOutput(t, dir, "log", "-1", "--pretty=%s"); output != "chore: initial snapshot before YVCDB workflow" {
		t.Fatalf("unexpected snapshot message: %q", output)
	}
}

func TestWorktreeRebaseAndFastForward(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "YVCDB Test")
	runGit(t, dir, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}
	branch, err := CurrentBranch(dir)
	if err != nil || branch != "main" {
		t.Fatalf("branch=%q err=%v", branch, err)
	}

	worktree := filepath.Join(t.TempDir(), "feature-worktree")
	if err := WorktreeAdd(dir, worktree, "feature"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(worktree, "feature"); err != nil {
		t.Fatal(err)
	}
	if err := Rebase(worktree, "main"); err != nil {
		t.Fatal(err)
	}
	if err := WorktreeRemove(dir, worktree); err != nil {
		t.Fatal(err)
	}
	if err := MergeFastForward(dir, "feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Fatalf("merged file missing: %v", err)
	}
}

func TestEnsureInfoExcludeEntryIsIdempotent(t *testing.T) {
	dir := initRepo(t)
	entry := ".yvcdb_*"
	added, err := EnsureInfoExcludeEntry(dir, entry)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected entry to be added")
	}
	added, err = EnsureInfoExcludeEntry(dir, entry)
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("expected second call to be idempotent")
	}
	path := resolveGitPath(t, dir, runGitOutput(t, dir, "rev-parse", "--git-path", "info/exclude"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), entry); got != 1 {
		t.Fatalf("expected one entry, got %d in %q", got, data)
	}
}

func TestEnsureInfoExcludeEntryResolvesWorktreeCommonGitDir(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(t.TempDir(), "feature-worktree")
	if err := WorktreeAdd(dir, worktree, "feature"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = WorktreeRemove(dir, worktree)
	})
	entry := ".yvcdb_*"
	if _, err := EnsureInfoExcludeEntry(worktree, entry); err != nil {
		t.Fatal(err)
	}
	helperPath := resolveGitPath(t, worktree, runGitOutput(t, worktree, "rev-parse", "--git-path", "info/exclude"))
	data, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), entry) {
		t.Fatalf("expected entry in worktree exclude path %s: %q", helperPath, data)
	}
}

func resolveGitPath(t *testing.T, baseDir, path string) string {
	t.Helper()
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	if output, err := exec.Command("git", cmdArgs...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	output, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(bytes.TrimSpace(output))
}
