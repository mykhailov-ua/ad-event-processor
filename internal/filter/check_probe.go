package filter

import (
	"context"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
)

//go:noinline
func FilterCheckEnter(slot uint32) { _ = slot }

//go:noinline
func FilterCheckExit(slot uint32) { _ = slot }

type BufWrapper struct {
	Buf []byte
}

type bufWrapper = BufWrapper

var bufPool = sync.Pool{
	New: func() any {
		return &BufWrapper{Buf: make([]byte, 0, 64)}
	},
}

func AddFraudSignal(evt *domain.Event, id FraudReasonID) {
	addFraudSignal(evt, id)
}

func FilterDeadlineExceededOnEvent(evt *domain.Event, ctx context.Context) bool {
	return filterDeadlineExceededEvt(evt, ctx)
}

func FilterDeadlineRemainingOnEvent(evt *domain.Event, ctx context.Context) (time.Duration, bool) {
	return filterDeadlineRemainingEvt(evt, ctx)
}

func AcquireBufWrapper() *BufWrapper {
	return bufPool.Get().(*BufWrapper)
}

func ReleaseBufWrapper(w *BufWrapper) {
	if w != nil {
		bufPool.Put(w)
	}
}
