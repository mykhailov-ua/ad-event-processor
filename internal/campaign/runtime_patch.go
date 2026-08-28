package campaign

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/proxyupstream"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func patchCampaign(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error) {
	camp, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return CampaignDTO{}, err
	}
	if err := assertMediaBuyerCampaignAccess(ctx, camp); err != nil {
		return CampaignDTO{}, err
	}
	if req.ExpectedRevision != nil {
		currentRev := campaignRevision(camp.UpdatedAt.Time.Format(time.RFC3339))
		if currentRev != strings.TrimSpace(*req.ExpectedRevision) {
			return CampaignDTO{}, ErrCampaignRevisionConflict
		}
	}

	if req.FlowID != nil {
		if err := fx.AssignCampaignFlow(ctx, campaignID, *req.FlowID); err != nil {
			return CampaignDTO{}, err
		}
	}

	if req.IngressCostConfig != nil {
		if err := fx.ApplyCampaignIngressCostPatch(ctx, campaignID, *req.IngressCostConfig); err != nil {
			return CampaignDTO{}, err
		}
	}

	clickPresetPatch := req.TrafficTemplateID != nil || req.ClickQueryParams != nil
	if clickPresetPatch {
		if err := fx.ApplyCampaignClickPresetPatch(ctx, campaignID, req.TrafficTemplateID, req.ClickQueryParams); err != nil {
			return CampaignDTO{}, err
		}
	}

	if req.BrandID != nil {
		if err := fx.AssignCampaignBrand(ctx, campaignID, *req.BrandID); err != nil {
			return CampaignDTO{}, err
		}
	}

	if req.PacingMode != nil {
		if _, err := updateCampaignPacing(ctx, pool, fx, campaignID, *req.PacingMode); err != nil {
			return CampaignDTO{}, err
		}
	}

	budgetMicro, err := resolvePatchBudgetLimitMicro(req)
	if err != nil {
		return CampaignDTO{}, err
	}
	statusWant, statusSet, err := parsePatchStatus(req.Status)
	if err != nil {
		return CampaignDTO{}, err
	}
	schedulePatch := req.StartAt != nil || req.EndAt != nil || req.DaypartHours != nil

	adminPatch := req.Name != nil || req.DailyBudgetMicro != nil || req.Timezone != nil ||
		req.FreqLimit != nil || req.FreqWindow != nil || req.TargetCountries != nil ||
		req.TargetURL != nil || req.ReferrerFilter != nil ||
		req.SafePageURL != nil || req.SafePageEnabled != nil || req.AttestationEnabled != nil || req.AttestationMode != nil || req.AttestationTTLSec != nil || req.DmrEnabled != nil ||
		req.CIDRBlockEnabled != nil || req.ProxyVPNBlockEnabled != nil || req.ModeratorIntelEnabled != nil ||
		req.ReviewTrafficAction != nil ||
		req.TLSFingerprintBlockEnabled != nil || req.ConnTypePolicy != nil ||
		req.LinkSigningEnabled != nil || req.LinkSigningTTLSec != nil ||
		req.ClickDelivery != nil || req.ProxyUpstreamURL != nil || req.ProxyRewriteAssets != nil
	if !adminPatch && budgetMicro == nil && !statusSet && !schedulePatch && !clickPresetPatch {
		return getCampaign(ctx, pool, fx, campaignID)
	}

	var updated db.Campaign
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}

		if budgetMicro != nil {
			if err := fx.ApplyCampaignBudgetPatch(ctx, q, locked, *budgetMicro); err != nil {
				return err
			}
			locked, err = q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
			if err != nil {
				return err
			}
		}

		if schedulePatch {
			startAt := timestamptzPtr(locked.StartAt)
			endAt := timestamptzPtr(locked.EndAt)
			if req.StartAt != nil {
				startAt = req.StartAt
			}
			if req.EndAt != nil {
				endAt = req.EndAt
			}
			daypart := locked.DaypartHours
			if req.DaypartHours != nil {
				daypart = req.DaypartHours
			}
			if err := fx.ApplyCampaignSchedulePatch(ctx, q, campaignID, locked, startAt, endAt, daypart); err != nil {
				return err
			}
			locked, err = q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
			if err != nil {
				return err
			}
		}

		if adminPatch {
			name := locked.Name
			if req.Name != nil {
				name = strings.TrimSpace(*req.Name)
				if name == "" {
					return fmt.Errorf("name is required")
				}
			}
			dailyBudget := locked.DailyBudget
			if req.DailyBudgetMicro != nil {
				if *req.DailyBudgetMicro < 0 {
					return fmt.Errorf("invalid daily_budget")
				}
				dailyBudget = *req.DailyBudgetMicro
			}
			timezone := locked.Timezone
			if req.Timezone != nil {
				timezone = strings.TrimSpace(*req.Timezone)
				if timezone == "" {
					timezone = "UTC"
				}
			}
			freqLimit := locked.FreqLimit
			if req.FreqLimit != nil {
				freqLimit = pgtype.Int4{Int32: *req.FreqLimit, Valid: true}
			}
			freqWindow := locked.FreqWindow
			if req.FreqWindow != nil {
				freqWindow = pgtype.Int4{Int32: *req.FreqWindow, Valid: true}
			}
			countries := locked.TargetCountries
			if req.TargetCountries != nil {
				countries = countriesOrEmpty(req.TargetCountries)
			}
			targetURL := locked.TargetUrl
			if req.TargetURL != nil {
				targetURL = *req.TargetURL
			}
			referrerFilter := locked.ReferrerFilter
			if req.ReferrerFilter != nil {
				referrerFilter = *req.ReferrerFilter
			}
			safePageURL := locked.SafePageUrl
			if req.SafePageURL != nil {
				safePageURL = *req.SafePageURL
			}
			safePageEnabled := locked.SafePageEnabled
			if req.SafePageEnabled != nil {
				safePageEnabled = *req.SafePageEnabled
			}
			attestationEnabled := locked.AttestationEnabled
			if req.AttestationEnabled != nil {
				attestationEnabled = *req.AttestationEnabled
			}
			attestationMode := locked.AttestationMode
			if req.AttestationMode != nil {
				parsedMode, _, err := parsePatchAttestationMode(req.AttestationMode)
				if err != nil {
					return err
				}
				attestationMode = string(parsedMode)
			}
			if safePageEnabled && !locked.SafePageEnabled && req.AttestationMode == nil && req.AttestationEnabled == nil {
				attestationMode = string(domain.AttestationModeLight)
			}
			resolvedMode := domain.ResolveAttestationMode(domain.ParseAttestationMode(attestationMode), attestationEnabled)
			attestationMode = string(resolvedMode)
			attestationEnabled = resolvedMode.RequiresProbe()
			attestationTTL := locked.AttestationTtlSec
			if req.AttestationTTLSec != nil {
				parsed, _, err := parsePatchAttestationTTLSec(req.AttestationTTLSec)
				if err != nil {
					return err
				}
				attestationTTL = parsed
			}
			if attestationEnabled && !safePageEnabled {
				return fmt.Errorf("attestation_enabled requires safe_page_enabled")
			}
			if resolvedMode.RequiresProbe() && !safePageEnabled {
				return fmt.Errorf("attestation_mode requires safe_page_enabled")
			}
			dmrEnabled := locked.DmrEnabled
			if req.DmrEnabled != nil {
				dmrEnabled = *req.DmrEnabled
			}
			cidrBlock := locked.CidrBlockEnabled
			if req.CIDRBlockEnabled != nil {
				cidrBlock = *req.CIDRBlockEnabled
			}
			proxyVPNBlock := locked.ProxyVpnBlockEnabled
			if req.ProxyVPNBlockEnabled != nil {
				proxyVPNBlock = *req.ProxyVPNBlockEnabled
			}
			moderatorIntel := locked.ModeratorIntelEnabled
			if req.ModeratorIntelEnabled != nil {
				moderatorIntel = *req.ModeratorIntelEnabled
			}
			reviewTrafficAction := locked.ReviewTrafficAction
			if req.ReviewTrafficAction != nil {
				parsed := domain.ParseReviewTrafficAction(*req.ReviewTrafficAction)
				if !parsed.Valid() {
					return fmt.Errorf("invalid review_traffic_action")
				}
				reviewTrafficAction = string(parsed)
			}
			tlsFingerprintBlock := locked.TlsFingerprintBlockEnabled
			if req.TLSFingerprintBlockEnabled != nil {
				tlsFingerprintBlock = *req.TLSFingerprintBlockEnabled
			}
			connTypePolicy := locked.ConnTypePolicy
			if req.ConnTypePolicy != nil {
				parsed, _, err := parsePatchConnTypePolicy(req.ConnTypePolicy)
				if err != nil {
					return err
				}
				connTypePolicy = parsed
			}
			linkSigningEnabled := locked.LinkSigningEnabled
			if req.LinkSigningEnabled != nil {
				linkSigningEnabled = *req.LinkSigningEnabled
			}
			linkSigningTTL := locked.LinkSigningTtlSec
			if req.LinkSigningTTLSec != nil {
				parsed, _, err := parsePatchLinkSigningTTLSec(req.LinkSigningTTLSec)
				if err != nil {
					return err
				}
				linkSigningTTL = parsed
			}
			clickDelivery := locked.ClickDelivery
			if req.ClickDelivery != nil {
				clickDelivery = strings.TrimSpace(*req.ClickDelivery)
			}
			if clickDelivery == "" {
				clickDelivery = proxyupstream.ClickDeliveryRedirect
			}
			proxyUpstream := locked.ProxyUpstreamUrl
			if req.ProxyUpstreamURL != nil {
				proxyUpstream = strings.TrimSpace(*req.ProxyUpstreamURL)
			}
			proxyRewrite := locked.ProxyRewriteAssets
			if req.ProxyRewriteAssets != nil {
				proxyRewrite = *req.ProxyRewriteAssets
			}
			allowHTTP := fx.ProxyAllowHTTPInsecure()
			if err := proxyupstream.ValidateDeliveryPair(ctx, clickDelivery, proxyUpstream, allowHTTP); err != nil {
				return err
			}

			locked, err = q.UpdateCampaignAdmin(ctx, db.UpdateCampaignAdminParams{
				ID:                         domain.ToUUID(campaignID),
				Name:                       name,
				DailyBudget:                dailyBudget,
				Timezone:                   timezone,
				FreqLimit:                  freqLimit,
				FreqWindow:                 freqWindow,
				TargetCountries:            countries,
				TargetUrl:                  targetURL,
				ReferrerFilter:             referrerFilter,
				SafePageUrl:                safePageURL,
				SafePageEnabled:            safePageEnabled,
				AttestationEnabled:         attestationEnabled,
				AttestationTtlSec:          attestationTTL,
				AttestationMode:            attestationMode,
				DmrEnabled:                 dmrEnabled,
				ClickDelivery:              clickDelivery,
				ProxyUpstreamUrl:           proxyUpstream,
				ProxyRewriteAssets:         proxyRewrite,
				TlsFingerprintBlockEnabled: tlsFingerprintBlock,
				ConnTypePolicy:             connTypePolicy,
				LinkSigningEnabled:         linkSigningEnabled,
				LinkSigningTtlSec:          linkSigningTTL,
				CidrBlockEnabled:           cidrBlock,
				ProxyVpnBlockEnabled:       proxyVPNBlock,
				ModeratorIntelEnabled:      moderatorIntel,
				ReviewTrafficAction:        reviewTrafficAction,
			})
			if err != nil {
				return err
			}

			var uid uuid.UUID
			if u, ok := authz.GetUser(ctx); ok {
				uid = u.UserID
			}
			fx.AuditLog(ctx, q, uid, "PATCH_CAMPAIGN", "campaign", &campaignID, map[string]any{
				"name":             name,
				"daily_budget":     dailyBudget,
				"timezone":         timezone,
				"target_countries": countries,
			}, nil)
		}

		if statusSet {
			if err := fx.ApplyCampaignStatusPatch(ctx, q, locked, statusWant, "patch", req.PublishForce); err != nil {
				return err
			}
		}

		updated, err = q.GetCampaign(ctx, domain.ToUUID(campaignID))
		return err
	})
	if err != nil {
		return CampaignDTO{}, err
	}

	fx.PublishCampaignUpdate(ctx, campaignID.String())
	return scrubCampaignDTO(ctx, updated), nil
}
