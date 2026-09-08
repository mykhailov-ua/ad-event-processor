// Package crowdwave detects coordinated hybrid crowd probe waves per campaign.
//
// Role:
// - BehaviorSimhash from telemetry event-order template.
// - WavePolicy thresholds; EvaluateWave for ring-buffer scoring.
// - Redis I/O in internal/filter CrowdWaveStore.
//
// Verify:
// go test ./pkg/crowdwave/ -short -run Wave -count=1
// go test ./internal/filter/ -short -run CrowdWave -count=1
package crowdwave
