package git

import (
	"os"
	"os/exec"
	"path/filepath"
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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", dir}, args...)
	if output, err := exec.Command("git", cmdArgs...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
