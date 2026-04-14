package team

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var prepareTaskWorktree = defaultPrepareTaskWorktree
var cleanupTaskWorktree = defaultCleanupTaskWorktree

var overlaySourceWorkspaceSkipExact = map[string]struct{}{
	".playwright-cli": {},
	".playwright-mcp": {},
}

var overlaySourceWorkspaceSkipPrefixes = []string{
	".playwright-cli/",
	".playwright-mcp/",
	".wuphf/cache/",
}

func defaultPrepareTaskWorktree(taskID string) (string, string, error) {
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return "", "", err
	}

	branch := worktreeBranchName(taskID)
	path := filepath.Join(os.TempDir(), "wuphf-task-"+sanitizeWorktreeToken(taskID))
	finish := func(path, branch string) (string, string, error) {
		if err := overlaySourceWorkspace(repoRoot, path); err != nil {
			_ = defaultCleanupTaskWorktree(path, branch)
			return "", "", fmt.Errorf("overlay source workspace: %w", err)
		}
		return path, branch, nil
	}
	firstErr := runGit(repoRoot, "worktree", "add", "-b", branch, path, "HEAD")
	if firstErr == nil {
		return finish(path, branch)
	}
	_ = defaultCleanupTaskWorktree(path, branch)
	if err := runGit(repoRoot, "worktree", "add", "-b", branch, path, "HEAD"); err == nil {
		return finish(path, branch)
	}
	if err := runGit(repoRoot, "worktree", "add", path, branch); err == nil {
		return finish(path, branch)
	}

	return "", "", fmt.Errorf("create git worktree for %s: %w", taskID, firstErr)
}

func defaultCleanupTaskWorktree(path, branch string) error {
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return err
	}

	var failures []string
	if strings.TrimSpace(path) != "" {
		if err := runGit(repoRoot, "worktree", "remove", "--force", path); err != nil {
			if _, statErr := os.Stat(path); statErr == nil {
				if worktreePathLooksSafe(path) {
					if rmErr := os.RemoveAll(path); rmErr != nil {
						failures = append(failures, rmErr.Error())
					}
				} else {
					failures = append(failures, err.Error())
				}
			}
		}
	}
	if strings.TrimSpace(branch) != "" {
		if gitRefExists(repoRoot, "refs/heads/"+branch) {
			if err := runGit(repoRoot, "branch", "-D", branch); err != nil {
				failures = append(failures, err.Error())
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func gitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("resolve repo root: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runGitOutput(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func overlaySourceWorkspace(repoRoot, worktreePath string) error {
	changed, err := runGitOutput(repoRoot, "diff", "--name-only", "-z", "HEAD", "--")
	if err != nil {
		return err
	}
	untracked, err := runGitOutput(repoRoot, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}

	seen := map[string]struct{}{}
	for _, raw := range append(bytes.Split(changed, []byte{0}), bytes.Split(untracked, []byte{0})...) {
		rel := strings.TrimSpace(string(raw))
		if rel == "" {
			continue
		}
		if !shouldOverlaySourceWorkspacePath(rel) {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		seen[rel] = struct{}{}
		src := filepath.Join(repoRoot, filepath.FromSlash(rel))
		dst := filepath.Join(worktreePath, filepath.FromSlash(rel))
		info, statErr := os.Lstat(src)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
					return err
				}
				continue
			}
			return statErr
		}
		if err := copyWorkspacePath(src, dst, info); err != nil {
			return err
		}
	}
	return nil
}

func shouldOverlaySourceWorkspacePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	if _, skip := overlaySourceWorkspaceSkipExact[rel]; skip {
		return false
	}
	for _, prefix := range overlaySourceWorkspaceSkipPrefixes {
		if strings.HasPrefix(rel, prefix) {
			return false
		}
	}
	return true
}

func copyWorkspacePath(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		_ = os.RemoveAll(dst)
		return os.Symlink(target, dst)
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
		return err
	}
	return nil
}

func gitRefExists(dir, ref string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", ref)
	cmd.Dir = dir
	return cmd.Run() == nil
}

func worktreeBranchName(taskID string) string {
	return "wuphf-" + sanitizeWorktreeToken(taskID)
}

func CleanupPersistedTaskWorktrees() error {
	path := brokerStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state struct {
		Tasks []teamTask `json:"tasks"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	seen := make(map[string]struct{})
	var firstErr error
	for _, task := range state.Tasks {
		worktreePath := strings.TrimSpace(task.WorktreePath)
		worktreeBranch := strings.TrimSpace(task.WorktreeBranch)
		if worktreePath == "" && worktreeBranch == "" {
			continue
		}
		key := worktreePath + "\x00" + worktreeBranch
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := cleanupTaskWorktree(worktreePath, worktreeBranch); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func worktreePathLooksSafe(path string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return false
	}
	tempRoot := filepath.Clean(os.TempDir())
	if tempRoot == "." || tempRoot == "" {
		return false
	}
	if !strings.HasPrefix(path, tempRoot+string(os.PathSeparator)) && path != tempRoot {
		return false
	}
	return strings.Contains(filepath.Base(path), "wuphf-task-")
}

func sanitizeWorktreeToken(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "task"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(strings.ReplaceAll(b.String(), "--", "-"), "-")
}
