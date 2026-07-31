package notifier

import (
	"errors"

	"espx/internal/notifier/pb"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedNotifierServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func mapRPCError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRecipientRequired) ||
		errors.Is(err, ErrBodyRequired) ||
		errors.Is(err, ErrUnsupportedProvider) ||
		errors.Is(err, ErrInvalidNotificationID) ||
		errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrBatchEmpty) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if errors.Is(err, ErrNotificationNotFound) || errors.Is(err, pgx.ErrNoRows) {
		return status.Error(codes.NotFound, ErrNotificationNotFound.Error())
	}
	return status.Error(codes.Internal, "internal server error")
}
