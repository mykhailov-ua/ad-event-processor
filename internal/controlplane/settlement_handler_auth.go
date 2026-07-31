package controlplane

import (
	"context"
	"crypto/subtle"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

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
