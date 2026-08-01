package ledger

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (handler *Handler) requireInternalToken(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing metadata")
	}
	expectedToken := string(handler.cfg.BillingInternalToken)
	if expectedToken == "" {
		return status.Error(codes.FailedPrecondition, "billing internal token not configured")
	}
	tokens := md.Get("x-internal-token")
	if len(tokens) == 0 || subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(expectedToken)) != 1 {
		return status.Error(codes.PermissionDenied, "invalid internal token")
	}
	return nil
}
