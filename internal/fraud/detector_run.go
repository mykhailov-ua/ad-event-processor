package fraud

import (
	"context"
	"fmt"
	"log/slog"
)

type claimedThreat struct {
	ip     string
	reason string
	action string
}

func (detector *Detector) Run(ctx context.Context) (RunResult, error) {
	var result RunResult
	if detector == nil {
		return result, fmt.Errorf("detector: nil receiver")
	}

	backlogged, err := detector.outboxBacklogged(ctx)
	if err != nil {
		return result, err
	}
	if backlogged {
		result.Backlogged = true
		ivtBackpressureDropsTotal.Inc()
		return result, ErrOutboxBackpressure
	}

	candidates, err := detector.analyzer.FindSuspiciousIPs(ctx)
	if err != nil {
		return result, err
	}
	result.Candidates = len(candidates)

	var batchItems []FraudThreatEnqueueItem
	var batchClaimed []claimedThreat
	var batchPending int

	flushThreatBatch := func() error {
		if len(batchItems) == 0 {
			return nil
		}
		_, err := detector.management.EnqueueFraudThreatBatch(ctx, batchItems)
		if err != nil {
			for _, claim := range batchClaimed {
				detector.releaseThreatClaim(ctx, claim)
			}
			batchItems = nil
			batchClaimed = nil
			batchPending = 0
			return err
		}
		result.Enqueued += batchPending
		batchItems = nil
		batchClaimed = nil
		batchPending = 0
		return nil
	}

	for _, candidate := range candidates {
		switch candidate.Action {
		case "boost":
			claimed, claimErr := detector.idem.TryClaimFraudEnforcement(ctx, candidate.IP, candidate.Reason, "boost")
			if claimErr != nil {
				return result, claimErr
			}
			if !claimed {
				result.Skipped++
				continue
			}

			batchItems = append(batchItems, FraudThreatEnqueueItem{
				Action:     "boost",
				IP:         candidate.IP,
				CampaignID: candidate.CampaignID,
				Score:      candidate.Score,
				Boost:      candidate.Boost,
				TTLSeconds: candidate.TTLSeconds,
			})
			batchClaimed = append(batchClaimed, claimedThreat{
				ip:     candidate.IP,
				reason: candidate.Reason,
				action: "boost",
			})
			batchPending++
			fraudEnforcementEnqueuedTotal.WithLabelValues("boost").Inc()
			slog.Info("ivt detector staged ml score boost",
				"ip", candidate.IP,
				"campaign_id", candidate.CampaignID,
				"score", candidate.Score,
				"boost", candidate.Boost,
			)
		case "silent_reject", "ghost":
			claimed, claimErr := detector.idem.TryClaimFraudEnforcement(ctx, candidate.IP, candidate.Reason, "silent_reject")
			if claimErr != nil {
				return result, claimErr
			}
			if !claimed {
				result.Skipped++
				continue
			}

			batchItems = append(batchItems, FraudThreatEnqueueItem{
				Action:     "silent_reject",
				IP:         candidate.IP,
				CampaignID: candidate.CampaignID,
				Score:      candidate.Score,
				TTLSeconds: candidate.TTLSeconds,
			})
			batchClaimed = append(batchClaimed, claimedThreat{
				ip:     candidate.IP,
				reason: candidate.Reason,
				action: "silent_reject",
			})
			batchPending++
			fraudEnforcementEnqueuedTotal.WithLabelValues("silent_reject").Inc()
			slog.Info("ivt detector staged ml silent reject",
				"ip", candidate.IP,
				"campaign_id", candidate.CampaignID,
				"signal", candidate.Reason,
				"score", candidate.Score,
			)
		case "blacklist":
			claimed, claimErr := detector.idem.TryClaimFraudEnforcement(ctx, candidate.IP, candidate.Reason, "blacklist")
			if claimErr != nil {
				return result, claimErr
			}
			if !claimed {
				result.Skipped++
				continue
			}

			batchItems = append(batchItems, FraudThreatEnqueueItem{
				Action:     "blacklist",
				IP:         candidate.IP,
				CampaignID: candidate.CampaignID,
				Score:      candidate.Score,
				TTLSeconds: candidate.TTLSeconds,
			})
			batchClaimed = append(batchClaimed, claimedThreat{
				ip:     candidate.IP,
				reason: candidate.Reason,
				action: "blacklist",
			})
			batchPending++
			fraudEnforcementEnqueuedTotal.WithLabelValues("blacklist").Inc()
			slog.Info("ivt detector staged ml blacklist",
				"ip", candidate.IP,
				"campaign_id", candidate.CampaignID,
				"signal", candidate.Reason,
				"score", candidate.Score,
			)
		default:
			claimed, claimErr := detector.idem.TryClaim(ctx, candidate.IP)
			if claimErr != nil {
				return result, claimErr
			}
			if !claimed {
				result.Skipped++
				continue
			}

			blockErr := detector.management.BlockIP(ctx, candidate.IP)
			if blockErr != nil {
				if releaseErr := detector.idem.Release(ctx, candidate.IP); releaseErr != nil {
					slog.Error("failed to release idempotency claim after management error",
						"ip", candidate.IP,
						"block_error", blockErr,
						"release_error", releaseErr,
					)
				}
				return result, blockErr
			}

			result.Enqueued++
			ivtEnqueuedTotal.Inc()
			slog.Info("ivt detector enqueued fraud blacklist",
				"ip", candidate.IP,
				"signal", candidate.Reason,
				"score", candidate.Score,
			)
		}
	}

	if err := flushThreatBatch(); err != nil {
		return result, err
	}

	return result, nil
}

func (detector *Detector) releaseThreatClaim(ctx context.Context, claim claimedThreat) {
	switch claim.action {
	case "boost", "silent_reject", "ghost", "blacklist":
		if releaseErr := detector.idem.ReleaseFraudEnforcement(ctx, claim.ip, claim.reason, claim.action); releaseErr != nil {
			slog.Error("failed to release fraud enforcement claim after batch error",
				"ip", claim.ip,
				"action", claim.action,
				"release_error", releaseErr,
			)
		}
	default:
		if releaseErr := detector.idem.Release(ctx, claim.ip); releaseErr != nil {
			slog.Error("failed to release idempotency claim after batch error",
				"ip", claim.ip,
				"release_error", releaseErr,
			)
		}
	}
}
