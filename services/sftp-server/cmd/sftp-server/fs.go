package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

// jailedFS implements pkg/sftp interfaces (note the lowercase method names in your version).
// It enforces a strict root and writes JSON audit logs.
type jailedFS struct {
	root       string
	user       string
	remote     string
	quotaBytes int64
	quotaFiles int64
}

func (fs jailedFS) clean(p string) (string, string, error) {
	// Returns (absPath, relPath, error)
	if p == "" {
		p = "."
	}
	// Normalize for safety
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	clean := filepath.Clean(p)

	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("invalid path")
	}

	abs := filepath.Join(fs.root, clean)

	rootAbs, err := filepath.Abs(fs.root)
	if err != nil {
		return "", "", err
	}
	absAbs, err := filepath.Abs(abs)
	if err != nil {
		return "", "", err
	}
	if absAbs != rootAbs && !strings.HasPrefix(absAbs, rootAbs+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path escapes root")
	}

	// rel as the SFTP-visible path
	rel := clean
	if rel == "." {
		rel = "/"
	} else {
		rel = "/" + strings.ReplaceAll(rel, string(os.PathSeparator), "/")
	}
	return absAbs, rel, nil
}

// --- FileReader interface ---
func (fs jailedFS) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	abs, rel, err := fs.clean(r.Filepath)
	audit(fs.user, fs.remote, "get_open", rel, "", 0, err)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(abs)
	audit(fs.user, fs.remote, "get_open", rel, "", 0, err)
	return f, err
}

// --- FileWriter interface ---
func (fs jailedFS) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	abs, rel, err := fs.clean(r.Filepath)
	if err != nil {
		audit(fs.user, fs.remote, "put_open", rel, "", 0, nil)
		return nil, err
	}

	// Ensure parent exists
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		audit(fs.user, fs.remote, "put_open", rel, "", 0, nil)
		return nil, err
	}

	// Check current usage for quota enforcement
	u, err := dirUsage(fs.root)
	if err != nil {
		audit(fs.user, fs.remote, "quota_usage_failed", rel, "", 0, err)
		return nil, err
	}

	if fs.quotaFiles > 0 && u.Files >= fs.quotaFiles {
		err := fmt.Errorf("file quota exceeded")
		audit(fs.user, fs.remote, "put_open", rel, "", 0, err)
		return nil, err
	}

	// Write to temp file in the same directory for atomic rename
	tmp := abs + ".uploading"

	// Remove old temp if exists
	_ = os.Remove(tmp)

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		audit(fs.user, fs.remote, "put_open", rel, "", 0, err)
		return nil, err
	}

	audit(fs.user, fs.remote, "put_open", rel, "", 0, nil)

	return &atomicQuotaWriterAt{
		user:	   fs.user,
		remote:	   fs.remote,
		rel:	   rel,
		tmpPath:   tmp,
		finalPath: abs,
		f:         f,
		baseBytes: u.Bytes,
		quota:     fs.quotaBytes,
	}, nil
}


// --- FileCmder interface ---
func (fs jailedFS) Filecmd(r *sftp.Request) error {
	abs, rel, err := fs.clean(r.Filepath)
	if err != nil {
		audit(fs.user, fs.remote, "cmd_"+r.Method, rel, "", 0, err)
		return err
	}

	switch r.Method {
	case "Remove":
		err = os.Remove(abs)
		audit(fs.user, fs.remote, "rm", rel, "", 0, err)
		return err

	case "Mkdir":
		err = os.MkdirAll(abs, 0o750)
		audit(fs.user, fs.remote, "mkdir", rel, "", 0, err)
		return err

	case "Rmdir":
		err = os.Remove(abs)
		audit(fs.user, fs.remote, "rmdir", rel, "", 0, err)
		return err

	case "Rename":
		tAbs, tRel, tErr := fs.clean(r.Target)
		if tErr != nil {
			audit(fs.user, fs.remote, "rename", rel, tRel, 0, tErr)
			return tErr
		}
		err = os.Rename(abs, tAbs)
		audit(fs.user, fs.remote, "rename", rel, tRel, 0, err)
		return err

	default:
		err = fmt.Errorf("unsupported method: %s", r.Method)
		audit(fs.user, fs.remote, "cmd_unsupported", rel, "", 0, err)
		return err
	}
}

// --- FileLister interface ---
func (fs jailedFS) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	abs, rel, err := fs.clean(r.Filepath)
	if err != nil {
		audit(fs.user, fs.remote, "list_"+r.Method, rel, "", 0, err)
		return nil, err
	}

	switch r.Method {
	case "List":
		entries, err := os.ReadDir(abs)
		if err != nil {
			audit(fs.user, fs.remote, "ls", rel, "", 0, err)
			return nil, err
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
		audit(fs.user, fs.remote, "ls", rel, "", 0, nil)
		return listerAtFromFileInfo(infos), nil

	case "Stat":
		info, err := os.Stat(abs)
		audit(fs.user, fs.remote, "stat", rel, "", 0, err)
		if err != nil {
			return nil, err
		}
		return listerAtFromFileInfo([]os.FileInfo{info}), nil

	default:
		err := fmt.Errorf("unsupported list method: %s", r.Method)
		audit(fs.user, fs.remote, "list_unsupported", rel, "", 0, err)
		return nil, err
	}
}

// Adapter: []os.FileInfo -> sftp.ListerAt
type listerAtFromFileInfo []os.FileInfo

func (l listerAtFromFileInfo) ListAt(dst []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(dst, l[offset:])
	if n < len(dst) {
		return n, io.EOF
	}
	return n, nil
}

