package ingest

import "time"

func clickResponseTimingPad(startMono int64, minPadMs int) {
	if minPadMs <= 0 || startMono <= 0 {
		return
	}
	elapsed := (monotonicNano() - startMono) / int64(time.Millisecond)
	if elapsed >= int64(minPadMs) {
		return
	}
	time.Sleep(time.Duration(int64(minPadMs)-elapsed) * time.Millisecond)
}

func (h *AdsPacketHandler) clickTimingPadMs() int {
	if h == nil || h.cfg == nil {
		return 0
	}
	return h.cfg.ClickTimingPadMs
}
