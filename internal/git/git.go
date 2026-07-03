package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// IsRepo reports whether dir belongs to a Git repository.
func IsRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// Init initializes a repository and creates a snapshot commit.
func Init(dir string) error {
	cmds := [][]string{
		{"git", "-C", dir, "init"},
		{"git", "-C", dir, "add", "."},
		{"git", "-C", dir, "commit", "-m", "chore: initial snapshot before YVCDB refactoring"},
	}
	for _, args := range cmds {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", strings.Join(args, " "), err, out)
		}
	}
	return nil
}

// CreateBranch creates and checks out branch in dir.
func CreateBranch(dir, branch string) error {
	out, err := exec.Command("git", "-C", dir, "checkout", "-b", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout -b %s: %w\n%s", branch, err, out)
	}
	return nil
}

// BranchExists reports whether a local branch exists.
func BranchExists(dir, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "show-ref", "--quiet", "refs/heads/"+branch)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check branch %s: %w", branch, err)
}

// HasChanges reports whether the working tree contains tracked or untracked changes.
func HasChanges(dir string) (bool, error) {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// CurrentBranch returns the currently checked-out branch.
func CurrentBranch(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("resolve current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeAdd creates a new branch from HEAD and checks it out in a separate worktree.
func WorktreeAdd(repoDir, worktreeDir, branch string) error {
	out, err := exec.Command("git", "-C", repoDir, "worktree", "add", "-b", branch, worktreeDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("worktree add %s: %w\n%s", branch, err, out)
	}
	return nil
}

// WorktreeRemove forcibly removes a Git worktree.
func WorktreeRemove(repoDir, worktreeDir string) error {
	out, err := exec.Command("git", "-C", repoDir, "worktree", "remove", "--force", worktreeDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("worktree remove: %w\n%s", err, out)
	}
	return nil
}

// Rebase rebases a worktree branch on the target branch. Conflicts are aborted,
// leaving both branches intact for a manual resolution.
func Rebase(worktreeDir, targetBranch string) error {
	out, err := exec.Command("git", "-C", worktreeDir, "rebase", targetBranch).CombinedOutput()
	if err != nil {
		rebaseErr := fmt.Errorf("rebase onto %s failed (worktree preserved at %s): %w\n%s", targetBranch, worktreeDir, err, out)
		if abortErr := exec.Command("git", "-C", worktreeDir, "rebase", "--abort").Run(); abortErr != nil {
			return errors.Join(rebaseErr, fmt.Errorf("abort rebase: %w", abortErr))
		}
		return rebaseErr
	}
	return nil
}

// MergeFastForward fast-forwards the current branch to branch.
func MergeFastForward(dir, branch string) error {
	out, err := exec.Command("git", "-C", dir, "merge", "--ff-only", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fast-forward merge %s failed:\n%s", branch, out)
	}
	return nil
}

// CommitAll stages all changes and creates a commit with message.
func CommitAll(dir, message string) error {
	add := exec.Command("git", "-C", dir, "add", "-A")
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}
	commit := exec.Command("git", "-C", dir, "commit", "-m", message)
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	return nil
}
