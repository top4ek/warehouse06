package sync

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"warehouse06/internal/gitdate"
	"warehouse06/internal/parser"
	"warehouse06/internal/rebus"
	"warehouse06/internal/repository"
	"warehouse06/internal/storage"
)

type Syncer struct {
	log              *zap.Logger
	holder           *repository.Holder
	primaryDSN       string
	parser           *parser.Parser
	storageDir       string
	storageURL       string
	interval         time.Duration
	status           *Status
	rebusExportPath  string
	sqliteExportPath string
	mu               sync.Mutex
	running          bool
	wg               sync.WaitGroup
}

func NewSyncer(
	storageDir, storageURL, primaryDSN string,
	interval time.Duration,
	holder *repository.Holder,
	parser *parser.Parser,
	status *Status,
	rebusExportPath string,
	sqliteExportPath string,
	log *zap.Logger,
) *Syncer {
	return &Syncer{
		log:              log,
		holder:           holder,
		primaryDSN:       primaryDSN,
		parser:           parser,
		storageDir:       storageDir,
		storageURL:       storageURL,
		interval:         interval,
		status:           status,
		rebusExportPath:  rebusExportPath,
		sqliteExportPath: sqliteExportPath,
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
			s.status.SetSuccess(time.Now(), head, hasHead)
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

	if err := s.rebuildSQLiteExport(ctx, stagingRepo); err != nil {
		// Same policy as the REBUS export below: a derived artifact must
		// never fail the catalog rebuild. Done here rather than after the
		// swap because the staging database serves no traffic yet, so
		// copying it cannot contend with API handlers.
		s.log.Error("sqlite export rebuild failed", zap.Error(err))
	}

	oldRepo, oldDSN := s.holder.Swap(stagingRepo, stagingDSN)
	if oldRepo != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.disposeRepo(oldRepo, oldDSN)
		}()
	}

	if err := s.rebuildRebusExport(ctx); err != nil {
		// The catalog rebuild itself succeeded; the REBUS export is a
		// secondary derived artifact. Do not fail Sync() over it - the
		// export endpoint gracefully serves the previous file (or a 503
		// if none exists yet) when this fails.
		s.log.Error("rebus export rebuild failed", zap.Error(err))
	}

	s.status.SetSuccess(time.Now(), head, hasHead)
	s.log.Info("sync completed successfully")
	return nil
}

// rebuildRebusExport regenerates the REBUS-compatible export from the
// freshly-swapped active repository and atomically replaces the static file
// the HTTP endpoint serves. Writing to a temp path and renaming over the
// final path (rather than truncating it in place) guarantees a concurrent
// reader never observes a partially-written file: POSIX rename is atomic,
// and an already-open reader keeps referencing the old inode's complete
// data.
func (s *Syncer) rebuildRebusExport(ctx context.Context) error {
	if s.rebusExportPath == "" {
		return nil
	}

	repo := s.holder.Get()
	entries, err := repo.ListAllEntries(ctx)
	if err != nil {
		return fmt.Errorf("list entries for rebus export: %w", err)
	}
	authors, err := repo.ListAllAuthors(ctx)
	if err != nil {
		return fmt.Errorf("list authors for rebus export: %w", err)
	}
	tags, err := repo.ListTags(ctx)
	if err != nil {
		return fmt.Errorf("list tags for rebus export: %w", err)
	}

	zipData, warnings, err := rebus.Export(rebus.ExportInput{Entries: entries, Authors: authors, Tags: tags})
	if err != nil {
		return fmt.Errorf("build rebus export: %w", err)
	}
	for _, w := range warnings {
		s.log.Warn("rebus export warning", zap.String("detail", w))
	}

	tmpPath := s.rebusExportPath + ".tmp"
	if err := os.WriteFile(tmpPath, zipData, 0o644); err != nil {
		return fmt.Errorf("write rebus export temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.rebusExportPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename rebus export into place: %w", err)
	}

	s.log.Info("rebus export rebuilt", zap.String("path", s.rebusExportPath), zap.Int("size", len(zipData)))
	return nil
}

// sqliteExportMemberName is the name the database carries inside the export zip.
const sqliteExportMemberName = "warehouse06.sqlite"

// rebuildSQLiteExport snapshots the freshly-built staging database into a
// standalone SQLite file, zips it, and atomically replaces the static file the
// HTTP endpoint serves (same rename rationale as rebuildRebusExport).
//
// The snapshot is a copy taken with VACUUM INTO, so the catalog itself keeps
// running unchanged from whichever DSN mode is configured - :memory: or file.
func (s *Syncer) rebuildSQLiteExport(ctx context.Context, repo *repository.SQLiteRepository) error {
	if s.sqliteExportPath == "" {
		return nil
	}

	snapshotPath := s.sqliteExportPath + ".db.tmp"
	if err := repo.SnapshotTo(ctx, snapshotPath); err != nil {
		return fmt.Errorf("snapshot database: %w", err)
	}
	defer func() { _ = os.Remove(snapshotPath) }()

	snapshot, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("read database snapshot: %w", err)
	}

	zipData, err := zipSQLiteSnapshot(snapshot)
	if err != nil {
		return err
	}

	tmpPath := s.sqliteExportPath + ".tmp"
	if err := os.WriteFile(tmpPath, zipData, 0o644); err != nil {
		return fmt.Errorf("write sqlite export temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.sqliteExportPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename sqlite export into place: %w", err)
	}

	s.log.Info("sqlite export rebuilt", zap.String("path", s.sqliteExportPath), zap.Int("size", len(zipData)))
	return nil
}

func zipSQLiteSnapshot(snapshot []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(sqliteExportMemberName)
	if err != nil {
		return nil, fmt.Errorf("add %s to zip: %w", sqliteExportMemberName, err)
	}
	if _, err := w.Write(snapshot); err != nil {
		return nil, fmt.Errorf("write %s to zip: %w", sqliteExportMemberName, err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close sqlite export zip: %w", err)
	}
	return buf.Bytes(), nil
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
