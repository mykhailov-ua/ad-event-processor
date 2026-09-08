package crowdprobe

import "ad-event-processor/internal/domain"

type ScoreResult struct {
	BehaviorScore uint8
	BehaviorHit   bool
	TimingHit     bool
}

func Score(features BehaviorSessionFeatures, asnTier uint8, clusterPriorMilli uint16) ScoreResult {
	var points uint16
	if features.PathEfficiencyMilli >= 850 {
		points += 25
	}
	if features.ScrollCVMilli > 0 && features.ScrollCVMilli < 40 {
		points += 20
	}
	if features.FittsResidualMilli > 0 && features.FittsResidualMilli < 120 {
		points += 15
	}
	if features.EventOrderEntropyMilli > 0 && features.EventOrderEntropyMilli < 350 {
		points += 20
	}
	if features.FooterReachMs > 0 && features.FooterReachMs < 2500 {
		points += 15
	}
	if asnTier >= 3 && clusterPriorMilli > 0 {
		boost := clusterPriorMilli / 50
		if boost > 20 {
			boost = 20
		}
		points += boost
	}
	score := uint8(points)
	if points > 100 {
		score = 100
	}
	behaviorHit := score >= BehaviorSignalThreshold &&
		(features.PathEfficiencyMilli >= 800 || features.EventOrderEntropyMilli < 400)
	timingHit := features.FooterReachMs > 0 &&
		features.FooterReachMs < 2000 &&
		features.ScrollCVMilli > 0 &&
		features.ScrollCVMilli < 45
	return ScoreResult{
		BehaviorScore: score,
		BehaviorHit:   behaviorHit,
		TimingHit:     timingHit,
	}
}

func ApplyEventFields(evt *domain.Event, features BehaviorSessionFeatures, result ScoreResult) {
	if evt == nil {
		return
	}
	evt.ProbeBehaviorScore = result.BehaviorScore
	evt.ProbeFooterReachMs = features.FooterReachMs
	evt.EventOrderEntropyMilli = features.EventOrderEntropyMilli
	evt.ProbeFeaturesSet = 1
}
