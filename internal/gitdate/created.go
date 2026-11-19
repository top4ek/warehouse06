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

// BatchFileFirstCommitTimes returns first-commit author dates, following renamed files.
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
		"log", "--diff-filter=AR", "--find-renames", "--reverse",
		"--format=COMMIT\t%aI", "--name-status", "-z",
		"--",
	}
	// Do not restrict the history to current paths: that would hide rename sources
	// and can make Git report a move as a new file instead.

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

	wanted := make(map[string]bool, len(relPaths))
	for _, path := range relPaths {
		wanted[path] = true
	}
	renamed := make(map[string]bool)
	var current time.Time
	remaining := string(out)
	for remaining != "" {
		token, rest, _ := strings.Cut(remaining, "\x00")
		remaining = rest
		token = strings.TrimPrefix(token, "\n")
		if token == "" {
			continue
		}
		if strings.HasPrefix(token, "COMMIT\t") {
			parsed, parseErr := time.Parse(time.RFC3339, strings.TrimPrefix(token, "COMMIT\t"))
			if parseErr != nil {
				return nil, fmt.Errorf("parse git date %q: %w", token, parseErr)
			}
			current = parsed
			continue
		}
		path, rest, ok := strings.Cut(remaining, "\x00")
		if !ok {
			return nil, fmt.Errorf("missing path in git history for status %q", token)
		}
		remaining = rest
		switch {
		case token == "A":
			if wanted[path] {
				if _, seen := result[path]; !seen {
					result[path] = current
				}
			}
		case strings.HasPrefix(token, "R"):
			dest, rest, ok := strings.Cut(remaining, "\x00")
			if !ok {
				return nil, fmt.Errorf("missing destination in git rename history")
			}
			remaining = rest
			if wanted[dest] {
				renamed[dest] = true
			}
		default:
			return nil, fmt.Errorf("unexpected git history status %q", token)
		}
	}
	// Most entries need only the batch query. Follow each moved entry separately
	// because Git's --follow supports just one path and handles rename chains.
	for path := range renamed {
		created, ok, err := FileFirstCommitTime(ctx, repoDir, path)
		if err != nil {
			return nil, err
		}
		if ok {
			result[path] = created
		}
	}

	return result, nil
}
