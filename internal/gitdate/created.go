package gitdate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// FileFirstCommitTime returns the author date of the commit that added relPath (relative to repo root).
func FileFirstCommitTime(ctx context.Context, repoDir, relPath string) (time.Time, bool, error) {
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return time.Time{}, false, nil
	}

	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.hooksPath=/dev/null",
		"-c", "safe.directory="+repoDir,
		"log", "--diff-filter=A", "--follow",
		"--format=%aI", "-1", "--", relPath,
	)
	cmd.Dir = repoDir
	cmd.WaitDelay = 10 * time.Second

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return time.Time{}, false, fmt.Errorf("git log %s: %w: %s", relPath, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return time.Time{}, false, fmt.Errorf("git log %s: %w", relPath, err)
	}

	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, false, nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse git date %q: %w", s, err)
	}
	return t, true, nil
}

// BatchFileFirstCommitTimes returns the first-commit author date for each relative path.
func BatchFileFirstCommitTimes(ctx context.Context, repoDir string, relPaths []string) (map[string]time.Time, error) {
	result := make(map[string]time.Time, len(relPaths))
	if len(relPaths) == 0 {
		return result, nil
	}

	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return result, nil
	}

	args := []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "safe.directory=" + repoDir,
		"log", "--diff-filter=A", "--reverse",
		"--format=COMMIT\t%aI", "--name-only",
		"--",
	}
	args = append(args, relPaths...)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	cmd.WaitDelay = 10 * time.Second

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("git log batch: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git log batch: %w", err)
	}

	var current time.Time
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "COMMIT\t") {
			parsed, parseErr := time.Parse(time.RFC3339, strings.TrimPrefix(line, "COMMIT\t"))
			if parseErr != nil {
				return nil, fmt.Errorf("parse git date %q: %w", line, parseErr)
			}
			current = parsed
			continue
		}
		if _, seen := result[line]; !seen && !current.IsZero() {
			result[line] = current
		}
	}

	return result, nil
}
