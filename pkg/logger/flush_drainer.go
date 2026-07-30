package logger

import (
	"encoding/binary"
	"runtime"
	"sync/atomic"
	"time"
)

func (l *Logger) StartDrainer() {
	defer l.wg.Done()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	buf := l.getBuffer()
	var firstLogAt time.Time
	for {
		select {
		case <-l.closeChan:
			buf, _ = l.drainShards(buf)
			l.sendBuffer(buf, true)
			close(l.persistCh)
			return
		case <-ticker.C:
			var flushed bool
			buf, flushed = l.drainShards(buf)
			if flushed && firstLogAt.IsZero() {
				firstLogAt = time.Now()
			}
			if buf.offset > 0 {
				if buf.offset >= l.cfg.FlushBufferSize || time.Since(firstLogAt) >= 50*time.Millisecond {
					l.sendBuffer(buf, false)
					buf = l.getBuffer()
					firstLogAt = time.Time{}
				}
			}
		}
	}
}

func (l *Logger) drainShards(buf *AlignedBuffer) (*AlignedBuffer, bool) {
	degraded := l.diskDegraded.Load() == 1
	var flushed bool
	for _, shard := range l.shards {
		writeCursor := atomic.LoadUint64(&shard.writeCursor)
		readCursor := atomic.LoadUint64(&shard.readCursor)
		for readCursor < writeCursor {
			idx := readCursor & RingMask
			payload := &shard.slots[idx]
			for payload.ready.Load() == 0 {
				runtime.Gosched()
			}
			if degraded && payload.Priority == 0 {
				l.loadSheddingEvents.Add(1)
				readCursor++
				continue
			}
			logBytes := payload.Data[:payload.Length]
			totalSize := 4 + len(logBytes)
			if buf.Available() < totalSize {
				l.sendBuffer(buf, false)
				buf = l.getBuffer()
			}
			var lenBuf [4]byte
			binary.BigEndian.PutUint32(lenBuf[:], uint32(len(logBytes)))
			buf.Write(lenBuf[:])
			buf.Write(logBytes)
			payload.ready.Store(0)
			flushed = true
			readCursor++
		}
		atomic.StoreUint64(&shard.readCursor, readCursor)
	}
	return buf, flushed
}

func (l *Logger) recordPersistQueueDrop(buf *AlignedBuffer) {
	l.persistQueueDrops.Add(1)
	l.persistQueueDropBytes.Add(uint64(buf.offset))
	l.loadSheddingEvents.Add(uint64(buf.offset / 100))
}

func (l *Logger) sendBuffer(buf *AlignedBuffer, blocking bool) {
	if buf.offset == 0 {
		bufferPool.Put(buf)
		return
	}
	if l.persistCh == nil {
		buf.Reset()
		bufferPool.Put(buf)
		return
	}
	if blocking {
		l.persistCh <- buf
		return
	}
	timer := time.NewTimer(l.cfg.PersistEnqueueTimeout)
	defer timer.Stop()
	select {
	case l.persistCh <- buf:
	case <-timer.C:
		l.recordPersistQueueDrop(buf)
		buf.Reset()
		bufferPool.Put(buf)
	}
}
