package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"warehouse06/internal/gitdate"
	"warehouse06/internal/parser"
	"warehouse06/internal/repository"
	"warehouse06/internal/storage"
)

type Syncer struct {
	log        *zap.Logger
	holder     *repository.Holder
	primaryDSN string
	parser     *parser.Parser
	storageDir string
	storageURL string
	interval   time.Duration
	status     *Status
	mu         sync.Mutex
	running    bool
	wg         sync.WaitGroup
}

func NewSyncer(
	storageDir, storageURL, primaryDSN string,
	interval time.Duration,
	holder *repository.Holder,
	parser *parser.Parser,
	status *Status,
	log *zap.Logger,
) *Syncer {
	return &Syncer{
		log:        log,
		holder:     holder,
		primaryDSN: primaryDSN,
		parser:     parser,
		storageDir: storageDir,
		storageURL: storageURL,
		interval:   interval,
		status:     status,
	}
}

func (s *Syncer) Run(ctx context.Context) {
	if err := s.Sync(ctx); err != nil {
		s.log.Error("sync failed", zap.Error(err))
	}

	if s.interval <= 0 {
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Sync(ctx); err != nil {
				s.log.Error("sync failed", zap.Error(err))
			}
		}
	}
}

func (s *Syncer) Sync(ctx context.Context) error {
	if !s.tryStart() {
		s.log.Info("sync already in progress, skipping")
		return nil
	}
	defer s.finish()

	s.status.SetSyncing(true)
	defer s.status.SetSyncing(false)

	s.log.Info("starting sync process")

	var head gitdate.Commit
	var hasHead bool

	if s.storageURL != "" {
		// Bound git network operations so a hung remote cannot stall the
		// long-lived sync loop indefinitely.
		gitCtx, cancelGit := context.WithTimeout(ctx, 10*time.Minute)
		gitResult, err := storage.SyncGit(gitCtx, s.storageDir, s.storageURL, s.log)
		cancelGit()
		if err != nil {
			s.log.Error("git sync failed", zap.Error(err))
			return fmt.Errorf("git sync: %w", err)
		}
		head, hasHead = gitResult.Head, gitResult.HasHead
		rebuild := gitResult.Changed || s.status.LastSyncedAt().IsZero()
		if !rebuild {
			s.log.Info("storage unchanged after git sync, skipping database rebuild")
			return nil
		}
	}

	s.log.Info("scanning directory", zap.String("dir", s.storageDir))
	entries, authors, err := s.parser.ScanDirectory()
	if err != nil {
		return fmt.Errorf("scan directory: %w", err)
	}

	readmePaths := make([]string, 0, len(entries))
	for _, e := range entries {
		readmePaths = append(readmePaths, filepath.Join(e.Path, "README.md"))
	}
	batchCtx, cancelBatch := context.WithTimeout(ctx, 5*time.Minute)
	createdAt, err := gitdate.BatchFileFirstCommitTimes(batchCtx, s.storageDir, readmePaths)
	cancelBatch()
	if err != nil {
		s.log.Warn("batch git created_at lookup failed", zap.Error(err))
	} else {
		for _, e := range entries {
			readmePath := filepath.Join(e.Path, "README.md")
			if t, ok := createdAt[readmePath]; ok {
				e.CreatedAt = t
			}
		}
	}

	stagingDSN := repository.PeerDSN(s.primaryDSN, s.holder.DSN())
	if !repository.IsInMemoryDSN(s.primaryDSN) {
		removeDBFiles(stagingDSN)
	}

	stagingRepo, err := repository.NewSQLiteRepository(stagingDSN, s.log)
	if err != nil {
		return fmt.Errorf("open staging database: %w", err)
	}

	s.log.Info("building staging database", zap.String("dsn", stagingDSN), zap.Int("entries", len(entries)), zap.Int("authors", len(authors)))
	if err := stagingRepo.SaveEntriesAndAuthors(ctx, entries, authors); err != nil {
		_ = stagingRepo.Close()
		if !repository.IsInMemoryDSN(stagingDSN) {
			removeDBFiles(stagingDSN)
		}
		return fmt.Errorf("save staging database: %w", err)
	}

	oldRepo, oldDSN := s.holder.Swap(stagingRepo, stagingDSN)
	if oldRepo != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.disposeRepo(oldRepo, oldDSN)
		}()
	}

	s.status.SetSuccess(time.Now(), head, hasHead)
	s.log.Info("sync completed successfully")
	return nil
}

func (s *Syncer) tryStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *Syncer) finish() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

// Wait blocks until background repo disposals complete. Call after Run has returned.
func (s *Syncer) Wait() {
	s.wg.Wait()
}

// removeDBFiles removes a SQLite database file together with its WAL/SHM
// sidecars, so a later rebuild cannot replay a stale write-ahead log.
func removeDBFiles(dsn string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(dsn + suffix)
	}
}

func (s *Syncer) disposeRepo(repo *repository.SQLiteRepository, dsn string) {
	if err := repo.Close(); err != nil {
		s.log.Warn("close old database", zap.String("dsn", dsn), zap.Error(err))
	}
	// Intentionally keep the previous on-disk DB file.
	// This allows fast warm-starts: on restart, the server can open the primary DSN
	// and serve existing data while a background rebuild creates the peer file.
}
