package billing

import (
	"context"
	"crypto/subtle"
	"errors"

	"espx/internal/billing/pb"
	"espx/internal/config"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedBillingServiceServer
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

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

func mapRPCError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrInvalidCustomerID),
		errors.Is(err, ErrInvalidInvoiceID),
		errors.Is(err, ErrInvalidBillingMonth):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrCustomerNotFound),
		errors.Is(err, ErrInvoiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrLedgerDrift):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, ErrNoSpend):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, pgx.ErrNoRows):
		return status.Error(codes.NotFound, "not found")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
