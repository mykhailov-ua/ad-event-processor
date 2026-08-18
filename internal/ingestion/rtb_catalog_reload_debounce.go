package ingestion

import (
	"context"
	"time"
)

const rtbCatalogReloadDebounce = 100 * time.Millisecond

func runRtbCatalogReloadDebouncer(ctx context.Context, trigger <-chan struct{}, reload func(), debounce time.Duration) {
	if debounce <= 0 {
		debounce = rtbCatalogReloadDebounce
	}
	debounceTimer := time.NewTimer(time.Hour)
	if !debounceTimer.Stop() {
		select {
		case <-debounceTimer.C:
		default:
		}
	}
	for {
		select {
		case <-ctx.Done():
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			return
		case <-trigger:
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer.Reset(debounce)
		case <-debounceTimer.C:
			reload()
		}
	}
}
