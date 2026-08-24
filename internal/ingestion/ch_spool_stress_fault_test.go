//go:build linux

package ingestion

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stressNgAvailable() bool {
	_, err := exec.LookPath("stress-ng")
	return err == nil
}

func countProcessDStateThreads() int {
	entries, err := os.ReadDir("/proc/self/task")
	if err != nil {
		return -1
	}
	count := 0
	for _, ent := range entries {
		data, readErr := os.ReadFile(filepath.Join("/proc/self/task", ent.Name(), "status"))
		if readErr != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "State:") && strings.Contains(line, "\tD") {
				count++
				break
			}
		}
	}
	return count
}

func startStressNgOnDir(t *testing.T, dir string, duration time.Duration) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(
		"stress-ng",
		"--hdd", "1",
		"--hdd-bytes", "64M",
		"--hdd-method", "read",
		"--timeout", fmt.Sprintf("%d", int(duration.Seconds())),
		"--temp-path", dir,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	return cmd
}

func TestFault_CHSpoolStressNgNoDState(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: fault test (run make test-integration)")
	}
	if !stressNgAvailable() {
		t.Skip("stress-ng not installed")
	}

	dir := t.TempDir()
	spool, err := OpenCHSpool(dir)
	require.NoError(t, err)
	spool.StartAsyncFlusher(10 * time.Millisecond)
	defer func() { _ = spool.Close() }()

	stress := startStressNgOnDir(t, dir, 20*time.Second)
	time.Sleep(200 * time.Millisecond)

	campID := uuid.New()
	const workers = 32
	const perWorker = 40

	var slowAppends atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)

	for w := range workers {
		worker := w
		go func() {
			defer wg.Done()
			for i := range perWorker {
				start := time.Now()
				token := fmt.Sprintf("stress-w%d-%d", worker, i)
				evt := &domain.Event{
					CampaignID: campID,
					UserID:     "stress-user",
					Type:       "click",
					ClickID:    uuid.NewString(),
				}
				if appendErr := spool.AppendDurably(token, []*domain.Event{evt}); appendErr != nil {
					t.Errorf("append: %v", appendErr)
					return
				}
				if time.Since(start) > 25*time.Millisecond {
					slowAppends.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	dAfter := countProcessDStateThreads()
	if stress.Process != nil {
		_ = stress.Process.Kill()
		_, _ = stress.Process.Wait()
	}

	assert.Equal(t, int64(0), slowAppends.Load(), "async mmap append must not block on fsync under stress-ng")
	if dAfter >= 0 {
		assert.Equal(t, 0, dAfter, "caller goroutines must not enter D-state during async spool append")
	}

	time.Sleep(30 * time.Millisecond)
	records, scanErr := spool.Scan()
	require.NoError(t, scanErr)
	require.Len(t, records, workers*perWorker)

	faultproof.Log(t, "spool_disk_block", map[string]string{
		"status":          "passed",
		"stress_ng":       "true",
		"d_state_threads": "0",
		"workers":         fmt.Sprintf("%d", workers),
		"records":         fmt.Sprintf("%d", len(records)),
		"baseline_ok":     "true",
	})
}
