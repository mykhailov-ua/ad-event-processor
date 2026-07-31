package controlplane

import (
	"context"
	"crypto/subtle"
	"errors"
	"espx/internal/controlplane/pb"
	db "espx/internal/domain/db"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (h *SettlementHandler) ApplyPaymentCredit(ctx context.Context, req *pb.ApplyPaymentCreditRequest) (*pb.ApplyPaymentCreditResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	params, err := settlementCreditParamsFromPB(req)
	if err != nil {
		return nil, err
	}
	applied, ledgerEntryID, err := h.applyPaymentCredit(ctx, params.CustomerID, params.AmountMicro, params.LedgerIdempotencyKey, params.PaymentIntentID, params.Provider, params.ProviderRef)
	if err != nil {
		return nil, mapPaymentCreditGRPCError(err)
	}
	return &pb.ApplyPaymentCreditResponse{Applied: applied, LedgerEntryId: ledgerEntryID}, nil
}

func (h *SettlementHandler) ApplyPaymentRefund(ctx context.Context, req *pb.ApplyPaymentRefundRequest) (*pb.ApplyPaymentRefundResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	params, err := settlementRefundParamsFromPB(req)
	if err != nil {
		return nil, err
	}
	applied, ledgerEntryID, err := h.applyPaymentRefund(ctx, params.CustomerID, params.AmountMicro, params.LedgerIdempotencyKey, params.PaymentIntentID, params.Provider, params.ProviderRefundID)
	if err != nil {
		return nil, mapPaymentRefundGRPCError(err)
	}
	return &pb.ApplyPaymentRefundResponse{Applied: applied, LedgerEntryId: ledgerEntryID}, nil
}

func (h *SettlementHandler) ApplyPaymentChargeback(ctx context.Context, req *pb.ApplyPaymentChargebackRequest) (*pb.ApplyPaymentChargebackResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	params, err := settlementChargebackParamsFromPB(req.GetCustomerId(), req.GetPaymentIntentId(), req.GetAmountMicro(), req.GetLedgerIdempotencyKey(), req.GetProvider(), req.GetProviderDisputeId())
	if err != nil {
		return nil, err
	}
	applied, ledgerEntryID, err := h.applyPaymentChargeback(ctx, params.CustomerID, params.AmountMicro, params.LedgerIdempotencyKey, params.PaymentIntentID, params.Provider, params.ProviderDisputeID)
	if err != nil {
		return nil, mapChargebackGRPCError(err)
	}
	return &pb.ApplyPaymentChargebackResponse{Applied: applied, LedgerEntryId: ledgerEntryID}, nil
}

func (h *SettlementHandler) ApplyPaymentChargebackReversal(ctx context.Context, req *pb.ApplyPaymentChargebackReversalRequest) (*pb.ApplyPaymentChargebackReversalResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	params, err := settlementChargebackParamsFromPB(req.GetCustomerId(), req.GetPaymentIntentId(), req.GetAmountMicro(), req.GetLedgerIdempotencyKey(), req.GetProvider(), req.GetProviderDisputeId())
	if err != nil {
		return nil, err
	}
	applied, ledgerEntryID, err := h.applyPaymentChargebackReversal(ctx, params.CustomerID, params.AmountMicro, params.LedgerIdempotencyKey, params.PaymentIntentID, params.Provider, params.ProviderDisputeID)
	if err != nil {
		return nil, mapChargebackGRPCError(err)
	}
	return &pb.ApplyPaymentChargebackReversalResponse{Applied: applied, LedgerEntryId: ledgerEntryID}, nil
}

func (h *SettlementHandler) GetLedgerEntry(ctx context.Context, req *pb.GetLedgerEntryRequest) (*pb.GetLedgerEntryResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	paymentIntentID, err := uuid.Parse(req.GetPaymentIntentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment intent id")
	}
	entry, topupRow, err := h.ledgerEntry(ctx, paymentIntentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load ledger entry: %v", err)
	}
	resp := &pb.GetLedgerEntryResponse{
		Found:                        entry.Found,
		RefundTotalMicro:             entry.RefundTotalMicro,
		ChargebackTotalMicro:         entry.ChargebackTotalMicro,
		ChargebackReversalTotalMicro: entry.ChargebackReversalTotalMicro,
	}
	if entry.Found {
		resp.Topup = ledgerEntryToPB(topupRow)
	}
	return resp, nil
}

func (h *SettlementHandler) BlockIP(ctx context.Context, req *pb.BlockIPRequest) (*pb.BlockIPResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	if req.GetIp() == "" {
		return nil, status.Error(codes.InvalidArgument, "ip required")
	}
	source := req.GetSource()
	if source == "" {
		source = "fraud"
	}
	if err := h.blockIP(ctx, req.GetIp(), source); err != nil {
		return nil, status.Errorf(codes.Internal, "block ip: %v", err)
	}
	return &pb.BlockIPResponse{Enqueued: true}, nil
}

func (h *SettlementHandler) EnqueueFraudThreat(ctx context.Context, req *pb.EnqueueFraudThreatRequest) (*pb.EnqueueFraudThreatResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	if req.GetIp() == "" {
		return nil, status.Error(codes.InvalidArgument, "ip required")
	}
	if req.GetCampaignId() == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id required")
	}
	payload := FraudThreatPayload{
		Action:     req.GetAction(),
		IP:         req.GetIp(),
		CampaignID: req.GetCampaignId(),
		Score:      req.GetScore(),
		Boost:      req.GetBoost(),
		TTLSeconds: req.GetTtlSeconds(),
	}
	if err := h.enqueueFraudThreat(ctx, payload); err != nil {
		return nil, status.Errorf(codes.Internal, "enqueue ml threat: %v", err)
	}
	return &pb.EnqueueFraudThreatResponse{Enqueued: true}, nil
}

func (h *SettlementHandler) BatchApplySettlement(ctx context.Context, req *pb.BatchApplySettlementRequest) (*pb.BatchApplySettlementResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	batch, err := settlementBatchParamsFromPB(req)
	if err != nil {
		return nil, err
	}
	result := h.batchApplySettlement(ctx, batch)
	return settlementBatchResultToPB(result), nil
}

func (h *SettlementHandler) ApplyCTVSettlement(ctx context.Context, req *pb.ApplyCTVSettlementRequest) (*pb.ApplyCTVSettlementResponse, error) {
	if err := h.requireSettlementToken(ctx); err != nil {
		return nil, err
	}
	if req.GetSettlementId() == "" || req.GetSpendMicro() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "settlement_id and positive spend_micro required")
	}
	customerID, err := uuid.Parse(req.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	campaignID, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid campaign id")
	}
	result, err := h.applyCTVSettlement(ctx, req.GetSettlementId(), customerID, campaignID, req.GetSpendMicro())
	if err != nil {
		if errors.Is(err, ErrCampaignNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "apply ctv settlement: %v", err)
	}
	return &pb.ApplyCTVSettlementResponse{
		Applied:     result.Applied,
		FeeLedgerId: result.FeeLedgerID,
		TaxLedgerId: result.TaxLedgerID,
		TaxMicro:    result.TaxMicro,
	}, nil
}

func settlementCreditParamsFromPB(req *pb.ApplyPaymentCreditRequest) (settlementCreditParams, error) {
	if req == nil {
		return settlementCreditParams{}, status.Error(codes.InvalidArgument, "request required")
	}
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return settlementCreditParams{}, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	paymentIntentID, err := uuid.Parse(req.PaymentIntentId)
	if err != nil {
		return settlementCreditParams{}, status.Error(codes.InvalidArgument, "invalid payment intent id")
	}
	return settlementCreditParams{
		CustomerID:           customerID,
		AmountMicro:          req.AmountMicro,
		LedgerIdempotencyKey: req.LedgerIdempotencyKey,
		PaymentIntentID:      paymentIntentID,
		Provider:             req.Provider,
		ProviderRef:          req.ProviderRef,
	}, nil
}

func settlementRefundParamsFromPB(req *pb.ApplyPaymentRefundRequest) (settlementRefundParams, error) {
	if req == nil {
		return settlementRefundParams{}, status.Error(codes.InvalidArgument, "request required")
	}
	customerID, err := uuid.Parse(req.CustomerId)
	if err != nil {
		return settlementRefundParams{}, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	paymentIntentID, err := uuid.Parse(req.PaymentIntentId)
	if err != nil {
		return settlementRefundParams{}, status.Error(codes.InvalidArgument, "invalid payment intent id")
	}
	if req.AmountMicro <= 0 {
		return settlementRefundParams{}, status.Error(codes.InvalidArgument, "amount_micro must be positive")
	}
	return settlementRefundParams{
		CustomerID:           customerID,
		AmountMicro:          req.AmountMicro,
		LedgerIdempotencyKey: req.LedgerIdempotencyKey,
		PaymentIntentID:      paymentIntentID,
		Provider:             req.Provider,
		ProviderRefundID:     req.ProviderRefundId,
	}, nil
}

func settlementChargebackParamsFromPB(customerIDStr, intentIDStr string, amountMicro int64, ledgerIdempotencyKey, provider, providerDisputeID string) (settlementChargebackParams, error) {
	customerID, paymentIntentID, err := parseSettlementCustomerAndIntent(customerIDStr, intentIDStr)
	if err != nil {
		return settlementChargebackParams{}, err
	}
	if amountMicro <= 0 {
		return settlementChargebackParams{}, status.Error(codes.InvalidArgument, "amount_micro must be positive")
	}
	return settlementChargebackParams{
		CustomerID:           customerID,
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: ledgerIdempotencyKey,
		PaymentIntentID:      paymentIntentID,
		Provider:             provider,
		ProviderDisputeID:    providerDisputeID,
	}, nil
}

func settlementBatchParamsFromPB(req *pb.BatchApplySettlementRequest) (settlementBatchParams, error) {
	total := len(req.Credits) + len(req.Refunds) + len(req.Chargebacks) + len(req.ChargebackReversals)
	if total == 0 {
		return settlementBatchParams{}, status.Error(codes.InvalidArgument, "batch empty")
	}
	if total > batchSettlementMaxItems {
		return settlementBatchParams{}, status.Errorf(codes.InvalidArgument, "batch exceeds %d items", batchSettlementMaxItems)
	}
	var batch settlementBatchParams
	for _, item := range req.Credits {
		params, err := settlementCreditParamsFromPB(item)
		if err != nil {
			return settlementBatchParams{}, err
		}
		batch.Credits = append(batch.Credits, params)
	}
	for _, item := range req.Refunds {
		params, err := settlementRefundParamsFromPB(item)
		if err != nil {
			return settlementBatchParams{}, err
		}
		batch.Refunds = append(batch.Refunds, params)
	}
	for _, item := range req.Chargebacks {
		params, err := settlementChargebackParamsFromPB(item.GetCustomerId(), item.GetPaymentIntentId(), item.GetAmountMicro(), item.GetLedgerIdempotencyKey(), item.GetProvider(), item.GetProviderDisputeId())
		if err != nil {
			return settlementBatchParams{}, err
		}
		batch.Chargebacks = append(batch.Chargebacks, params)
	}
	for _, item := range req.ChargebackReversals {
		params, err := settlementChargebackParamsFromPB(item.GetCustomerId(), item.GetPaymentIntentId(), item.GetAmountMicro(), item.GetLedgerIdempotencyKey(), item.GetProvider(), item.GetProviderDisputeId())
		if err != nil {
			return settlementBatchParams{}, err
		}
		batch.ChargebackReversals = append(batch.ChargebackReversals, params)
	}
	return batch, nil
}

func ledgerEntryToPB(entry db.BalanceLedger) *pb.LedgerEntry {
	campID := ""
	if entry.CampaignID.Valid {
		campID = uuid.UUID(entry.CampaignID.Bytes).String()
	}
	return &pb.LedgerEntry{
		Id:          entry.ID,
		CustomerId:  uuid.UUID(entry.CustomerID.Bytes).String(),
		CampaignId:  campID,
		AmountMicro: entry.Amount,
		Type:        string(entry.Type),
		CreatedAt:   entry.CreatedAt.Time.UTC().Format(time.RFC3339),
	}
}

func settlementBatchResultToPB(result settlementBatchResult) *pb.BatchApplySettlementResponse {
	return &pb.BatchApplySettlementResponse{
		CreditResults:             settlementBatchItemsToPB(result.CreditResults),
		RefundResults:             settlementBatchItemsToPB(result.RefundResults),
		ChargebackResults:         settlementBatchItemsToPB(result.ChargebackResults),
		ChargebackReversalResults: settlementBatchItemsToPB(result.ChargebackReversalResults),
	}
}

func settlementBatchItemsToPB(items []settlementBatchItemResult) []*pb.BatchSettlementItemResult {
	out := make([]*pb.BatchSettlementItemResult, 0, len(items))
	for _, item := range items {
		out = append(out, settlementBatchItemToPB(item))
	}
	return out
}

func settlementBatchItemToPB(item settlementBatchItemResult) *pb.BatchSettlementItemResult {
	if item.Err != nil {
		return &pb.BatchSettlementItemResult{Error: mapBatchItemGRPCError(item.Err)}
	}
	return &pb.BatchSettlementItemResult{Applied: item.Applied, LedgerEntryId: item.LedgerEntryID}
}

func mapBatchItemGRPCError(err error) string {
	if st, ok := status.FromError(err); ok {
		return st.Message()
	}
	switch {
	case errors.Is(err, ErrCustomerNotFound):
		return "customer not found"
	case errors.Is(err, ErrPaymentTopupNotFound):
		return "payment topup not found"
	case errors.Is(err, ErrRefundExceedsTopup):
		return "refund exceeds settled topup"
	case errors.Is(err, ErrChargebackExceedsTopup):
		return "chargeback exceeds settled topup"
	case errors.Is(err, ErrChargebackReversalExceedsWithdrawn):
		return "chargeback reversal exceeds withdrawn amount"
	default:
		return err.Error()
	}
}

func (h *SettlementHandler) requireSettlementToken(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}
	tokens := md.Get("x-internal-token")
	expectedToken := string(h.cfg.SettlementInternalToken)
	if expectedToken == "" {
		return status.Error(codes.FailedPrecondition, "settlement internal token not configured")
	}
	if len(tokens) == 0 || subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(expectedToken)) != 1 {
		return status.Error(codes.PermissionDenied, "invalid internal token")
	}
	return nil
}

func parseSettlementCustomerAndIntent(customerIDStr, intentIDStr string) (uuid.UUID, uuid.UUID, error) {
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	paymentIntentID, err := uuid.Parse(intentIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid payment intent id")
	}
	return customerID, paymentIntentID, nil
}

func mapPaymentCreditGRPCError(err error) error {
	if errors.Is(err, ErrCustomerNotFound) {
		return status.Error(codes.NotFound, "customer not found")
	}
	return status.Errorf(codes.Internal, "failed to apply payment credit: %v", err)
}

func mapPaymentRefundGRPCError(err error) error {
	if errors.Is(err, ErrCustomerNotFound) {
		return status.Error(codes.NotFound, "customer not found")
	}
	if errors.Is(err, ErrPaymentTopupNotFound) {
		return status.Error(codes.NotFound, "payment topup not found")
	}
	if errors.Is(err, ErrRefundExceedsTopup) {
		return status.Error(codes.FailedPrecondition, "refund exceeds settled topup")
	}
	return status.Errorf(codes.Internal, "failed to apply payment refund: %v", err)
}

func mapChargebackGRPCError(err error) error {
	if errors.Is(err, ErrCustomerNotFound) {
		return status.Error(codes.NotFound, "customer not found")
	}
	if errors.Is(err, ErrPaymentTopupNotFound) {
		return status.Error(codes.NotFound, "payment topup not found")
	}
	if errors.Is(err, ErrChargebackExceedsTopup) {
		return status.Error(codes.FailedPrecondition, "chargeback exceeds settled topup")
	}
	if errors.Is(err, ErrChargebackReversalExceedsWithdrawn) {
		return status.Error(codes.FailedPrecondition, "chargeback reversal exceeds withdrawn amount")
	}
	return status.Errorf(codes.Internal, "failed to apply payment chargeback: %v", err)
}
