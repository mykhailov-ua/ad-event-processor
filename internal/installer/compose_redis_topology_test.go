package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeRedisTopology_applianceFourShards(t *testing.T) {
	path := filepath.Join(repoRoot(), "deploy", "compose", "docker-compose.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "redis-4:")
	block4 := extractServiceBlock(content, "redis-4:")
	require.NotEmpty(t, block4)
	assert.Contains(t, block4, "- infra")
	block5 := extractServiceBlock(content, "redis-5:")
	require.NotEmpty(t, block5)
	assert.Contains(t, block5, "- infra")

	for _, tracker := range []string{"tracker-0:", "tracker-1:", "tracker-2:", "tracker-3:"} {
		block := extractServiceBlock(content, tracker)
		require.NotEmpty(t, block, "missing %s block", tracker)
		assert.NotContains(t, block, "redis-4:")
		assert.NotContains(t, block, "redis-5:")
	}
}

func TestStackSh_singleVPSRedisServiceList(t *testing.T) {
	path := filepath.Join(repoRoot(), "scripts", "dev", "stack.sh")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "SINGLE_VPS=") {
			continue
		}
		assert.NotContains(t, line, "redis-4")
		assert.NotContains(t, line, "redis-5")
		return
	}
	t.Fatal("SINGLE_VPS line missing in stack.sh")
}

func extractServiceBlock(yaml, service string) string {
	idx := strings.Index(yaml, "\n  "+service)
	if idx < 0 {
		idx = strings.Index(yaml, "\n "+service)
	}
	if idx < 0 {
		return ""
	}
	rest := yaml[idx+1:]
	end := strings.Index(rest, "\n\n ")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
