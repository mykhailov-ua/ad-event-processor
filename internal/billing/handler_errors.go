package billing

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
