package landerhost

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractZipSafe(r io.ReaderAt, size int64, dest string) (int, error) {
	if size <= 0 {
		return 0, fmt.Errorf("empty zip")
	}
	if size > DefaultMaxZipBytes {
		return 0, fmt.Errorf("zip exceeds max size %d bytes", DefaultMaxZipBytes)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}
	dest = filepath.Clean(dest)
	written := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rel, err := cleanRelativePath(f.Name)
		if err != nil {
			return 0, err
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if !pathWithinRoot(dest, target) {
			return 0, fmt.Errorf("zip path escapes destination: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return 0, err
		}
		if err := writeZipEntry(f, target); err != nil {
			return 0, err
		}
		written++
	}
	if written == 0 {
		return 0, fmt.Errorf("zip contains no files")
	}
	return written, nil
}

func writeZipEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm()&0o777)
	if err != nil {
		return err
	}
	defer out.Close()
	n, err := io.Copy(out, io.LimitReader(rc, DefaultMaxZipBytes))
	if err != nil {
		return err
	}
	if n > DefaultMaxZipBytes {
		return fmt.Errorf("zip entry too large")
	}
	return nil
}

func cleanRelativePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")
	if name == "" || name == "." {
		return "", fmt.Errorf("invalid zip entry name")
	}
	parts := strings.Split(name, "/")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			return "", fmt.Errorf("zip entry traverses upward: %s", name)
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return "", fmt.Errorf("invalid zip entry name")
	}
	return strings.Join(clean, "/"), nil
}
