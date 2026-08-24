package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func applyCampaignAttestation(camp *Campaign, mode string, enabled bool, ttl int32) {
	if camp == nil {
		return
	}
	camp.AttestationEnabled = enabled
	camp.AttestationTTLSec = ttl
	camp.AttestationMode = ResolveAttestationMode(ParseAttestationMode(mode), enabled)
}

func CampaignFromDBRow(row db.Campaign) *Campaign {
	id := uuid.UUID(row.ID.Bytes)
	customerID := uuid.UUID(row.CustomerID.Bytes)

	loc, err := time.LoadLocation(row.Timezone)
	if err != nil {
		loc = time.UTC
	}

	var brandIDPtr *uuid.UUID
	if row.BrandID.Valid {
		brandID := uuid.UUID(row.BrandID.Bytes)
		brandIDPtr = &brandID
	}

	idStr := id.String()
	customerIDStr := customerID.String()
	dailyBudgetMicro := row.DailyBudget

	fcapPrefix := fcapKeyPrefix(id, row.BrandFcapKey)

	camp := &Campaign{
		ID:                      id,
		IDStr:                   idStr,
		IDStrAny:                idStr,
		CustomerID:              customerID,
		CustomerIDStr:           customerIDStr,
		CustomerIDStrAny:        customerIDStr,
		BrandID:                 brandIDPtr,
		BrandFcapKey:            row.BrandFcapKey,
		Name:                    row.Name,
		Status:                  CampaignStatus(row.Status),
		PacingMode:              PacingMode(row.PacingMode),
		BudgetLimit:             row.BudgetLimit,
		CurrentSpend:            row.CurrentSpend,
		DailyBudget:             row.DailyBudget,
		DailyBudgetMicro:        dailyBudgetMicro,
		DailyBudgetMicroAny:     dailyBudgetMicro,
		Timezone:                row.Timezone,
		Location:                loc,
		FreqLimit:               row.FreqLimit.Int32,
		FreqLimitAny:            row.FreqLimit.Int32,
		FreqWindow:              row.FreqWindow.Int32,
		FreqWindowAny:           row.FreqWindow.Int32,
		TargetCountries:         SliceToMap(row.TargetCountries),
		BudgetCampaignKey:       budgetCampaignKey(id),
		CampaignSyncKey:         campaignSyncKey(id),
		CustomerSyncKey:         customerSyncKey(id, customerID),
		FcapKeyPrefix:           fcapPrefix,
		DailySpendKeyPrefix:     dailySpendKeyPrefix(id),
		FraudThresholdPass:      uint8(row.FraudThresholdPass),
		FraudThresholdSuspect:   uint8(row.FraudThresholdSuspect),
		FraudThresholdIVT:       uint8(row.FraudThresholdIvt),
		FraudThresholdBlock:     uint8(row.FraudThresholdBlock),
		SilentRejectEnabled:     row.SilentRejectEnabled,
		BehaviorFlags:           BehaviorFlags(row.BehaviorFlags),
		RequireConsentPurposes:  row.RequireConsentPurposes,
		MigrationGen:            row.MigrationGen,
		SafePageURL:             row.SafePageUrl,
		SafePageEnabled:         row.SafePageEnabled,
		DmrEnabled:              row.DmrEnabled,
		L1CIDRBlockEnabled:      row.L1CidrBlockEnabled,
		L15ProxyVPNBlockEnabled: row.L15ProxyVpnBlockEnabled,
		SocialInAppEnabled:      row.SocialInAppEnabled,
		ClickDelivery:           row.ClickDelivery,
		ProxyUpstreamURL:        row.ProxyUpstreamUrl,
		ProxyRewriteAssets:      row.ProxyRewriteAssets,
	}
	applyCampaignAttestation(camp, row.AttestationMode, row.AttestationEnabled, row.AttestationTtlSec)
	applyCampaignLandingProtectionFields(camp, row.TlsFingerprintBlockEnabled, row.ConnTypePolicy, row.LinkSigningEnabled, row.LinkSigningTtlSec)
	applyCampaignSegmentFields(camp, row.RetargetSegmentID, row.SegmentInclude, row.SegmentExclude, row.SegmentTtlHours)
	return camp
}

func CampaignFromGetCampaignFullRow(row db.GetCampaignFullRow) *Campaign {
	id := uuid.UUID(row.ID.Bytes)
	customerID := uuid.UUID(row.CustomerID.Bytes)

	loc, err := time.LoadLocation(row.Timezone)
	if err != nil {
		loc = time.UTC
	}

	var brandIDPtr *uuid.UUID
	if row.BrandID.Valid {
		brandID := uuid.UUID(row.BrandID.Bytes)
		brandIDPtr = &brandID
	}

	idStr := id.String()
	customerIDStr := customerID.String()
	dailyBudgetMicro := row.DailyBudget

	fcapPrefix := fcapKeyPrefix(id, row.BrandFcapKey)

	camp := &Campaign{
		ID:                      id,
		IDStr:                   idStr,
		IDStrAny:                idStr,
		CustomerID:              customerID,
		CustomerIDStr:           customerIDStr,
		CustomerIDStrAny:        customerIDStr,
		BrandID:                 brandIDPtr,
		BrandFcapKey:            row.BrandFcapKey,
		Name:                    row.Name,
		Status:                  CampaignStatus(row.Status),
		PacingMode:              PacingMode(row.PacingMode),
		BudgetLimit:             row.BudgetLimit,
		CurrentSpend:            row.CurrentSpend,
		DailyBudget:             row.DailyBudget,
		DailyBudgetMicro:        dailyBudgetMicro,
		DailyBudgetMicroAny:     dailyBudgetMicro,
		Timezone:                row.Timezone,
		Location:                loc,
		FreqLimit:               row.FreqLimit.Int32,
		FreqLimitAny:            row.FreqLimit.Int32,
		FreqWindow:              row.FreqWindow.Int32,
		FreqWindowAny:           row.FreqWindow.Int32,
		TargetCountries:         SliceToMap(row.TargetCountries),
		BudgetCampaignKey:       budgetCampaignKey(id),
		CampaignSyncKey:         campaignSyncKey(id),
		CustomerSyncKey:         customerSyncKey(id, customerID),
		FcapKeyPrefix:           fcapPrefix,
		DailySpendKeyPrefix:     dailySpendKeyPrefix(id),
		FraudThresholdPass:      uint8(row.FraudThresholdPass),
		FraudThresholdSuspect:   uint8(row.FraudThresholdSuspect),
		FraudThresholdIVT:       uint8(row.FraudThresholdIvt),
		FraudThresholdBlock:     uint8(row.FraudThresholdBlock),
		SilentRejectEnabled:     row.SilentRejectEnabled,
		BehaviorFlags:           BehaviorFlags(row.BehaviorFlags),
		RequireConsentPurposes:  row.RequireConsentPurposes,
		MigrationGen:            row.MigrationGen,
		SafePageURL:             row.SafePageUrl,
		SafePageEnabled:         row.SafePageEnabled,
		DmrEnabled:              row.DmrEnabled,
		L1CIDRBlockEnabled:      row.L1CidrBlockEnabled,
		L15ProxyVPNBlockEnabled: row.L15ProxyVpnBlockEnabled,
		SocialInAppEnabled:      row.SocialInAppEnabled,
		ClickDelivery:           row.ClickDelivery,
		ProxyUpstreamURL:        row.ProxyUpstreamUrl,
		ProxyRewriteAssets:      row.ProxyRewriteAssets,
	}
	applyCampaignAttestation(camp, row.AttestationMode, row.AttestationEnabled, row.AttestationTtlSec)
	applyCampaignLandingProtectionFields(camp, row.TlsFingerprintBlockEnabled, row.ConnTypePolicy, row.LinkSigningEnabled, row.LinkSigningTtlSec)
	applyCampaignSegmentFields(camp, row.RetargetSegmentID, row.SegmentInclude, row.SegmentExclude, row.SegmentTtlHours)

	if row.PrimaryAShard.Valid {
		camp.HasTriplet = true
		camp.PrimaryAShard = row.PrimaryAShard.Int16
		camp.PrimaryBShard = row.PrimaryBShard.Int16
		camp.ReserveShard = row.ReserveShard.Int16
		camp.HEma = row.HEma.Float64
		camp.CEma = row.CEma.Float64
	}
	if row.RoutingEpoch.Valid {
		camp.RoutingEpoch = row.RoutingEpoch.Int64
	}

	return camp
}

func CampaignFromListActiveCampaignsRow(row db.ListActiveCampaignsRow) *Campaign {
	id := uuid.UUID(row.ID.Bytes)
	customerID := uuid.UUID(row.CustomerID.Bytes)

	loc, err := time.LoadLocation(row.Timezone)
	if err != nil {
		loc = time.UTC
	}

	var brandIDPtr *uuid.UUID
	if row.BrandID.Valid {
		brandID := uuid.UUID(row.BrandID.Bytes)
		brandIDPtr = &brandID
	}

	idStr := id.String()
	customerIDStr := customerID.String()
	dailyBudgetMicro := row.DailyBudget

	fcapPrefix := fcapKeyPrefix(id, row.BrandFcapKey)

	camp := &Campaign{
		ID:                      id,
		IDStr:                   idStr,
		IDStrAny:                idStr,
		CustomerID:              customerID,
		CustomerIDStr:           customerIDStr,
		CustomerIDStrAny:        customerIDStr,
		BrandID:                 brandIDPtr,
		BrandFcapKey:            row.BrandFcapKey,
		Name:                    row.Name,
		Status:                  CampaignStatus(row.Status),
		PacingMode:              PacingMode(row.PacingMode),
		BudgetLimit:             row.BudgetLimit,
		CurrentSpend:            row.CurrentSpend,
		DailyBudget:             row.DailyBudget,
		DailyBudgetMicro:        dailyBudgetMicro,
		DailyBudgetMicroAny:     dailyBudgetMicro,
		ReserveMicro:            row.ReserveMicro,
		Timezone:                row.Timezone,
		Location:                loc,
		FreqLimit:               row.FreqLimit.Int32,
		FreqLimitAny:            row.FreqLimit.Int32,
		FreqWindow:              row.FreqWindow.Int32,
		FreqWindowAny:           row.FreqWindow.Int32,
		TargetCountries:         SliceToMap(row.TargetCountries),
		BudgetCampaignKey:       budgetCampaignKey(id),
		CampaignSyncKey:         campaignSyncKey(id),
		CustomerSyncKey:         customerSyncKey(id, customerID),
		FcapKeyPrefix:           fcapPrefix,
		DailySpendKeyPrefix:     dailySpendKeyPrefix(id),
		FraudThresholdPass:      uint8(row.FraudThresholdPass),
		FraudThresholdSuspect:   uint8(row.FraudThresholdSuspect),
		FraudThresholdIVT:       uint8(row.FraudThresholdIvt),
		FraudThresholdBlock:     uint8(row.FraudThresholdBlock),
		SilentRejectEnabled:     row.SilentRejectEnabled,
		BehaviorFlags:           BehaviorFlags(row.BehaviorFlags),
		RequireConsentPurposes:  row.RequireConsentPurposes,
		MigrationGen:            row.MigrationGen,
		SafePageURL:             row.SafePageUrl,
		SafePageEnabled:         row.SafePageEnabled,
		DmrEnabled:              row.DmrEnabled,
		L1CIDRBlockEnabled:      row.L1CidrBlockEnabled,
		L15ProxyVPNBlockEnabled: row.L15ProxyVpnBlockEnabled,
		SocialInAppEnabled:      row.SocialInAppEnabled,
		ClickDelivery:           row.ClickDelivery,
		ProxyUpstreamURL:        row.ProxyUpstreamUrl,
		ProxyRewriteAssets:      row.ProxyRewriteAssets,
	}
	applyCampaignAttestation(camp, row.AttestationMode, row.AttestationEnabled, row.AttestationTtlSec)
	applyCampaignLandingProtectionFields(camp, row.TlsFingerprintBlockEnabled, row.ConnTypePolicy, row.LinkSigningEnabled, row.LinkSigningTtlSec)
	applyCampaignSegmentFields(camp, row.RetargetSegmentID, row.SegmentInclude, row.SegmentExclude, row.SegmentTtlHours)

	if row.PrimaryAShard.Valid {
		camp.HasTriplet = true
		camp.PrimaryAShard = row.PrimaryAShard.Int16
		camp.PrimaryBShard = row.PrimaryBShard.Int16
		camp.ReserveShard = row.ReserveShard.Int16
		camp.HEma = row.HEma.Float64
		camp.CEma = row.CEma.Float64
	}
	if row.RoutingEpoch.Valid {
		camp.RoutingEpoch = row.RoutingEpoch.Int64
	}

	return camp
}

type dbQuerier struct {
	db.Querier
	dbtx db.DBTX
}

func (q *dbQuerier) DB() db.DBTX {
	return q.dbtx
}

func NewCampaignRepoWithDB(dbtx db.DBTX, queries db.Querier) *CampaignRepo {
	return &CampaignRepo{queries: &dbQuerier{Querier: queries, dbtx: dbtx}}
}

type CampaignRepo struct {
	queries                 db.Querier
	auditLedgerFlushSeq     atomic.Uint64
	auditLedgerFlushEnabled bool
	auditLedgerFlushMask    uint64
}

func NewCampaignRepo(queries db.Querier) *CampaignRepo {
	return &CampaignRepo{queries: queries}
}

func (r *CampaignRepo) SetDB(dbtx db.DBTX, queries db.Querier) {
	if r == nil || queries == nil {
		return
	}
	r.queries = &dbQuerier{Querier: queries, dbtx: dbtx}
}

func (r *CampaignRepo) ConfigureAuditLedgerFlush(cfgVal int) {
	if cfgVal < 0 {
		r.auditLedgerFlushEnabled = false
		return
	}
	r.auditLedgerFlushEnabled = true
	r.auditLedgerFlushMask = histogramSampleMaskFromConfig(cfgVal)
}

func (r *CampaignRepo) GetByID(ctx context.Context, id uuid.UUID) (*Campaign, error) {
	row, err := r.queries.GetCampaignFull(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, err
	}
	return CampaignFromGetCampaignFullRow(row), nil
}

func (r *CampaignRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status CampaignStatus) error {
	_, err := r.queries.UpdateCampaignStatus(ctx, db.UpdateCampaignStatusParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		Status: db.CampaignStatusType(status),
	})
	return err
}

func (r *CampaignRepo) UpdateSpend(ctx context.Context, id uuid.UUID, amount int64, txID string) error {
	outcomes, err := r.UpdateSpendBatch(ctx, []SpendFlushItem{{
		CampaignID:  id,
		AmountMicro: amount,
		TxID:        txID,
	}})
	if err != nil {
		return err
	}
	if len(outcomes) > 0 && outcomes[0].Err != nil {
		return outcomes[0].Err
	}
	return nil
}

func (r *CampaignRepo) UpdateSpendBatch(ctx context.Context, items []SpendFlushItem) ([]SpendFlushOutcome, error) {
	if len(items) == 0 {
		return nil, nil
	}

	outcomes := make([]SpendFlushOutcome, len(items))
	for i, item := range items {
		outcomes[i] = SpendFlushOutcome{CampaignID: item.CampaignID}
	}

	dbtx := r.resolveDBTX()
	if dbtx == nil {
		for i, item := range items {
			outcomes[i].Err = r.queries.UpdateCampaignSpend(ctx, db.UpdateCampaignSpendParams{
				ID:           pgtype.UUID{Bytes: item.CampaignID, Valid: true},
				CurrentSpend: item.AmountMicro,
			})
		}
		return outcomes, nil
	}

	tx, err := beginDBTX(ctx, dbtx)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		defer func() { _ = tx.Rollback(ctx) }()
	}

	exec := dbtx
	if tx != nil {
		exec = tx
	}
	q := r.querierInTx(tx)
	sharder := NewStaticSlotSharder(config.ExpectedRedisShardCount)

	for i, item := range items {
		err := r.applySpendFlush(ctx, exec, q, sharder, item, i)
		if errors.Is(err, ErrInsufficientCustomerBalance) || errors.Is(err, ErrCampaignSpendSkipped) {
			outcomes[i].Err = err
			continue
		}
		if err != nil {
			return nil, err
		}
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
	}
	return outcomes, nil
}

func (r *CampaignRepo) resolveDBTX() db.DBTX {
	getter, ok := r.queries.(interface{ DB() db.DBTX })
	if !ok {
		return nil
	}
	return getter.DB()
}

func beginDBTX(ctx context.Context, dbtx db.DBTX) (pgx.Tx, error) {
	beginner, ok := dbtx.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return nil, nil
	}
	return beginner.Begin(ctx)
}

func (r *CampaignRepo) querierInTx(tx pgx.Tx) db.Querier {
	switch v := r.queries.(type) {
	case *db.Queries:
		if tx != nil {
			return v.WithTx(tx)
		}
		return v
	case *dbQuerier:
		if inner, ok := v.Querier.(*db.Queries); ok && tx != nil {
			return &dbQuerier{Querier: inner.WithTx(tx), dbtx: tx}
		}
		return v
	default:
		return r.queries
	}
}

func (r *CampaignRepo) applySpendFlush(
	ctx context.Context,
	exec db.DBTX,
	q db.Querier,
	sharder *StaticSlotSharder,
	item SpendFlushItem,
	batchIdx int,
) error {
	savepoint := "lf_" + strconv.Itoa(batchIdx)
	if _, err := exec.Exec(ctx, "SAVEPOINT "+savepoint); err != nil {
		return err
	}

	tag, err := exec.Exec(ctx, "INSERT INTO sync_idempotency (id) VALUES ($1) ON CONFLICT DO NOTHING", item.TxID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, _ = exec.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint)
		return nil
	}

	budget, err := r.lockCampaignBudgetForFlush(ctx, exec, item)
	if errors.Is(err, ErrCampaignSpendSkipped) {
		_, _ = exec.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint)
		return err
	}
	if err != nil {
		return err
	}
	if budget.CustomerBalance < item.AmountMicro {
		_, _ = exec.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint)
		return ErrInsufficientCustomerBalance
	}

	_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
		CustomerID:      budget.CustomerID,
		CampaignID:      pgtype.UUID{Bytes: item.CampaignID, Valid: true},
		Amount:          -item.AmountMicro,
		Type:            db.LedgerTypeFEE,
		IdempotencyHash: pgtype.Text{String: ledgerBatchHash(item.TxID), Valid: true},
		PaymentIntentID: pgtype.UUID{},
	})
	if err != nil {
		return err
	}

	err = q.UpdateCampaignSpend(ctx, db.UpdateCampaignSpendParams{
		ID:           pgtype.UUID{Bytes: item.CampaignID, Valid: true},
		CurrentSpend: item.AmountMicro,
	})
	if err != nil {
		return err
	}

	if item.RtbCostMicro > 0 {
		if err := WriteMarginEconomicsLegs(ctx, q, budget.CustomerID, item.CampaignID, item.TxID, item.AmountMicro, item.RtbCostMicro); err != nil {
			return err
		}
	}

	if r.auditLedgerFlushEnabled && shouldSampleHistogram(r.auditLedgerFlushSeq.Add(1), r.auditLedgerFlushMask) {
		changes, _ := json.Marshal(map[string]any{
			"amount_micro": item.AmountMicro,
			"tx_id":        item.TxID,
		})
		metadata, _ := json.Marshal(map[string]any{"source": "sync_worker"})
		_, err = q.CreateAuditLog(ctx, db.CreateAuditLogParams{
			AdminID:    pgtype.UUID{Bytes: uuid.Nil, Valid: true},
			Action:     "LEDGER_BATCH_FLUSH",
			TargetType: "campaign",
			TargetID:   pgtype.UUID{Bytes: item.CampaignID, Valid: true},
			Changes:    changes,
			Metadata:   metadata,
		})
		if err != nil {
			return err
		}
	}

	shardID := int16(sharder.GetShard(item.CampaignID))
	_ = q.DecreaseCampaignQuotaReserved(ctx, db.DecreaseCampaignQuotaReservedParams{
		ShardID:        shardID,
		CampaignID:     pgtype.UUID{Bytes: item.CampaignID, Valid: true},
		ReservedAmount: item.AmountMicro,
	})

	_, err = exec.Exec(ctx, "RELEASE SAVEPOINT "+savepoint)
	return err
}

func (r *CampaignRepo) lockCampaignBudgetForFlush(ctx context.Context, exec db.DBTX, item SpendFlushItem) (db.GetCampaignBudgetRow, error) {
	var row db.GetCampaignBudgetRow
	lockClause := "FOR UPDATE SKIP LOCKED"
	if item.StrictFlush {
		lockClause = "FOR UPDATE"
	}
	err := exec.QueryRow(ctx, `
		SELECT c.id, c.customer_id, c.budget_limit, c.current_spend, c.status, cust.balance AS customer_balance
		FROM campaigns c
		JOIN customers cust ON c.customer_id = cust.id
		WHERE c.id = $1
		`+lockClause,
		pgtype.UUID{Bytes: item.CampaignID, Valid: true},
	).Scan(&row.ID, &row.CustomerID, &row.BudgetLimit, &row.CurrentSpend, &row.Status, &row.CustomerBalance)
	if errors.Is(err, pgx.ErrNoRows) && !item.StrictFlush {
		return row, ErrCampaignSpendSkipped
	}
	return row, err
}

func (r *CampaignRepo) ListActive(ctx context.Context) ([]*Campaign, error) {
	rows, err := r.queries.ListActiveCampaigns(ctx)
	if err != nil {
		return nil, err
	}

	campaigns := make([]*Campaign, len(rows))
	for i, row := range rows {
		campaigns[i] = CampaignFromListActiveCampaignsRow(row)
	}
	return campaigns, nil
}

type CustomerRepo struct {
	queries db.Querier
}

func NewCustomerRepo(queries db.Querier) *CustomerRepo {
	return &CustomerRepo{queries: queries}
}

func NewCustomerRepoWithDB(dbtx db.DBTX, queries db.Querier) *CustomerRepo {
	return &CustomerRepo{queries: &dbQuerier{Querier: queries, dbtx: dbtx}}
}

func (r *CustomerRepo) SetDB(dbtx db.DBTX, queries db.Querier) {
	if r == nil || queries == nil {
		return
	}
	r.queries = &dbQuerier{Querier: queries, dbtx: dbtx}
}

func (r *CustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	row, err := r.queries.GetCustomerByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, err
	}

	return &Customer{
		ID:       id,
		Name:     row.Name,
		Balance:  row.Balance,
		Currency: row.Currency,
	}, nil
}

func (r *CustomerRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount int64, txID string) error {
	var dbtx db.DBTX
	if getter, ok := r.queries.(interface{ DB() db.DBTX }); ok {
		dbtx = getter.DB()
	}

	if dbtx == nil {
		return r.queries.UpdateCustomerBalance(ctx, db.UpdateCustomerBalanceParams{
			ID:      pgtype.UUID{Bytes: id, Valid: true},
			Balance: amount,
		})
	}

	var tx pgx.Tx
	var err error
	if beginner, ok := dbtx.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}); ok {
		tx, err = beginner.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
	}

	exec := dbtx
	if tx != nil {
		exec = tx
	}

	tag, err := exec.Exec(ctx, "INSERT INTO sync_idempotency (id) VALUES ($1) ON CONFLICT DO NOTHING", txID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return nil
	}

	q := r.queries
	if tx != nil {
		if concreteQueries, ok := r.queries.(*db.Queries); ok {
			q = concreteQueries.WithTx(tx)
		}
	}

	err = q.UpdateCustomerBalance(ctx, db.UpdateCustomerBalanceParams{
		ID:      pgtype.UUID{Bytes: id, Valid: true},
		Balance: amount,
	})
	if err != nil {
		return err
	}

	if tx != nil {
		return tx.Commit(ctx)
	}
	return nil
}
