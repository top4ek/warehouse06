package storage

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// archiveBufferSize buffers writes to the destination file so the walk does not
// issue a syscall per compressed block.
const archiveBufferSize = 1 << 20

// ArchiveDir zips the regular files under dir into destPath, skipping every
// dot-prefixed entry - the repository's own .git directory and the .keep
// sentinel. That is the same filter the parser applies when it scans this tree
// (see internal/parser.Parser.ScanDirectory), so the archive holds exactly the
// files the catalog was built from.
//
// The archive is streamed straight to destPath rather than assembled in memory:
// the storage tree is two orders of magnitude larger than the database exports,
// so buffering it would tie peak RSS to the size of the content repository.
//
// On error the partial destPath is left in place for the caller to remove.
func ArchiveDir(ctx context.Context, dir, destPath string) (int64, error) {
	f, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create archive file: %w", err)
	}
	defer func() { _ = f.Close() }()

	bw := bufio.NewWriterSize(f, archiveBufferSize)
	zw := zip.NewWriter(bw)

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		// The walk root keeps its name whatever it starts with; only entries
		// inside the tree are subject to the dot filter.
		if path != dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			// Directories need no member of their own: unpackers recreate them
			// from the paths of the files inside.
			return nil
		}
		// Symlinks, sockets and devices have no meaningful zip representation.
		if !d.Type().IsRegular() {
			return nil
		}

		return archiveFile(zw, dir, path, d)
	})
	if err != nil {
		return 0, fmt.Errorf("walk %s: %w", dir, err)
	}

	if err := zw.Close(); err != nil {
		return 0, fmt.Errorf("close archive: %w", err)
	}
	if err := bw.Flush(); err != nil {
		return 0, fmt.Errorf("flush archive: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat archive file: %w", err)
	}
	if err := f.Close(); err != nil {
		return 0, fmt.Errorf("close archive file: %w", err)
	}

	return info.Size(), nil
}

func archiveFile(zw *zip.Writer, dir, path string, d fs.DirEntry) error {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return fmt.Errorf("relative path of %s: %w", path, err)
	}

	info, err := d.Info()
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("build zip header for %s: %w", path, err)
	}
	header.Name = filepath.ToSlash(rel)
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("add %s to archive: %w", header.Name, err)
	}

	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = src.Close() }()

	if _, err := io.Copy(w, src); err != nil {
		return fmt.Errorf("copy %s into archive: %w", header.Name, err)
	}
	return nil
}
