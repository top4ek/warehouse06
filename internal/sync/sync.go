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

// ExportPaths holds the destinations of the derived export artifacts. An empty
// path disables that export: the syncer skips its rebuild and the HTTP handler
// reports the export as unavailable.
type ExportPaths struct {
	Rebus   string
	SQLite  string
	Storage string
}

type Syncer struct {
	log        *zap.Logger
	holder     *repository.Holder
	primaryDSN string
	parser     *parser.Parser
	storageDir string
	storageURL string
	interval   time.Duration
	status     *Status
	exports    ExportPaths
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
	exports ExportPaths,
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
		exports:    exports,
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

	// The storage archive reads only the working tree, so it runs alongside the
	// parse and the database rebuild instead of after them. Waiting for it is
	// deferred right here: the goroutine must not outlive Sync() on any early
	// error return, or it would still be reading storage/ when the next sync
	// pulls into it.
	var storageExportWG sync.WaitGroup
	storageExportWG.Add(1)
	go func() {
		defer storageExportWG.Done()
		if err := s.rebuildStorageExport(ctx); err != nil {
			s.log.Error("storage export rebuild failed", zap.Error(err))
		}
	}()
	defer storageExportWG.Wait()

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

	// Both database exports read the staging repository, before the swap: it
	// serves no traffic yet, so copying it cannot contend with API handlers.
	// A derived artifact must never fail the catalog rebuild, so their errors
	// are logged and dropped. Under the default :memory: DSN the pool holds a
	// single connection, so the two goroutines take turns on it rather than
	// running truly in parallel - that is correct, just not faster.
	var dbExportWG sync.WaitGroup
	dbExportWG.Add(2)
	go func() {
		defer dbExportWG.Done()
		if err := s.rebuildSQLiteExport(ctx, stagingRepo); err != nil {
			s.log.Error("sqlite export rebuild failed", zap.Error(err))
		}
	}()
	go func() {
		defer dbExportWG.Done()
		if err := s.rebuildRebusExport(ctx, stagingRepo); err != nil {
			s.log.Error("rebus export rebuild failed", zap.Error(err))
		}
	}()
	dbExportWG.Wait()

	oldRepo, oldDSN := s.holder.Swap(stagingRepo, stagingDSN)
	if oldRepo != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.disposeRepo(oldRepo, oldDSN)
		}()
	}

	// Hold back the "synced" status until every artifact is on disk, so a
	// successful sync never advertises exports that are still being written.
	storageExportWG.Wait()

	s.status.SetSuccess(time.Now(), head, hasHead)
	s.log.Info("sync completed successfully")
	return nil
}

// rebuildRebusExport regenerates the REBUS-compatible export from the
// freshly-built staging repository and atomically replaces the static file
// the HTTP endpoint serves. Writing to a temp path and renaming over the
// final path (rather than truncating it in place) guarantees a concurrent
// reader never observes a partially-written file: POSIX rename is atomic,
// and an already-open reader keeps referencing the old inode's complete
// data.
func (s *Syncer) rebuildRebusExport(ctx context.Context, repo *repository.SQLiteRepository) error {
	if s.exports.Rebus == "" {
		return nil
	}

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

	tmpPath := s.exports.Rebus + ".tmp"
	if err := os.WriteFile(tmpPath, zipData, 0o644); err != nil {
		return fmt.Errorf("write rebus export temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.exports.Rebus); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename rebus export into place: %w", err)
	}

	s.log.Info("rebus export rebuilt", zap.String("path", s.exports.Rebus), zap.Int("size", len(zipData)))
	return nil
}

// rebuildStorageExport zips the content repository's working tree and
// atomically replaces the static file the HTTP endpoint serves (same rename
// rationale as rebuildRebusExport). Only the files the catalog is built from
// go in: storage.ArchiveDir skips the .git directory along with every other
// dot-prefixed entry.
func (s *Syncer) rebuildStorageExport(ctx context.Context) error {
	if s.exports.Storage == "" {
		return nil
	}

	tmpPath := s.exports.Storage + ".tmp"
	size, err := storage.ArchiveDir(ctx, s.storageDir, tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("archive storage dir: %w", err)
	}
	if err := os.Rename(tmpPath, s.exports.Storage); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename storage export into place: %w", err)
	}

	s.log.Info("storage export rebuilt", zap.String("path", s.exports.Storage), zap.Int64("size", size))
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
	if s.exports.SQLite == "" {
		return nil
	}

	snapshotPath := s.exports.SQLite + ".db.tmp"
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

	tmpPath := s.exports.SQLite + ".tmp"
	if err := os.WriteFile(tmpPath, zipData, 0o644); err != nil {
		return fmt.Errorf("write sqlite export temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.exports.SQLite); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename sqlite export into place: %w", err)
	}

	s.log.Info("sqlite export rebuilt", zap.String("path", s.exports.SQLite), zap.Int("size", len(zipData)))
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
