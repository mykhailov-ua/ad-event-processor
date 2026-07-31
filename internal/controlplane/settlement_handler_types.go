package controlplane

import (
	"espx/internal/config"
	"espx/internal/controlplane/pb"
)

type SettlementHandler struct {
	pb.UnimplementedSettlementServiceServer
	service *Service
	cfg     *config.Config
}

func NewSettlementHandler(service *Service, cfg *config.Config) *SettlementHandler {
	return &SettlementHandler{
		service: service,
		cfg:     cfg,
	}
}
