package ledger

import (
	"testing"
	"time"
)

func TestWorker_WaitDrainsInFlightCycle(t *testing.T) {
	w := &Worker{}

	done := make(chan struct{})
	w.cycleWG.Add(1)
	go func() {
		defer w.cycleWG.Done()
		defer close(done)
		time.Sleep(30 * time.Millisecond)
	}()

	w.Wait()
	<-done
}
