package payment

import (
	"context"

	"espx/internal/controlplane/pb"
	"espx/internal/domain"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type grpcSettlementClient struct {
	client pb.SettlementServiceClient
	token  string
}

func newGRPCSettlementClient(client pb.SettlementServiceClient, token string) domain.PaymentSettlement {
	return &grpcSettlementClient{client: client, token: token}
}

func (c *grpcSettlementClient) outgoing(ctx context.Context) context.Context {
	if c.token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-internal-token", c.token)
}

func (c *grpcSettlementClient) ApplyPaymentCredit(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRef string) (bool, int64, error) {
	resp, err := c.client.ApplyPaymentCredit(c.outgoing(ctx), &pb.ApplyPaymentCreditRequest{
		CustomerId:           customerID.String(),
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: ledgerIdempotencyKey,
		PaymentIntentId:      paymentIntentID.String(),
		Provider:             provider,
		ProviderRef:          providerRef,
	})
	if err != nil {
		return false, 0, mapGRPCSettlementError(err)
	}
	return resp.GetApplied(), resp.GetLedgerEntryId(), nil
}

func (c *grpcSettlementClient) ApplyPaymentRefund(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerRefundID string) (bool, int64, error) {
	resp, err := c.client.ApplyPaymentRefund(c.outgoing(ctx), &pb.ApplyPaymentRefundRequest{
		CustomerId:           customerID.String(),
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: ledgerIdempotencyKey,
		PaymentIntentId:      paymentIntentID.String(),
		Provider:             provider,
		ProviderRefundId:     providerRefundID,
	})
	if err != nil {
		return false, 0, mapGRPCSettlementError(err)
	}
	return resp.GetApplied(), resp.GetLedgerEntryId(), nil
}

func (c *grpcSettlementClient) ApplyPaymentChargeback(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	resp, err := c.client.ApplyPaymentChargeback(c.outgoing(ctx), &pb.ApplyPaymentChargebackRequest{
		CustomerId:           customerID.String(),
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: ledgerIdempotencyKey,
		PaymentIntentId:      paymentIntentID.String(),
		Provider:             provider,
		ProviderDisputeId:    providerDisputeID,
	})
	if err != nil {
		return false, 0, mapGRPCSettlementError(err)
	}
	return resp.GetApplied(), resp.GetLedgerEntryId(), nil
}

func (c *grpcSettlementClient) ApplyPaymentChargebackReversal(ctx context.Context, customerID uuid.UUID, amountMicro int64, ledgerIdempotencyKey string, paymentIntentID uuid.UUID, provider string, providerDisputeID string) (bool, int64, error) {
	resp, err := c.client.ApplyPaymentChargebackReversal(c.outgoing(ctx), &pb.ApplyPaymentChargebackReversalRequest{
		CustomerId:           customerID.String(),
		AmountMicro:          amountMicro,
		LedgerIdempotencyKey: ledgerIdempotencyKey,
		PaymentIntentId:      paymentIntentID.String(),
		Provider:             provider,
		ProviderDisputeId:    providerDisputeID,
	})
	if err != nil {
		return false, 0, mapGRPCSettlementError(err)
	}
	return resp.GetApplied(), resp.GetLedgerEntryId(), nil
}

func (c *grpcSettlementClient) GetLedgerEntry(ctx context.Context, paymentIntentID uuid.UUID) (domain.PaymentLedgerEntry, error) {
	resp, err := c.client.GetLedgerEntry(c.outgoing(ctx), &pb.GetLedgerEntryRequest{
		PaymentIntentId: paymentIntentID.String(),
	})
	if err != nil {
		return domain.PaymentLedgerEntry{}, mapGRPCSettlementError(err)
	}
	out := domain.PaymentLedgerEntry{
		Found:                        resp.GetFound(),
		RefundTotalMicro:             resp.GetRefundTotalMicro(),
		ChargebackTotalMicro:         resp.GetChargebackTotalMicro(),
		ChargebackReversalTotalMicro: resp.GetChargebackReversalTotalMicro(),
	}
	if resp.GetFound() && resp.GetTopup() != nil {
		out.HasTopup = true
		out.TopupAmountMicro = resp.GetTopup().GetAmountMicro()
	}
	return out, nil
}

func mapGRPCSettlementError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		msg := st.Message()
		if msg == "payment topup not found" {
			return domain.ErrSettlementTopupNotFound
		}
		return domain.ErrSettlementCustomerNotFound
	}
	return err
}
