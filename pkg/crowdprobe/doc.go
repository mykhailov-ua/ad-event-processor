// Package crowdprobe scores coordinated human-in-the-loop probe sessions from
// behavior telemetry and antifraud snapshot kinematics.
//
// Role:
// - Pure feature extraction and scoring; no Redis I/O.
// - CrowdProbeFilter in internal/filter adds ASN tier + cluster prior lookup.
//
// Verify:
// go test ./pkg/crowdprobe/ -short -run Corpus -count=1
// go test ./internal/filter/ -short -run CrowdProbe -count=1
package crowdprobe
