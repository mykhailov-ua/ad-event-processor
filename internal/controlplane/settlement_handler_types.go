package controlplane

import (
	"espx/internal/config"
)

type SettlementHandler struct {
	service *Service
	cfg     *config.Config
}

func NewSettlementHandler(service *Service, cfg *config.Config) *SettlementHandler {
	return &SettlementHandler{
		service: service,
		cfg:     cfg,
	}
}
