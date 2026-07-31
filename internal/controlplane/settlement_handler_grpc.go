package controlplane

import (
	"context"
	"errors"
	"espx/internal/controlplane/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
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
