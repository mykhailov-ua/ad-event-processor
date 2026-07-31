package controlplane

import (
	"errors"
	"espx/internal/controlplane/pb"
	db "espx/internal/domain/db"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
