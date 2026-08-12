package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestK8sProfileRemovedFromActiveCode(t *testing.T) {
	root := repoRoot()
	scanRoots := []string{
		filepath.Join(root, "internal"),
		filepath.Join(root, "pkg"),
		filepath.Join(root, "web", "src"),
		filepath.Join(root, "scripts", "ops"),
		filepath.Join(root, "deploy", "monitoring"),
	}

	var hits []string
	for _, scanRoot := range scanRoots {
		err := filepath.Walk(scanRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") || strings.Contains(path, ".test") {
				return nil
			}
			ext := filepath.Ext(path)
			if ext != ".go" && ext != ".js" && ext != ".sh" && ext != ".yaml" && ext != ".yml" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), "k8s_k3s") {
				hits = append(hits, path)
			}
			return nil
		})
		require.NoError(t, err)
	}
	require.Empty(t, hits, "k8s_k3s must not appear in active installer/platform/admin code")
}
