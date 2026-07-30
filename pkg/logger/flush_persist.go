package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func (l *Logger) StartPersister() {
	defer l.wg.Done()
	for buf := range l.persistCh {
		l.writeBuffer(buf)
		buf.Reset()
		bufferPool.Put(buf)
	}
}

func (l *Logger) writeBuffer(buf *AlignedBuffer) {
	l.checkRotation()
	if l.diskDegraded.Load() == 1 {
		l.loadSheddingEvents.Add(uint64(buf.offset / 100))
		return
	}
	data := buf.Bytes()
	start := time.Now()

	n, err := l.activeFile.Write(data)
	if err == nil {
		err = syscall.Fdatasync(int(l.activeFile.Fd()))
	}
	duration := time.Since(start)
	LogNVMEWriteDurationSeconds.Observe(duration.Seconds())
	if err != nil {
		l.diskDegraded.Store(1)
		l.loadSheddingEvents.Add(uint64(buf.offset / 100))
		return
	}
	latencyNs := uint64(duration.Nanoseconds())
	currentEMA := l.emaLatency.Load()
	var newEMA uint64
	if currentEMA == 0 {
		newEMA = latencyNs
	} else {
		newEMA = (latencyNs + 9*currentEMA) / 10
	}
	l.emaLatency.Store(newEMA)
	if newEMA > uint64(l.cfg.DiskLatencyLimit.Nanoseconds()) {
		l.diskDegraded.Store(1)
	}
	l.bytesWritten += int64(n)
}

func (l *Logger) checkDiskSpace() {
	var stat syscall.Statfs_t
	err := syscall.Statfs(l.cfg.LogDir, &stat)
	if err != nil {
		l.diskDegraded.Store(1)
		return
	}
	freeSpace := stat.Bavail * uint64(stat.Bsize)
	if freeSpace < 1024*1024*1024 {
		l.diskDegraded.Store(1)
	} else {
		ema := l.emaLatency.Load()
		if ema <= uint64(l.cfg.DiskLatencyLimit.Nanoseconds()) {
			l.diskDegraded.Store(0)
		} else {
			l.emaLatency.Store(0)
			l.diskDegraded.Store(0)
		}
	}
}

func (l *Logger) StartDiskMonitor() {
	defer l.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-l.closeChan:
			return
		case <-ticker.C:
			l.checkDiskSpace()
		}
	}
}

func (l *Logger) checkRotation() {
	if l.activeFile == nil {
		l.openActiveFile()
		return
	}
	sizeReached := l.bytesWritten >= l.cfg.RotateSize
	timeReached := time.Since(l.fileOpenedAt) >= l.cfg.RotateInterval
	if sizeReached || timeReached {
		_ = l.activeFile.Close()
		timestamp := time.Now().Format("20060102-150405.000000000")
		rotatedPath := filepath.Join(l.cfg.LogDir, fmt.Sprintf("segment_%s.log", timestamp))
		activePath := filepath.Join(l.cfg.LogDir, "active.log")
		_ = os.Rename(activePath, rotatedPath)
		LogRotationTotal.Inc()
		l.openActiveFile()
	}
}

func (l *Logger) openActiveFile() {
	activePath := filepath.Join(l.cfg.LogDir, "active.log")
	f, err := os.OpenFile(activePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		l.diskDegraded.Store(1)
		return
	}
	l.activeFile = f
	l.fileOpenedAt = time.Now()
	l.bytesWritten = 0
}
