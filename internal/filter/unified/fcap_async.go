package unified

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func (f *UnifiedFilter) scheduleFcapBump(client redis.UniversalClient, key string, window int32) {
	if f == nil || client == nil || key == "" || window <= 0 {
		return
	}
	f.fcapOnce.Do(func() {
		f.fcapQueue = make(chan fcapBumpJob, 512)
		go f.fcapWorkerLoop()
	})
	job := fcapBumpJob{client: client, key: key, secs: window}
	select {
	case f.fcapQueue <- job:
	default:
	}
}

func (f *UnifiedFilter) fcapWorkerLoop() {
	for job := range f.fcapQueue {
		fcapCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		pipe := job.client.Pipeline()
		pipe.Incr(fcapCtx, job.key)
		pipe.Expire(fcapCtx, job.key, time.Duration(job.secs)*time.Second)
		_, _ = pipe.Exec(fcapCtx)
		cancel()
	}
}
