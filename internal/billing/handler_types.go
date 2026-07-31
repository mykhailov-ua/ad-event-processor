package billing

import (
	"espx/internal/billing/pb"
	"espx/internal/config"
)

type Handler struct {
	pb.UnimplementedBillingServiceServer
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}
