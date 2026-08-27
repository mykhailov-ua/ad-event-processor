package notify

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Worker struct {
	service   *Service
	interval  time.Duration
	batchSize int32
	wg        sync.WaitGroup
}

func NewWorker(service *Service, interval time.Duration, batchSize int32) *Worker {
	if interval <= 0 {
		interval = time.Second
	}
	if batchSize <= 0 {
		batchSize = workerBatchSize
	}
	return &Worker{
		service:   service,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		slog.Info("notification worker starting polling loop", "interval", w.interval, "batch_size", w.batchSize)

		timer := time.NewTimer(w.interval)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("notification worker stopping polling loop")
				return
			case <-timer.C:
				processed, err := w.service.ProcessPending(ctx, w.batchSize)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					workerIterationErrorsTotal.Inc()
					slog.Error("notification worker processing iteration failed", "error", err, "retry_in", workerErrorBackoff)
					timer.Reset(workerErrorBackoff)
					continue
				}

				if processed > 0 {
					workerBatchProcessed.Observe(float64(processed))
					timer.Reset(0)
				} else {
					timer.Reset(w.interval)
				}
			}
		}
	}()
}

func (w *Worker) Wait() {
	w.wg.Wait()
}

func (w *Worker) StartPool(ctx context.Context, concurrency int) {
	if concurrency <= 1 {
		w.Start(ctx)
		return
	}
	for range concurrency {
		w.Start(ctx)
	}
}
