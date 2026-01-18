package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type usage struct {
	Bytes int64
	Files int64
}

func dirUsage(root string) (usage, error) {
	var u usage
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		u.Bytes += info.Size()
		u.Files++
		return nil
	})
	return u, err
}

// atomicQuotaWriterAt writes to a TEMP file and only renames to final on Close if within quota.
// If quota is exceeded at any point, it deletes the temp file.
type atomicQuotaWriterAt struct {
	user      string
	remote	  string
	rel	  string

	tmpPath   string
	finalPath string

	f         *os.File
	baseBytes int64
	quota     int64

	maxEnd   int64
	exceeded bool
}

func (w *atomicQuotaWriterAt) WriteAt(p []byte, off int64) (int, error) {
	if w.exceeded {
		return 0, fmt.Errorf("quota exceeded")
	}

	end := off + int64(len(p))
	if end > w.maxEnd {
		w.maxEnd = end
	}

	if w.quota > 0 && (w.baseBytes+w.maxEnd) > w.quota {
		w.exceeded = true
		_ = w.f.Close()
		_ = os.Remove(w.tmpPath)

		// Final outcome: fail
		audit(w.user, w.remote, "put_fail", w.rel, "", w.maxEnd, fmt.Errorf("quota exceeded"))
		return 0, fmt.Errorf("quota exceeded")
	}

	return w.f.WriteAt(p, off)
}

func (w *atomicQuotaWriterAt) Close() error {
	_ = w.f.Close()

	// If we exceeded, temp already removed.
	if w.exceeded {
		return fmt.Errorf("quota exceeded")
	}

	// Commit atomically
	if err := os.Rename(w.tmpPath, w.finalPath); err != nil {
		_ = os.Remove(w.tmpPath)
		return err
	}


	// Final outcome: success
	audit(w.user, w.remote, "put_commit", w.rel, "", w.maxEnd, nil)
	return nil
}

var _ io.WriterAt = (*atomicQuotaWriterAt)(nil)
var _ io.Closer = (*atomicQuotaWriterAt)(nil)

