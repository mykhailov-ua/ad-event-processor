package landerhost

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const DefaultMaxEditorFileBytes int64 = 1 << 20

type FileEntry struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Editable bool   `json:"editable"`
}

func IsEditableTextPath(relPath string) bool {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(relPath)))
	switch {
	case strings.HasSuffix(name, ".html"), strings.HasSuffix(name, ".htm"):
		return true
	case strings.HasSuffix(name, ".css"), strings.HasSuffix(name, ".js"):
		return true
	case strings.HasSuffix(name, ".txt"), strings.HasSuffix(name, ".json"), strings.HasSuffix(name, ".svg"):
		return true
	default:
		return false
	}
}

func (st *Store) ListVersionFiles(landerID uuid.UUID, version int) ([]FileEntry, error) {
	if st == nil {
		return nil, fmt.Errorf("lander store unavailable")
	}
	if version <= 0 {
		return nil, fmt.Errorf("version must be positive")
	}
	root := st.VersionDir(landerID, version)
	var out []FileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, FileEntry{
			Path:     rel,
			Size:     info.Size(),
			Editable: IsEditableTextPath(rel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("version has no files")
	}
	return out, nil
}

func (st *Store) ReadVersionFile(landerID uuid.UUID, version int, relPath string) ([]byte, error) {
	rc, info, err := st.openVersionFile(landerID, version, relPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	if info.Size() > DefaultMaxEditorFileBytes {
		return nil, fmt.Errorf("file exceeds editor size limit")
	}
	if !IsEditableTextPath(relPath) {
		return nil, fmt.Errorf("file is not editable as text")
	}
	return io.ReadAll(io.LimitReader(rc, DefaultMaxEditorFileBytes+1))
}

func (st *Store) CloneVersion(landerID uuid.UUID, fromVersion, toVersion int) error {
	if st == nil {
		return fmt.Errorf("lander store unavailable")
	}
	if fromVersion <= 0 || toVersion <= 0 {
		return fmt.Errorf("version must be positive")
	}
	src := st.VersionDir(landerID, fromVersion)
	dst := st.VersionDir(landerID, toVersion)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source version missing: %w", err)
	}
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("clear destination version: %w", err)
	}
	return copyDir(src, dst)
}

func (st *Store) WriteVersionTextFile(landerID uuid.UUID, version int, relPath string, content []byte) error {
	if st == nil {
		return fmt.Errorf("lander store unavailable")
	}
	if int64(len(content)) > DefaultMaxEditorFileBytes {
		return fmt.Errorf("file exceeds editor size limit")
	}
	if !IsEditableTextPath(relPath) {
		return fmt.Errorf("file is not editable as text")
	}
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return err
	}
	root := st.VersionDir(landerID, version)
	target := filepath.Join(root, filepath.FromSlash(clean))
	if !pathWithinRoot(root, target) {
		return fmt.Errorf("path outside version root")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, content, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func (st *Store) openVersionFile(landerID uuid.UUID, version int, relPath string) (io.ReadCloser, os.FileInfo, error) {
	if st == nil {
		return nil, nil, fmt.Errorf("lander store unavailable")
	}
	relPath = strings.TrimSpace(relPath)
	if relPath == "" || relPath == "/" {
		relPath = indexFileName
	}
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return nil, nil, err
	}
	root := st.VersionDir(landerID, version)
	full := filepath.Join(root, filepath.FromSlash(clean))
	if !pathWithinRoot(root, full) {
		return nil, nil, fmt.Errorf("path outside version root")
	}
	info, err := os.Lstat(full)
	if err != nil {
		return nil, nil, err
	}
	if info.IsDir() {
		full = filepath.Join(full, indexFileName)
		info, err = os.Lstat(full)
		if err != nil {
			return nil, nil, err
		}
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, nil, err
	}
	return f, info, nil
}

func (st *Store) OpenPreviewFile(landerID uuid.UUID, version int, relPath string) (io.ReadCloser, os.FileInfo, error) {
	return st.openVersionFile(landerID, version, relPath)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()
		_, err = io.Copy(out, io.LimitReader(in, DefaultMaxZipBytes))
		return err
	})
}
