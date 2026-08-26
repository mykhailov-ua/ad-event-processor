package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultCloneNameSuffix = " (copy)"

type CloneCampaignSpec struct {
	SourceID       uuid.UUID
	NamePrefix     string
	NameSuffix     string
	IdempotencyKey string
}

type CloneCampaignResult struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}

func cloneCampaignName(sourceName, prefix, suffix string) string {
	if prefix != "" {
		return prefix + sourceName
	}
	if suffix != "" {
		return sourceName + suffix
	}
	return sourceName + defaultCloneNameSuffix
}

func cloneBrandFcapKey(campaignID uuid.UUID, brandID pgtype.UUID) string {
	if brandID.Valid {
		return "fcap:b:" + uuid.UUID(brandID.Bytes).String()
	}
	return "fcap:c:" + campaignID.String()
}

func (s *Service) CloneCampaign(ctx context.Context, spec CloneCampaignSpec) (CloneCampaignResult, error) {
	if spec.SourceID == uuid.Nil {
		return CloneCampaignResult{}, errValidation("source campaign id is required")
	}
	if strings.TrimSpace(spec.IdempotencyKey) == "" {
		return CloneCampaignResult{}, errValidation("idempotency key is required")
	}

	var (
		newCampaignID uuid.UUID
		clonedName    string
		flowCloned    bool
	)
	err := pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: spec.IdempotencyKey, Valid: true})
		if err == nil {
			if existing.CampaignID.Valid {
				newCampaignID = uuid.UUID(existing.CampaignID.Bytes)
				row, err := q.GetCampaign(ctx, existing.CampaignID)
				if err != nil {
					return err
				}
				clonedName = row.Name
				return nil
			}
			return fmt.Errorf("%w ledger row for key %q", ErrIncompleteIdempotency, spec.IdempotencyKey)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency lookup failed: %w", err)
		}

		src, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(spec.SourceID))
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}
		if err := assertMediaBuyerCampaignAccess(ctx, src); err != nil {
			return err
		}
		if src.DeletedAt.Valid {
			return ErrCampaignNotFound
		}

		cust, err := q.GetCustomerForUpdate(ctx, src.CustomerID)
		if err != nil {
			return mapNotFound(err, ErrCustomerNotFound)
		}
		if cust.Balance+cust.AllowedOverdraft < src.BudgetLimit {
			return ErrInsufficientBalance
		}

		newCampaignID, err = uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate campaign id: %w", err)
		}
		clonedName = cloneCampaignName(src.Name, spec.NamePrefix, spec.NameSuffix)

		var clonedFlowID pgtype.UUID
		if src.FlowID.Valid {
			flowID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate flow id: %w", err)
			}
			if _, err := q.CloneFlowFromSource(ctx, db.CloneFlowFromSourceParams{
				ID:   domain.ToUUID(flowID),
				Name: " (copy)",
				ID_2: src.FlowID,
			}); err != nil {
				return fmt.Errorf("clone flow: %w", err)
			}
			clonedFlowID = domain.ToUUID(flowID)
			flowCloned = true
		}

		startAt := timestamptzPtr(src.StartAt)
		endAt := timestamptzPtr(src.EndAt)
		initialStatus := resolveScheduleStatus(time.Now(), startAt, endAt)

		if _, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      src.CustomerID,
			Balance: -src.BudgetLimit,
		}); err != nil {
			return err
		}

		_, err = q.InsertClonedCampaign(ctx, db.InsertClonedCampaignParams{
			ID:           domain.ToUUID(newCampaignID),
			Name:         clonedName,
			Status:       initialStatus,
			BrandFcapKey: cloneBrandFcapKey(newCampaignID, src.BrandID),
			FlowID:       clonedFlowID,
			ID_2:         src.ID,
		})
		if err != nil {
			return fmt.Errorf("insert cloned campaign: %w", err)
		}

		if _, err := q.ClonePostbackConfig(ctx, db.ClonePostbackConfigParams{
			CampaignID:   domain.ToUUID(newCampaignID),
			CampaignID_2: src.ID,
		}); err != nil {
			return fmt.Errorf("clone postback config: %w", err)
		}

		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      src.CustomerID,
			CampaignID:      domain.ToUUID(newCampaignID),
			Amount:          src.BudgetLimit,
			Type:            db.LedgerTypeFREEZE,
			IdempotencyHash: pgtype.Text{String: spec.IdempotencyKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err != nil {
			return err
		}

		if err := q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
			CampaignID: domain.ToUUID(newCampaignID),
			NewStatus:  initialStatus,
			Reason:     pgtype.Text{String: "Cloned from " + spec.SourceID.String(), Valid: true},
		}); err != nil {
			return err
		}

		s.AuditLog(ctx, q, uuid.Nil, "CLONE_CAMPAIGN", "campaign", &newCampaignID, auditCloneCampaignChange{
			SourceID: spec.SourceID.String(),
			Name:     clonedName,
		}, auditIdempotencyMeta{IdempotencyKey: spec.IdempotencyKey})

		return s.emitCampaignLifecycleOutbox(ctx, q, newCampaignID, initialStatus, src.BudgetLimit)
	})
	if err != nil {
		return CloneCampaignResult{}, err
	}

	_ = s.publishCampaignUpdate(ctx, newCampaignID.String())
	if flowCloned {
		_ = s.publishFlowReload(ctx)
	}

	return CloneCampaignResult{
		ID:       newCampaignID.String(),
		SourceID: spec.SourceID.String(),
		Name:     clonedName,
	}, nil
}

type auditCloneCampaignChange struct {
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}
