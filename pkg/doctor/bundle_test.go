package doctor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteBundle(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "bundle.tar.gz")
	err := WriteBundle(context.Background(), BundleOptions{
		Out:     out,
		Deps:    ProbeDeps{},
		Only:    []string{"kernel"},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	found := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		found[hdr.Name] = true
	}
	for _, want := range []string{"doctor/report.json", "doctor/checklist.json", "version.json", "config/sanitized.env", "README.txt"} {
		if !found[want] {
			t.Fatalf("missing %s in bundle, have %v", want, found)
		}
	}
}
