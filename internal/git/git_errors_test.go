package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "YVCDB Test")
	runGit(t, dir, "config", "user.email", "test@example.invalid")
	return dir
}

func TestInitFailsWithNothingToCommit(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "YVCDB Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "YVCDB Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
	// empty dir: init and add succeed, commit fails (nothing staged)
	if err := Init(t.TempDir()); err == nil {
		t.Fatal("expected commit failure on empty directory")
	}
}

func TestCreateBranchFailsOutsideRepo(t *testing.T) {
	if err := CreateBranch(t.TempDir(), "feature"); err == nil {
		t.Fatal("expected error outside a repository")
	}
}

func TestBranchExistsFailsOutsideRepo(t *testing.T) {
	// git show-ref exits 128 outside a repo, which is not the "absent" exit code 1
	if _, err := BranchExists(t.TempDir(), "feature"); err == nil {
		t.Fatal("expected error outside a repository")
	}
}

func TestHasChangesFailsOutsideRepo(t *testing.T) {
	if _, err := HasChanges(t.TempDir()); err == nil {
		t.Fatal("expected error outside a repository")
	}
}

func TestCurrentBranchFailsOutsideRepo(t *testing.T) {
	if _, err := CurrentBranch(t.TempDir()); err == nil {
		t.Fatal("expected error outside a repository")
	}
}

func TestWorktreeAddFailsOnDuplicateBranch(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}
	// branch "main" already exists → worktree add -b main fails
	if err := WorktreeAdd(dir, filepath.Join(t.TempDir(), "wt"), "main"); err == nil {
		t.Fatal("expected error on duplicate branch")
	}
}

func TestWorktreeRemoveFailsOnUnknownWorktree(t *testing.T) {
	dir := initRepo(t)
	if err := WorktreeRemove(dir, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for unknown worktree")
	}
}

func TestRebaseConflictIsAborted(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}

	worktree := filepath.Join(t.TempDir(), "wt")
	if err := WorktreeAdd(dir, worktree, "feature"); err != nil {
		t.Fatal(err)
	}

	// diverge: same file edited on both branches
	if err := os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("main change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "main change"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "shared.txt"), []byte("feature change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(worktree, "feature change"); err != nil {
		t.Fatal(err)
	}

	err := Rebase(worktree, "main")
	if err == nil {
		t.Fatal("expected rebase conflict")
	}
	if !strings.Contains(err.Error(), "rebase onto main failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	// worktree must be usable after abort (no rebase in progress)
	branch, berr := CurrentBranch(worktree)
	if berr != nil || branch != "feature" {
		t.Fatalf("worktree not restored after abort: branch=%q err=%v", branch, berr)
	}
}

func TestMergeFastForwardFailsOnDivergence(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "shared.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}

	worktree := filepath.Join(t.TempDir(), "wt")
	if err := WorktreeAdd(dir, worktree, "feature"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "main advance"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(worktree, "feature advance"); err != nil {
		t.Fatal(err)
	}

	// diverged histories cannot fast-forward
	if err := MergeFastForward(dir, "feature"); err == nil {
		t.Fatal("expected fast-forward failure on divergence")
	}
}

func TestCommitAllFailsOutsideRepo(t *testing.T) {
	if err := CommitAll(t.TempDir(), "message"); err == nil {
		t.Fatal("expected error outside a repository")
	}
}

func TestCommitAllFailsWithNothingToCommit(t *testing.T) {
	dir := initRepo(t)
	if err := CommitAll(dir, "empty"); err == nil {
		t.Fatal("expected error when there is nothing to commit")
	}
}

func TestRebaseFailureWithNoRebaseInProgress(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CommitAll(dir, "base"); err != nil {
		t.Fatal(err)
	}
	// rebase onto a nonexistent branch fails before starting, so --abort also fails
	err := Rebase(dir, "does-not-exist")
	if err == nil {
		t.Fatal("expected rebase failure")
	}
	if !strings.Contains(err.Error(), "abort rebase") {
		t.Fatalf("expected joined abort error, got: %v", err)
	}
}
