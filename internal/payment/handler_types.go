package payment

import (
	"espx/internal/config"
	"espx/internal/payment/pb"
)

type Handler struct {
	pb.UnimplementedPaymentServiceServer
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}
