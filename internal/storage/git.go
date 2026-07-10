package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"warehouse06/internal/gitdate"
)

// GitSyncResult describes the outcome of a storage git sync.
type GitSyncResult struct {
	// Changed is true after a fresh clone or when pull updated HEAD.
	Changed bool
	Head    gitdate.Commit
	HasHead bool
}

func storageLogger(log *zap.Logger) *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}

func dirEntryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// prepareDirForClone ensures dir exists and wipes non-git contents when present.
func prepareDirForClone(dir string, log *zap.Logger) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read storage dir: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}

	log.Info("git sync: wiping non-git storage directory",
		zap.String("dir", dir),
		zap.Int("entry_count", len(entries)),
		zap.Strings("entries", dirEntryNames(entries)),
	)

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("clear storage dir %q: %w", dir, err)
		}
	}
	return nil
}

// SyncGit clones storageURL into dir when it is not a git repo, otherwise pulls.
// No-op when storageURL is empty (Changed is false).
func SyncGit(ctx context.Context, dir, storageURL string, log *zap.Logger) (GitSyncResult, error) {
	log = storageLogger(log)

	if storageURL == "" {
		return GitSyncResult{}, nil
	}

	log.Info("git sync: starting", zap.String("dir", dir), zap.String("storage_url", storageURL))

	if !isGitRepo(dir) {
		if err := prepareDirForClone(dir, log); err != nil {
			return GitSyncResult{}, err
		}
		if err := GitClone(ctx, dir, storageURL, log); err != nil {
			return GitSyncResult{}, err
		}
		head, ok, err := gitdate.HeadCommit(ctx, dir)
		if err != nil {
			log.Error("git sync: read HEAD failed", zap.String("dir", dir), zap.String("phase", "after_clone"), zap.Error(err))
			return GitSyncResult{}, fmt.Errorf("read HEAD after clone: %w", err)
		}
		log.Info("git sync: clone completed",
			zap.String("dir", dir),
			zap.Bool("has_head", ok),
			zap.String("head", head.Hash),
		)
		return GitSyncResult{Changed: true, Head: head, HasHead: ok}, nil
	}

	before, beforeOK, err := gitdate.HeadCommit(ctx, dir)
	if err != nil {
		log.Error("git sync: read HEAD failed", zap.String("dir", dir), zap.String("phase", "before_pull"), zap.Error(err))
		return GitSyncResult{}, fmt.Errorf("read HEAD before pull: %w", err)
	}

	if err := GitPull(ctx, dir, log); err != nil {
		return GitSyncResult{}, err
	}

	after, afterOK, err := gitdate.HeadCommit(ctx, dir)
	if err != nil {
		log.Error("git sync: read HEAD failed", zap.String("dir", dir), zap.String("phase", "after_pull"), zap.Error(err))
		return GitSyncResult{}, fmt.Errorf("read HEAD after pull: %w", err)
	}

	changed := false
	switch {
	case !beforeOK && afterOK:
		changed = true
	case beforeOK && afterOK && before.Hash != after.Hash:
		changed = true
	}

	log.Info("git sync: pull completed",
		zap.String("dir", dir),
		zap.Bool("changed", changed),
		zap.String("head_before", before.Hash),
		zap.String("head_after", after.Hash),
	)

	return GitSyncResult{Changed: changed, Head: after, HasHead: afterOK}, nil
}

// GitClone clones storageURL into dir. Caller must ensure dir is empty or missing.
func GitClone(ctx context.Context, dir, storageURL string, log *zap.Logger) error {
	log = storageLogger(log)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}

	log.Info("git sync: cloning", zap.String("dir", dir), zap.String("storage_url", storageURL))

	// "--" stops flag parsing so a crafted URL cannot inject git options.
	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.hooksPath=/dev/null",
		"clone", "--", storageURL, ".",
	)
	cmd.Dir = dir
	cmd.WaitDelay = 10 * time.Second
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("git sync: clone failed",
			zap.String("dir", dir),
			zap.String("storage_url", storageURL),
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return fmt.Errorf("git clone: %w: %s", err, string(output))
	}

	log.Info("git sync: clone succeeded", zap.String("dir", dir))
	return nil
}

// GitPull updates an existing git repository in dir.
func GitPull(ctx context.Context, dir string, log *zap.Logger) error {
	log = storageLogger(log)

	if !isGitRepo(dir) {
		return fmt.Errorf("storage dir %q is not a git repository", dir)
	}

	log.Info("git sync: pulling", zap.String("dir", dir))

	cmd := exec.CommandContext(ctx, "git",
		"-c", "core.hooksPath=/dev/null",
		"-c", "safe.directory="+dir,
		"pull",
	)
	cmd.Dir = dir
	cmd.WaitDelay = 10 * time.Second
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("git sync: pull failed",
			zap.String("dir", dir),
			zap.String("output", string(output)),
			zap.Error(err),
		)
		return fmt.Errorf("git pull: %w: %s", err, string(output))
	}

	log.Info("git sync: pull succeeded", zap.String("dir", dir))
	return nil
}
