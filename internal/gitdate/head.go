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

// Commit describes the current HEAD of a git repository.
type Commit struct {
	Hash        string
	CommittedAt time.Time
	Subject     string
}

// HeadCommit returns HEAD hash, author date, and subject for repoDir.
func HeadCommit(ctx context.Context, repoDir string) (Commit, bool, error) {
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return Commit{}, false, nil
	}

	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.hooksPath=/dev/null",
		"-c", "safe.directory="+repoDir,
		"log", "-1", "--format=%H%x00%aI%x00%s",
	)
	cmd.Dir = repoDir
	cmd.WaitDelay = 10 * time.Second

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return Commit{}, false, fmt.Errorf("git log HEAD: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return Commit{}, false, fmt.Errorf("git log HEAD: %w", err)
	}

	parts := strings.SplitN(strings.TrimSuffix(string(out), "\n"), "\x00", 3)
	if len(parts) < 2 || parts[0] == "" {
		return Commit{}, false, nil
	}

	t, err := time.Parse(time.RFC3339, parts[1])
	if err != nil {
		return Commit{}, false, fmt.Errorf("parse git HEAD date %q: %w", parts[1], err)
	}

	subject := ""
	if len(parts) > 2 {
		subject = parts[2]
	}

	return Commit{
		Hash:        parts[0],
		CommittedAt: t,
		Subject:     subject,
	}, true, nil
}
