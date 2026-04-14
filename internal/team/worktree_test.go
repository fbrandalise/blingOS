package team

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupPersistedTaskWorktreesRemovesUniqueTrackedWorktrees(t *testing.T) {
	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, "broker-state.json")

	oldStatePath := brokerStatePath
	oldCleanup := cleanupTaskWorktree
	defer func() {
		brokerStatePath = oldStatePath
		cleanupTaskWorktree = oldCleanup
	}()

	brokerStatePath = func() string { return statePath }

	var calls []string
	cleanupTaskWorktree = func(path, branch string) error {
		calls = append(calls, path+"|"+branch)
		return nil
	}

	state := struct {
		Tasks []teamTask `json:"tasks"`
	}{
		Tasks: []teamTask{
			{ID: "task-1", WorktreePath: "/tmp/wuphf-task-1", WorktreeBranch: "wuphf-task-1"},
			{ID: "task-2", WorktreePath: "/tmp/wuphf-task-1", WorktreeBranch: "wuphf-task-1"},
			{ID: "task-3", WorktreePath: "/tmp/wuphf-task-3", WorktreeBranch: "wuphf-task-3"},
		},
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(statePath, raw, 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	if err := CleanupPersistedTaskWorktrees(); err != nil {
		t.Fatalf("cleanup persisted task worktrees: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 unique cleanup calls, got %d (%v)", len(calls), calls)
	}
}

func TestCleanupPersistedTaskWorktreesMissingStateIsNoOp(t *testing.T) {
	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, "broker-state.json")

	oldStatePath := brokerStatePath
	defer func() { brokerStatePath = oldStatePath }()
	brokerStatePath = func() string { return statePath }

	if err := CleanupPersistedTaskWorktrees(); err != nil {
		t.Fatalf("expected missing state cleanup to succeed, got %v", err)
	}
}

func TestDefaultPrepareTaskWorktreeOverlaysDirtyWorkspace(t *testing.T) {
	repoDir := t.TempDir()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "WUPHF Test")
	run("git", "config", "user.email", "wuphf@example.com")
	if err := os.WriteFile(filepath.Join(repoDir, "tracked.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write tracked baseline: %v", err)
	}
	run("git", "add", "tracked.txt")
	run("git", "commit", "-m", "base")

	if err := os.WriteFile(filepath.Join(repoDir, "tracked.txt"), []byte("modified\n"), 0o644); err != nil {
		t.Fatalf("write tracked modification: %v", err)
	}
	untrackedPath := filepath.Join(repoDir, "docs", "youtube-factory", "episode-launch-packets", "vid_01-inbox-operator.yaml")
	if err := os.MkdirAll(filepath.Dir(untrackedPath), 0o755); err != nil {
		t.Fatalf("mkdir untracked parent: %v", err)
	}
	if err := os.WriteFile(untrackedPath, []byte("id: vid_01\n"), 0o644); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}
	skippedCachePath := filepath.Join(repoDir, ".wuphf", "cache", "go-build", "ceo", "trim.txt")
	if err := os.MkdirAll(filepath.Dir(skippedCachePath), 0o755); err != nil {
		t.Fatalf("mkdir skipped cache parent: %v", err)
	}
	if err := os.WriteFile(skippedCachePath, []byte("skip me\n"), 0o644); err != nil {
		t.Fatalf("write skipped cache file: %v", err)
	}
	skippedPlaywrightPath := filepath.Join(repoDir, ".playwright-cli", "console.log")
	if err := os.MkdirAll(filepath.Dir(skippedPlaywrightPath), 0o755); err != nil {
		t.Fatalf("mkdir skipped playwright parent: %v", err)
	}
	if err := os.WriteFile(skippedPlaywrightPath, []byte("skip me too\n"), 0o644); err != nil {
		t.Fatalf("write skipped playwright file: %v", err)
	}

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}

	path, branch, err := defaultPrepareTaskWorktree("task-overlay")
	if err != nil {
		t.Fatalf("defaultPrepareTaskWorktree: %v", err)
	}
	defer func() {
		if err := defaultCleanupTaskWorktree(path, branch); err != nil {
			t.Fatalf("cleanup task worktree: %v", err)
		}
	}()

	trackedRaw, err := os.ReadFile(filepath.Join(path, "tracked.txt"))
	if err != nil {
		t.Fatalf("read tracked file from worktree: %v", err)
	}
	if got := string(trackedRaw); got != "modified\n" {
		t.Fatalf("expected tracked overlay in worktree, got %q", got)
	}

	untrackedRaw, err := os.ReadFile(filepath.Join(path, "docs", "youtube-factory", "episode-launch-packets", "vid_01-inbox-operator.yaml"))
	if err != nil {
		t.Fatalf("read untracked file from worktree: %v", err)
	}
	if got := string(untrackedRaw); got != "id: vid_01\n" {
		t.Fatalf("expected untracked overlay in worktree, got %q", got)
	}

	if _, err := os.Stat(filepath.Join(path, ".wuphf", "cache", "go-build", "ceo", "trim.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected generated cache file to be skipped, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(path, ".playwright-cli", "console.log")); !os.IsNotExist(err) {
		t.Fatalf("expected playwright log to be skipped, stat err=%v", err)
	}
}
