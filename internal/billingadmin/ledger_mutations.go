package billingadmin

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type pauseCampaignOutboxPayload struct {
	CampaignID string `json:"campaign_id"`
}

func (st *LedgerStore) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (found bool, entry db.BalanceLedger, refundTotal, chargebackTotal, reversalTotal int64, err error) {
	q := db.New(st.host.Pool())
	entry, err = q.GetLedgerByPaymentIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			entry = db.BalanceLedger{}
		} else {
			return false, db.BalanceLedger{}, 0, 0, 0, err
		}
	} else {
		found = true
	}
	refundTotal, err = q.SumPaymentRefundAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return false, db.BalanceLedger{}, 0, 0, 0, err
	}
	chargebackTotal, err = q.SumPaymentChargebackAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return false, db.BalanceLedger{}, 0, 0, 0, err
	}
	reversalTotal, err = q.SumPaymentChargebackReversalAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return false, db.BalanceLedger{}, 0, 0, 0, err
	}
	return found, entry, refundTotal, chargebackTotal, reversalTotal, nil
}

func (st *LedgerStore) GetLedgerEntries(ctx context.Context, paymentIntentIDs []uuid.UUID) (map[uuid.UUID]domain.PaymentLedgerEntry, error) {
	out := make(map[uuid.UUID]domain.PaymentLedgerEntry, len(paymentIntentIDs))
	if len(paymentIntentIDs) == 0 {
		return out, nil
	}
	pgIDs := make([]pgtype.UUID, len(paymentIntentIDs))
	for i, id := range paymentIntentIDs {
		pgIDs[i] = domain.ToUUID(id)
	}
	rows, err := db.New(st.host.Pool()).SumPaymentLedgerTotalsByIntentIDs(ctx, pgIDs)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		id, err := uuid.FromBytes(row.PaymentIntentID.Bytes[:])
		if err != nil {
			continue
		}
		entry := domain.PaymentLedgerEntry{
			Found:                        row.HasTopup,
			HasTopup:                     row.HasTopup,
			TopupAmountMicro:             row.TopupMicro,
			RefundTotalMicro:             row.RefundMicro,
			ChargebackTotalMicro:         row.ChargebackMicro,
			ChargebackReversalTotalMicro: row.ChargebackReversalMicro,
		}
		if row.TopupMicro != 0 || row.RefundMicro != 0 || row.ChargebackMicro != 0 || row.ChargebackReversalMicro != 0 {
			entry.Found = true
		}
		out[id] = entry
	}
	return out, nil
}

func (st *LedgerStore) UpdateOverdraft(ctx context.Context, id uuid.UUID, newOverdraft int64) error {
	return pgx.BeginFunc(ctx, st.host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		cust, err := q.GetCustomerForUpdate(ctx, domain.ToUUID(id))
		if err != nil {
			return fmt.Errorf("failed to fetch customer for overdraft update: %w", err)
		}

		prevOverdraft := cust.AllowedOverdraft
		if newOverdraft == prevOverdraft {
			return nil
		}

		if newOverdraft < prevOverdraft {
			availableLimit := cust.Balance + newOverdraft
			if availableLimit < 0 {
				camps, err := q.ListCampaigns(ctx, db.ListCampaignsParams{
					Limit:      10000,
					Offset:     0,
					CustomerID: domain.ToUUID(id),
					Status:     pgtype.Text{String: string(db.CampaignStatusTypeACTIVE), Valid: true},
				})
				if err != nil {
					return fmt.Errorf("failed to list active campaigns for overdraft decrease: %w", err)
				}

				for _, c := range camps {
					if availableLimit >= 0 {
						break
					}

					locked, err := q.GetCampaignForUpdate(ctx, c.ID)
					if err != nil {
						return fmt.Errorf("failed to lock campaign for overdraft suspend: %w", err)
					}
					if locked.Status != db.CampaignStatusTypeACTIVE {
						continue
					}

					_, err = q.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
						ID:     locked.ID,
						Status: db.CampaignStatusTypePAUSED,
					})
					if err != nil {
						return fmt.Errorf("failed to pause campaign: %w", err)
					}

					err = q.CreateStatusHistory(ctx, db.CreateStatusHistoryParams{
						CampaignID: locked.ID,
						OldStatus:  db.NullCampaignStatusType{CampaignStatusType: db.CampaignStatusTypeACTIVE, Valid: true},
						NewStatus:  db.CampaignStatusTypePAUSED,
						Reason:     pgtype.Text{String: "Overdraft reduced, campaign suspended", Valid: true},
					})
					if err != nil {
						return fmt.Errorf("failed to write status history: %w", err)
					}

					budgetLimit := locked.BudgetLimit
					currentSpend := locked.CurrentSpend
					remaining := budgetLimit - currentSpend
					if remaining < 0 {
						remaining = 0
					}

					if remaining > 0 {
						_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
							ID:      domain.ToUUID(id),
							Balance: remaining,
						})
						if err != nil {
							return fmt.Errorf("failed to refund balance for suspended campaign: %w", err)
						}

						_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
							CustomerID:      domain.ToUUID(id),
							CampaignID:      locked.ID,
							Amount:          remaining,
							Type:            db.LedgerTypeRELEASE,
							PaymentIntentID: pgtype.UUID{},
						})
						if err != nil {
							return fmt.Errorf("failed to record release ledger entry: %w", err)
						}

						availableLimit += remaining
					}

					payloadBytes, marshalErr := coldpath.MarshalOutbox(pauseCampaignOutboxPayload{CampaignID: uuid.UUID(locked.ID.Bytes).String()})
					if marshalErr != nil {
						return fmt.Errorf("marshal pause campaign outbox payload: %w", marshalErr)
					}
					_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
						EventType: "PAUSE_CAMPAIGN",
						Payload:   payloadBytes,
					})
					if err != nil {
						return fmt.Errorf("failed to emit outbox event for paused campaign: %w", err)
					}

					campID := uuid.UUID(locked.ID.Bytes)
					st.host.AuditLog(ctx, q, uuid.Nil, "SUSPEND_CAMPAIGN", "campaign", &campID, map[string]string{"reason": "overdraft_reduced"}, nil)
				}
			}
		}

		_, err = q.UpdateCustomerOverdraft(ctx, db.UpdateCustomerOverdraftParams{
			ID:               domain.ToUUID(id),
			AllowedOverdraft: newOverdraft,
		})
		if err != nil {
			return err
		}

		st.host.AuditLog(ctx, q, uuid.Nil, "UPDATE_CUSTOMER_OVERDRAFT", "customer", &id, map[string]string{
			"old_overdraft": money.FormatDecimal(prevOverdraft),
			"new_overdraft": money.FormatDecimal(newOverdraft),
		}, nil)
		return nil
	})
}

func (st *LedgerStore) TopUpBalance(ctx context.Context, customerID uuid.UUID, amount int64, idempotencyKey string) error {
	if err := st.host.RequirePgFencing(ctx); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, st.host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		_, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: idempotencyKey, Valid: true})
		if err == nil {
			return nil
		}
		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: amount,
		})
		if err != nil {
			return fmt.Errorf("failed to update balance: %w", err)
		}
		_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          amount,
			Type:            db.LedgerTypeTOPUP,
			IdempotencyHash: pgtype.Text{String: idempotencyKey, Valid: true},
			PaymentIntentID: pgtype.UUID{},
		})
		if err == nil {
			metrics.AddControlBalanceTopup("USD", money.APIValueFloat(amount))
			st.host.AuditLog(ctx, q, uuid.Nil, "TOPUP_BALANCE", "customer", &customerID, map[string]any{"amount": amount}, map[string]string{"idempotency_key": idempotencyKey})
		}
		return err
	})
}

func (st *LedgerStore) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amount int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	if err := st.host.RequirePgFencing(ctx); err != nil {
		return false, 0, err
	}
	var ledgerEntryID int64
	var applied bool

	err := pgx.BeginFunc(ctx, st.host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)

		existingPI, err := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID))
		if err == nil {
			ledgerEntryID = existingPI.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("payment intent idempotency check failed: %w", err)
		}

		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true})
		if err == nil {
			ledgerEntryID = existing.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("idempotency check failed: %w", err)
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: amount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return st.host.ErrCustomerNotFound()
			}
			return fmt.Errorf("failed to update balance: %w", err)
		}

		row, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          amount,
			Type:            db.LedgerType("PAYMENT_TOPUP"),
			IdempotencyHash: pgtype.Text{String: ledgerIdempotencyKey, Valid: true},
			PaymentIntentID: domain.ToUUID(paymentIntentID),
		})
		if err != nil {
			if st.host.IsPgUniqueViolation(err) {
				if existingPI, lookupErr := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID)); lookupErr == nil {
					ledgerEntryID = existingPI.ID
					applied = false
					return nil
				}
				if existing, lookupErr := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true}); lookupErr == nil {
					ledgerEntryID = existing.ID
					applied = false
					return nil
				}
			}
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}

		ledgerEntryID = row.ID
		applied = true

		metrics.AddControlBalanceTopup("USD", money.APIValueFloat(amount))
		st.host.AuditLog(ctx, q, uuid.Nil, "PAYMENT_SETTLEMENT", "customer", &customerID, map[string]any{
			"amount":            amount,
			"payment_intent_id": paymentIntentID.String(),
			"provider":          provider,
			"provider_ref":      providerRef,
		}, map[string]string{"idempotency_key": ledgerIdempotencyKey})
		return nil
	})

	return applied, ledgerEntryID, err
}

func (st *LedgerStore) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	if err := st.host.RequirePgFencing(ctx); err != nil {
		return false, 0, err
	}
	if amountMicro <= 0 {
		return false, 0, st.host.ErrValidation("refund amount must be positive")
	}

	var ledgerEntryID int64
	var applied bool

	err := pgx.BeginFunc(ctx, st.host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)

		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true})
		if err == nil {
			ledgerEntryID = existing.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("refund idempotency check failed: %w", err)
		}

		topup, err := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return st.host.ErrPaymentTopupNotFound()
			}
			return fmt.Errorf("payment topup lookup failed: %w", err)
		}

		refundedSoFar, err := q.SumPaymentRefundAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment refunds failed: %w", err)
		}
		if refundedSoFar+amountMicro > topup.Amount {
			return st.host.ErrRefundExceedsTopup()
		}

		debitAmount := -amountMicro
		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: debitAmount,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return st.host.ErrCustomerNotFound()
			}
			return fmt.Errorf("failed to debit balance: %w", err)
		}

		row, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          debitAmount,
			Type:            db.LedgerType("PAYMENT_REFUND"),
			IdempotencyHash: pgtype.Text{String: ledgerIdempotencyKey, Valid: true},
			PaymentIntentID: domain.ToUUID(paymentIntentID),
		})
		if err != nil {
			if st.host.IsPgUniqueViolation(err) {
				if existing, lookupErr := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true}); lookupErr == nil {
					ledgerEntryID = existing.ID
					applied = false
					return nil
				}
			}
			return fmt.Errorf("failed to create refund ledger entry: %w", err)
		}

		ledgerEntryID = row.ID
		applied = true

		st.host.AuditLog(ctx, q, uuid.Nil, "PAYMENT_REFUND", "customer", &customerID,
			map[string]any{
				"amount":             amountMicro,
				"payment_intent_id":  paymentIntentID.String(),
				"provider":           provider,
				"provider_refund_id": providerRefundID,
			},
			map[string]string{"idempotency_key": ledgerIdempotencyKey})
		return nil
	})

	return applied, ledgerEntryID, err
}

func (st *LedgerStore) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return st.applyPaymentChargebackMovement(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID, "PAYMENT_CHARGEBACK", true)
}

func (st *LedgerStore) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	return st.applyPaymentChargebackMovement(ctx, customerID, amountMicro, ledgerIdempotencyKey, paymentIntentID, provider, providerDisputeID, "PAYMENT_CHARGEBACK_REVERSAL", false)
}

func (st *LedgerStore) applyPaymentChargebackMovement(
	ctx context.Context,
	customerID uuid.UUID,
	amountMicro int64,
	ledgerIdempotencyKey string,
	paymentIntentID uuid.UUID,
	provider string,
	providerDisputeID string,
	ledgerType string,
	isDebit bool,
) (bool, int64, error) {
	if err := st.host.RequirePgFencing(ctx); err != nil {
		return false, 0, err
	}
	if amountMicro <= 0 {
		return false, 0, st.host.ErrValidation("chargeback amount must be positive")
	}

	var ledgerEntryID int64
	var applied bool

	err := pgx.BeginFunc(ctx, st.host.Pool(), func(tx pgx.Tx) error {
		q := db.New(tx)

		existing, err := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true})
		if err == nil {
			ledgerEntryID = existing.ID
			applied = false
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("chargeback idempotency check failed: %w", err)
		}

		topup, err := q.GetLedgerByPaymentIntentForUpdate(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return st.host.ErrPaymentTopupNotFound()
			}
			return fmt.Errorf("payment topup lookup failed: %w", err)
		}

		refundedSoFar, err := q.SumPaymentRefundAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment refunds failed: %w", err)
		}
		chargebackSoFar, err := q.SumPaymentChargebackAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment chargebacks failed: %w", err)
		}
		reversalSoFar, err := q.SumPaymentChargebackReversalAmountForIntent(ctx, domain.ToUUID(paymentIntentID))
		if err != nil {
			return fmt.Errorf("sum payment chargeback reversals failed: %w", err)
		}

		netChargeback := chargebackSoFar - reversalSoFar
		if isDebit {
			if refundedSoFar+netChargeback+amountMicro > topup.Amount {
				return st.host.ErrChargebackExceedsTopup()
			}
		} else if amountMicro > netChargeback {
			return st.host.ErrChargebackReversalExceedsWithdrawn()
		}

		balanceDelta := amountMicro
		ledgerAmount := amountMicro
		if isDebit {
			balanceDelta = -amountMicro
			ledgerAmount = -amountMicro
		}

		_, err = q.UpdateCustomerBalanceManagement(ctx, db.UpdateCustomerBalanceManagementParams{
			ID:      domain.ToUUID(customerID),
			Balance: balanceDelta,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return st.host.ErrCustomerNotFound()
			}
			return fmt.Errorf("failed to update balance for chargeback: %w", err)
		}

		row, err := q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
			CustomerID:      domain.ToUUID(customerID),
			Amount:          ledgerAmount,
			Type:            db.LedgerType(ledgerType),
			IdempotencyHash: pgtype.Text{String: ledgerIdempotencyKey, Valid: true},
			PaymentIntentID: domain.ToUUID(paymentIntentID),
		})
		if err != nil {
			if st.host.IsPgUniqueViolation(err) {
				if existing, lookupErr := q.GetLedgerByHashForUpdate(ctx, pgtype.Text{String: ledgerIdempotencyKey, Valid: true}); lookupErr == nil {
					ledgerEntryID = existing.ID
					applied = false
					return nil
				}
			}
			return fmt.Errorf("failed to create chargeback ledger entry: %w", err)
		}

		ledgerEntryID = row.ID
		applied = true

		action := "PAYMENT_CHARGEBACK"
		if !isDebit {
			action = "PAYMENT_CHARGEBACK_REVERSAL"
		}
		st.host.AuditLog(ctx, q, uuid.Nil, action, "customer", &customerID,
			map[string]any{
				"amount":              amountMicro,
				"payment_intent_id":   paymentIntentID.String(),
				"provider":            provider,
				"provider_dispute_id": providerDisputeID,
			},
			map[string]string{"idempotency_key": ledgerIdempotencyKey})
		return nil
	})

	return applied, ledgerEntryID, err
}
