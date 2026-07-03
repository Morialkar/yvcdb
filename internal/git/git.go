package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func IsRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

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

func CreateBranch(dir, branch string) error {
	out, err := exec.Command("git", "-C", dir, "checkout", "-b", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkout -b %s: %w\n%s", branch, err, out)
	}
	return nil
}

func BranchExists(dir, branch string) bool {
	cmd := exec.Command("git", "-C", dir, "show-ref", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

func HasChanges(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	return err != nil || len(strings.TrimSpace(string(out))) > 0
}

func CurrentBranch(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
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

func WorktreeRemove(repoDir, worktreeDir string) error {
	out, err := exec.Command("git", "-C", repoDir, "worktree", "remove", "--force", worktreeDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("worktree remove: %w\n%s", err, out)
	}
	return nil
}

// Merge merges branch into the current branch. On conflict the merge is aborted.
func Merge(dir, branch string) error {
	out, err := exec.Command("git", "-C", dir, "merge", "--no-edit", branch).CombinedOutput()
	if err != nil {
		_ = exec.Command("git", "-C", dir, "merge", "--abort").Run()
		return fmt.Errorf("merge %s failed (possible conflicts) — manual merge required:\n%s", branch, out)
	}
	return nil
}

// Rebase rebases a worktree branch on the target branch. Conflicts are aborted,
// leaving both branches intact for a manual resolution.
func Rebase(worktreeDir, targetBranch string) error {
	out, err := exec.Command("git", "-C", worktreeDir, "rebase", targetBranch).CombinedOutput()
	if err != nil {
		_ = exec.Command("git", "-C", worktreeDir, "rebase", "--abort").Run()
		return fmt.Errorf("rebase onto %s failed (worktree preserved at %s):\n%s", targetBranch, worktreeDir, out)
	}
	return nil
}

func MergeFastForward(dir, branch string) error {
	out, err := exec.Command("git", "-C", dir, "merge", "--ff-only", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fast-forward merge %s failed:\n%s", branch, out)
	}
	return nil
}

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
