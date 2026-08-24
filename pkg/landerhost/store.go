package landerhost

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	DefaultMaxZipBytes int64 = 32 << 20
	indexFileName            = "index.html"
)

type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("lander store root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("lander store root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir lander store: %w", err)
	}
	return &Store{root: abs}, nil
}

func (st *Store) Root() string {
	if st == nil {
		return ""
	}
	return st.root
}

func PublicURL(base string, landerID uuid.UUID) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	return base + "/lp/" + landerID.String() + "/"
}

func (st *Store) LiveDir(landerID uuid.UUID) string {
	return filepath.Join(st.root, landerID.String(), "live")
}

func (st *Store) VersionDir(landerID uuid.UUID, version int) string {
	return filepath.Join(st.root, landerID.String(), fmt.Sprintf("v%d", version))
}

func (st *Store) ExtractZip(landerID uuid.UUID, version int, r io.ReaderAt, size int64) (entryCount int, err error) {
	if st == nil {
		return 0, fmt.Errorf("lander store unavailable")
	}
	if landerID == uuid.Nil {
		return 0, fmt.Errorf("lander id required")
	}
	if version <= 0 {
		return 0, fmt.Errorf("version must be positive")
	}
	dest := st.VersionDir(landerID, version)
	if err := os.RemoveAll(dest); err != nil {
		return 0, fmt.Errorf("clear version dir: %w", err)
	}
	if err := os.MkdirAll(dest, 0o750); err != nil {
		return 0, fmt.Errorf("mkdir version dir: %w", err)
	}
	count, err := extractZipSafe(r, size, dest)
	if err != nil {
		_ = os.RemoveAll(dest)
		return 0, err
	}
	if err := ensureIndexHTML(dest); err != nil {
		_ = os.RemoveAll(dest)
		return 0, err
	}
	return count, nil
}

func (st *Store) PublishVersion(landerID uuid.UUID, version int) error {
	if st == nil {
		return fmt.Errorf("lander store unavailable")
	}
	src := st.VersionDir(landerID, version)
	if _, err := os.Stat(filepath.Join(src, indexFileName)); err != nil {
		return fmt.Errorf("published version missing index.html: %w", err)
	}
	parent := filepath.Join(st.root, landerID.String())
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return fmt.Errorf("mkdir lander dir: %w", err)
	}
	live := st.LiveDir(landerID)
	tmp := live + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := os.Symlink(filepath.Join(parent, fmt.Sprintf("v%d", version)), tmp); err != nil {
		return fmt.Errorf("symlink live: %w", err)
	}
	_ = os.Remove(live)
	if err := os.Rename(tmp, live); err != nil {
		_ = os.RemoveAll(tmp)
		return fmt.Errorf("activate live symlink: %w", err)
	}
	return nil
}

func (st *Store) OpenLiveFile(landerID uuid.UUID, relPath string) (io.ReadCloser, os.FileInfo, error) {
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
	full := filepath.Join(st.LiveDir(landerID), filepath.FromSlash(clean))
	liveRoot := st.LiveDir(landerID)
	if !pathWithinRoot(liveRoot, full) {
		return nil, nil, fmt.Errorf("path outside lander root")
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

func pathWithinRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}

func ensureIndexHTML(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, indexFileName)); err == nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return fmt.Errorf("zip must contain index.html at root or a single top-level folder")
	}
	nested := filepath.Join(dir, entries[0].Name())
	if _, err := os.Stat(filepath.Join(nested, indexFileName)); err != nil {
		return fmt.Errorf("zip must contain index.html")
	}
	flat, err := os.ReadDir(nested)
	if err != nil {
		return err
	}
	for _, ent := range flat {
		src := filepath.Join(nested, ent.Name())
		dst := filepath.Join(dir, ent.Name())
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("flatten zip root: %w", err)
		}
	}
	return os.Remove(nested)
}
